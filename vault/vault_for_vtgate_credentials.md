# Using Vault to store vtgate credentials for authentication

A new feature of Vitess 9.0 is the ability to store and retrieve
vtgate credentials, that is the usernames and passwords used by
MySQL protocol clients of vtgate, in a HashiCorp Vault server.
This document is a brief walkthough on how to use this feature,
including sample steps of the setup required on the Vault server
side.

Note that the Vault configurations in this document are not
necessarily recommended for production use. You should ensure that
your Vault administrator or architect validate and adjust the
necessary configurations, and ensure that they align with your security and
compliance requirements and operational procedures.

You will probably want to use some type of bootstrapping process
to obtain the necessary `role_id` and `secret_id` values you
need for configuring vtgate at vtgate startup time.  Manually
bootstrapping your vtgate instances, as this document walks you
through, is not an operationally scalable strategy.

Note also that if you use Vault to retrieve your vtgate
credential information as detailed in this document, your Vault server
becomes an integral part of you Vitess serving infrastructure. In particular,
if you Vault server is down at the time you (re)start one or more of your
vtgate instances, you will end up with running vtgate instances that do
not have authentication information.  As a result, your vtgates will be
available, but your application will not be able to log into them.  You
may want to have backup procedures (e.g. to switch to file-based vtgate
auth) in place to deal with an event like this.  It would also be important
to have automated monitoring of the vtgate and/or application log files
in place to detect and alert events like these.

## Setup a Vault server

For the purposes of this walkthrough, we will be using a
standalone Vault server.  You could also use the Vault server docker
container for the same purpose.  Note that we will be using pre-generated
certificates to allow the Vault server to serve requests over TLS.
In your environment, you should obviously be using certificates that you
generated yourself, or obtained otherwise in accordance with your
certificate issuing policies and procedures.

We will be assuming that this walkthrough is run on a Linux machine.

 * Obtain the Vault binary:
 ```sh
$ wget https://vitess-operator.storage.googleapis.com/install/vault
--2021-01-28 00:48:21--  https://vitess-operator.storage.googleapis.com/install/vault
Resolving vitess-operator.storage.googleapis.com (vitess-operator.storage.googleapis.com)... 172.217.6.48, 2607:f8b0:4005:809::2010
Connecting to vitess-operator.storage.googleapis.com (vitess-operator.storage.googleapis.com)|172.217.6.48|:443... connected.
HTTP request sent, awaiting response... 200 OK
Length: unspecified [application/octet-stream]
Saving to: ‘vault’

vault                             [                                         <=>     ] 126.59M  13.3MB/s    in 10s     

2021-01-28 00:48:32 (12.4 MB/s) - ‘vault’ saved [132738840]

$ chmod a+x ./vault
 ```
 
 * Complete the Vault server configfile, and start the server:
 ```sh
$ cat vault.hcl.template | sed -e "s#PWD#$PWD#g" > vault.hcl
$ ./vault server -config=vault.hcl
==> Vault server configuration:

                     Cgo: disabled
              Go Version: go1.15.4
              Listener 1: tcp (addr: "127.0.0.1:8200", cluster address: "127.0.0.1:8201", max_request_duration: "1m30s", max_request_size: "33554432", tls: "enabled")
               Log Level: info
                   Mlock: supported: true, enabled: false
           Recovery Mode: false
                 Storage: inmem
                 Version: Vault v1.6.1
             Version Sha: 6d2db3f033e02e70202bef9ec896360062b88b03

==> Vault server started! Log data will stream in below:

2021-01-28T00:59:41.659-0800 [INFO]  proxy environment: http_proxy= https_proxy= no_proxy=
2021-01-28T00:59:41.660-0800 [WARN]  no `api_addr` value specified in config or in VAULT_API_ADDR; falling back to detection if possible, but this value should be manually set
 ```

 * Run the Vault server setup process.  This initializes the server with a
   single master key share, something you would not do in production:
 ```sh
$ VAULT_CACERT=./vault-cert.pem ./vault operator init -key-shares=1 -key-threshold=1
Unseal Key 1: l/NdRa2WDSF/wLt9upQdPTow1W/cbKqVf2bBF+hkOQk=

Initial Root Token: s.CXte7Z3lOSH601asfcHKr2ra

Vault initialized with 1 key shares and a key threshold of 1. Please securely
distribute the key shares printed above. When the Vault is re-sealed,
restarted, or stopped, you must supply at least 1 of these keys to unseal it
before it can start servicing requests.

Vault does not store the generated master key. Without at least 1 key to
reconstruct the master key, Vault will remain permanently sealed!

It is possible to generate new unseal keys, provided you have a quorum of
existing unseal keys shares. See "vault operator rekey" for more information.
 ```
 * Be sure to save the `Unseal Key 1` and `Initial Root Token` values. Yours
   will be different from this example.  You will need them later.
   You should leave the Vault server running in one terminal, and open a new
   terminal window to use for the rest of this walkthrough.

 * Unseal the Vault server.  This is required to store anything in the Vault
   server, or retrieve tokens/secrets from it.  You will need to provide the 
   value for `Unseal Key 1` as you obtained above:
 ```sh
$ VAULT_CACERT=./vault-cert.pem ./vault operator unseal l/NdRa2WDSF/wLt9upQdPTow1W/cbKqVf2bBF+hkOQk=
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.6.1
Storage Type    inmem
Cluster Name    vault-cluster-67366070
Cluster ID      c174754f-d1e4-e88f-9eb5-272edad31d3f
HA Enabled      false
```
 * The Vault server is now unsealed, and you can start configuring it further.

 * Enable the `kv` secrets engine.  Note that you should use your value of
   `Initial Root Token` obtained above as the value for `VAULT_TOKEN`:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault secrets enable -version=2 kv
Success! Enabled the kv secrets engine at: kv/ 
 ```

 * Enable the `approle` engine.  Again, you should use your value of
   `Initial Root Token`:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault auth enable approle
Success! Enabled approle auth method at: approle/
 ```

 * Create a custom policy called `dbcreds` to allow access to the credentials
   we are going to create soon.  Look through the `dbcreds_policy.hcl` file. It
   defines the the Vault paths and capabilities allowed by this policy. It has
   been scoped to be the minimum needed for use by Vitess.  Now, let's create
   this policy:
  ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault policy write dbcreds dbcreds_policy.hcl 
Success! Uploaded policy: dbcreds
  ```
 * This policy is written to allow storing both vtgate credentials as well
   as MySQL credentials used by vttablet in the Vault server.  We will be
   covering vttablet MySQL credentials in a separate document.

 * Next, we are going to upload the actual vtgate credentials to the Vault
   server.  This credential file has the same format as the `file` format 
   vtgate credentials, as documented here:
   https://vitess.io/docs/user-guides/configuration-advanced/user-management/#authentication
   We have provided an example file, called `vtgatecreds_secret.json`, that just
   looks like this:
 ```sh
{
  "vtgate_user": [
    {
      "Password": "password123"
    }
  ]
}
 ```
 * Basically, it defines a single user (`vtgate_user`) and sets the password
   to `password123`.  Note that you can still use MySQL password hashes
   for your passwords here, in accordance with the documentation linked above,
   this is just an example.

 * Now, let's go ahead and store the vtgate credentials in the
   `vtgatecreds_secret.json` file in the Vault server.  As per usual,
   you should be using your `Initial Root Token`:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault kv put kv/prod/vtgatecreds @vtgatecreds_secret.json
Key              Value
---              -----
created_time     2021-01-28T09:23:06.161462207Z
deletion_time    n/a
destroyed        false
version          1
 ```

 * Now, we want to configure an approle that binds to the access policy we
   uploaded earlier:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault write auth/approle/role/vitess secret_id_ttl=30m token_num_uses=0 token_ttl=10m token_max_ttl=0 secret_id_num_uses=4 policies=dbcreds
Success! Data written to: auth/approle/role/vitess
 ```
 * Note that in your case, you should configure the approle parameters
   (especially the TTLs) appropriately for your environment.  In this
   case we use a very low TTL for test purposes.  In a production
   environment a low TTL probably does not make sense, since you will
   be hitting your Vault server much more often than necessary to
   refresh tokens. You should also adjust the `secret_id_num_uses`
   appropriately.  If you have a lot of vtgate instances, you may
   need to make this (much) higher, assuming you will be using the
   same approle for every vtgate instance.

 * Now, we need to retrieve and save  the `role_id` of the approle we just
   created.  We will need it when configuring vtgate to use Vault:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault read auth/approle/role/vitess/role-id | grep ^role_id
role_id    1ad616e9-7498-b7c3-742a-6ee962489629
 ```
 * Save the `role_id` value.

 * Next, obtain a `secret_id` for the approle, we will also need it
to configure Vitess later:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault write auth/approle/role/vitess/secret-id k=v | grep ^secret_id | head -1
secret_id             3111006f-5891-f326-d0f8-d29237829b6a
 ```
 * Again, save the `secret_id` value. Since vtgate **has** to read this from
   a file, we will save it to a file now:
 ```sh
$ echo 3111006f-5891-f326-d0f8-d29237829b6a > secret_id
 ```


## Configuring vtgate

Now we have obtained all the necessary information we need about our
Vault configuration to complete the commandline parameters for vtgate
and start vtgate. Let's review the necessary (static) parameters, and
then the parameters that will use the non-static information we have
collected. Note the values for these parameters we prescribe here
align to this walkthrough example.

### vtgate static parameters:

You should add the following parameters to your vtgate startup; with the
example values to align to the Vault server we setup earlier:
 * Use Vault to retrieve vtgate auth:  **`-mysql_auth_server_impl vault`**
 * Set a timeout for speaking to the vault server:  **`-mysql_auth_vault_timeout 10s`**
 * Set a kv secret path where you stored the vtgate credentials (see earlier
   in this document):  **`-mysql_auth_vault_path kv/prod/vtgatecreds`**
 * Point to the Vault server.  This also could be a load balancer pointing to
   an HA Vault installation:  **`-mysql_auth_vault_addr https://127.0.0.1:8200`**
 * Point to the PEM file containing the CA certificate we are going to
   validate the TLS server certificate the Vault server presents against:
   **`-mysql_auth_vault_tls_ca $PWD/ca.pem`**
 * How often to renew the Vault tokens we obtain and refresh the authentication
   information for vtgate from the Vault server.  This value should be
   significantly lower than the `token_ttl` value configured for the approle
   above.  If it is not, timeouts and delays might result in a situation where
   you cannot renew your Vault token.  In a case like this, vtgate will continue
   using the credentials it obtained previously indefinitely;  but that might
   not be the up-to-date credentials.  Again, monitoring the logs for messages
   of events such as this and alerting accordingly is advisable in a
   production environment:  **`-mysql_auth_vault_ttl 5m`**

### vtgate dynamic parameters:

Additionally, you will need to pass the following parameters to vtgate. Since
these parameters could change over time, you will probably need a bootstrap
script to populate or update them when you start vtgate:
 * The `role_id` we obtained before from the approle we created:
   **`-mysql_auth_vault_roleid 1ad616e9-7498-b7c3-742a-6ee962489629`**
 * Provide a file to read the `secret_id` we obtained earlier and saved
   to a file also called `secret_id`:
   **`-mysql_auth_vault_role_secretidfile $PWD/secret_id`**

### Log messages on vtgate startup

Upon starting vtgate with the appropriate options, as above, and running
the instance for a few minutes, you should see auth-related messages such
as the following:

```sh
$ grep auth vtgate.INFO
I0130 15:17:45.750539  191369 auth_server_clientcert.go:36] Not configuring AuthServerClientCert because mysql_server_ssl_ca is empty
I0130 15:17:45.750657  191369 auth_server_ldap.go:58] Not configuring AuthServerLdap because mysql_ldap_auth_config_file and mysql_ldap_auth_config_string are empty
I0130 15:17:45.750673  191369 auth_server_static.go:91] Not configuring AuthServerStatic, as mysql_auth_server_static_file and mysql_auth_server_static_string are empty
I0130 15:17:45.783696  191369 auth_server_vault.go:190] reloadVault(): success. Client status: Token ready
I0130 15:22:45.793098  191369 auth_server_vault.go:190] reloadVault(): success. Client status: token renewed
```

**Annotated:**
 * The first three lines reflect that the TLS client cert, LDAP and static
 file auth methods for vtgate are not configured.
 * The fourth line reflects that vtgate has:
   * successfully connected to Vault
   * successfully authenticated to Vault using the `role_id` and `secret_id`
   * successfully obtained a token
   Implicitly, vtgate also fetched the JSON value for the vtgate user/passwords
   from Vault, or an error would be seen.
 * The last line reflects that, 5 minutes after obtaining the intial token,
 as configured via `-mysql_auth_vault_ttl`, vtgate renewed the initial token
 with the Vault server.  Again, implicitly the JSON value for the vtgate
 user/passwords was also refreshed from the Vault server at the same time.

### Changing/adding users and passwords

The JSON value in Vault that contains the users and passwords can be changed
at any time, and would get picked up by the vtgate servers the next time they
refresh from Vault, as determined by the `-mysql_auth_vault_ttl` value.

If you want to have the values updated more promptly, you can restart vtgate,
or more gracefully, you can send a SIGHUP signal to the vtgate process (e.g. 
via `kill -HUP`).  The SIGHUP process will (among other things like reopening
log files), cause the vtgate server to refresh its credentials from Vault.

## What happens if Vault goes down?

If Vault is down at the time vtgate is started, it will not be able to fetch
any credentials.  As a result, vtgate will start, but not have any users
and passwords configured.  This means that no MySQL clients will be able to
authenticate to that vtgate instance.  This is undesireable, and you should
take some steps to avoid this scenario.  We suggest:

  * Monitoring the vtgate for errors related to this.  The error will be
    something like:  `Error in vault client initialization, will retry:` and
    then some detail about the type of error.  If an error occurs, vtgate
    will start retrying against Vault every 10 seconds until it successfully
    obtains credentials.
  * Having a login check running against your vtgate servers that periodically 
    authenticates using a MySQL user/password. This is good operational
    practice, even if you are not using Vault.
  * Make sure your Vault infrastructure is configured for HA, potentially
    across multiple datacenters or availability zones, if necessary.  If this is
    not possible, you may want to reconsider if you really want to use Vault to
    store your vtgate credentials.
  * If you **do** have problems with your Vault servers, **DO NOT** restart your
    vtgate instances.  If vtgate is already running, and has obtained
    credentials from the Vault server, it will retain those credentials, even
    if it cannot refresh them after the configured TTL.  This means that your
    clients will still be able to authenticate, even though it might not be
    with the most up-to-date credentials.  This design choice was made in the
    Vault implementation for vtgate to optimize for availability over
    authentication credential consistency.

## Additional operational guardrails

Note that similar to if Vault is down, if you update the Vault key that
contains the vtgate credentials to an empty value or invalid JSON, vtgate
will refuse to overwrite any credentials it previously obtained from Vault.
You could see messages in the vtgate log like:

```
Error parsing vtgate Vault auth server config
```
or
```
vtgate credentials from Vault empty! Not updating previously cached values
```

You should monitor for these in your logs as well.

