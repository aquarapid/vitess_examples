# Using Vault to store vttablet credentials for authentication to MySQL

A new feature of Vitess 9.0 is the ability to store and retrieve
vttablet credentials, that is the usernames and passwords used by
vttablet to connect to backend MySQL instances, in a HashiCorp Vault server.
Note that this is distinct from the usernames and passwords that
you will use to connect via MySQL protocol to Vitess (vtgate). We cover
that in a separate [document](vault_for_vtgate_credentials.md).
This document is a brief walkthough on how to use this feature,
including sample steps of the setup required on the Vault server
side.

If you have followed the accompanying guide to using Vault to
store vtgate credentials, and you already have your Vault server
up and running, you can skip the first part of the Vault setup 
part of this document. The only part that differs is the storing
of the vttablet to MySQL credentials (`dbcreds_secret.json`) in
Vault.

Note that the Vault configurations in this document are not
necessarily recommended for production use. You should ensure that
your Vault administrator or architect validate and adjust the
necessary configurations; and ensure that they align with your security and
compliance requirements and operational procedures.

You will probably want to use some type of bootstrapping process
to obtain the necessary `role_id` and `secret_id` values you
need for configuring vttablet at vttablet startup time.  Manually
bootstrapping your vttablet instances, as this document demonstrates, is not an operationally scalable strategy.

Note also that if you use Vault to retrieve your vttablet to MySQL server
credential information as detailed in this document, your Vault server
becomes an integral part of your Vitess serving infrastructure. In particular,
if your Vault server is down at the time you (re)start one or more of your
vttablet instances, you will end up with running vttablet instances that do
not have the necessary username and password information to log into the
underlying MySQL server instances.  As a result, part of (or all of) your
Vitess serving infrastructure might go down. You may want to have backup
procedures (e.g. to switch to file-based vttablet auth) in place to deal
with an event like this.  It would also be important to have automated
monitoring of the vttablet log files in place to detect and alert events like
these.

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
   You should leave the Vault server running in one terminal and open a new
   terminal window to use for the rest of this walkthrough.

 * Unseal the Vault server.  This is required to store anything in the Vault
   server, or to retrieve tokens/secrets from it.  You will need to provide the 
   value for `Unseal Key 1` that you obtained above:
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
 * The Vault server is now unsealed and you can start configuring it further.

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
   defines the Vault paths and capabilities allowed by this policy. It has
   been scoped to be the minimum needed for use by Vitess.  Now, let's create
   this policy:
  ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault policy write dbcreds dbcreds_policy.hcl 
Success! Uploaded policy: dbcreds
  ```
 * This policy is written to allow storing both vtgate credentials, as well
   as MySQL credentials used by vttablet in the Vault server.  We have 
   covered the vtgate MySQL credentials in this [document](vault_for_vtgate_credentials.md).

 * Next, we are going to upload the actual vttablet to MySQL server credentials
   to the Vault server.  This credential file has the same format as the
   `-db-credentials-file` format vttablet credentials.
   We have provided an example file, called `dbcreds_secret.json`, that just
   looks like this:
 ```sh
{
  "vt_app": [
    "password"
  ],
  "vt_dba": [
    "password"
  ],
  "vt_repl": [
    "password"
  ],
  "vt_appdebug": [
    "password"
  ],
  "vt_filtered": [
    "password"
  ]
}
 ```
 * Basically, it defines a list of passwords for users for the various
   connection pools used by vttablet.  The usernames like `vt_app`, `vt_dba`,
   etc. are the default ones used by vttablet, but you might use different ones
   in your environment (via flags like `-db_app_user`, `-db_dba_user`, etc.).
   In that case, you will need to adjust this credentials JSON appropriately.
   Note that the passwords need to specified in plaintext in this file. You
   will need to rely on the security and encryption of the Vault server to
   protect them.

 * Now, let's go ahead and store the vttablet to MySQL server credentials in the
   `dbcreds_secret.json` file in the Vault server.  As per usual,
   you should be using your `Initial Root Token`:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault kv put kv/prod/dbcreds @dbcreds_secret.json
Key              Value
---              -----
created_time     2021-01-28T09:25:49.428175913Z
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
   appropriately.  If you have a lot of vttablet instances, you may
   need to make this (much) higher, assuming you will be using the
   same approle for every vttablet instance.

 * Now, we need to retrieve and save the `role_id` of the approle we just
   created.  We will need it when configuring vttablet to use Vault:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault read auth/approle/role/vitess/role-id | grep ^role_id
role_id    1ad616e9-7498-b7c3-742a-6ee962489629
 ```
 * Save the `role_id` value.

 * Next, obtain a `secret_id` for the approle. We will also need it
to configure Vitess later:
 ```sh
$ VAULT_CACERT=./vault-cert.pem VAULT_TOKEN=s.CXte7Z3lOSH601asfcHKr2ra ./vault write auth/approle/role/vitess/secret-id k=v | grep ^secret_id | head -1
secret_id             3111006f-5891-f326-d0f8-d29237829b6a
 ```
 * Again, save the `secret_id` value. Since vttablet **has** to read this from
   a file, we will save it to a file now:
 ```sh
$ echo 3111006f-5891-f326-d0f8-d29237829b6a > secret_id
 ```


## Configuring vttablet

Now we have obtained all the necessary information we need about our
Vault configuration to complete the commandline parameters for vttablet
and start vttablet. Let's review the necessary (static) parameters, and
then the parameters that will use the non-static information we have
collected. Note the values for these parameters we prescribe here
align to this walkthrough example.

### vttablet static parameters:

You should add the following parameters to your vttablet startup; with the
example values to align to the Vault server we setup earlier:

 * Use Vault to retrieve vttablet to MySQL credentials:
   **`-db-credentials-server vault`**
 * Set a timeout for speaking to the vault server:
   **`-db-credentials-vault-timeout 10s`**
 * Set a kv secret path to where you stored the vttablet credentials (see earlier
   in this document):  **`-db-credentials-vault-path kv/prod/dbcreds`**
 * Point to the Vault server.  This also could be a load balancer pointing to
   an HA Vault installation:  **`-db-credentials-vault-addr https://127.0.0.1:8200`**
 * Point to the PEM file containing the CA certificate we are going to
   validate the TLS server certificate the Vault server presents against:
   **`-db-credentials-vault-tls-ca $PWD/ca.pem`**
 * How often to renew the Vault tokens we obtain and refresh the MySQL
   credentials used by vttablet from the Vault server.  This value should be
   significantly lower than the `token_ttl` value configured for the approle
   above.  If it is not, timeouts and delays might result in a situation where
   you cannot renew your Vault token.  Note that unlike the Vault integration
   for vtgate, in this case new vttablet logins to MySQL will start failing.
   Monitoring the logs for messages of events such as this and alerting
   accordingly is **strongly** advisable in a production environment:
   **`-db-credentials-vault-ttl 5m`**

### vttablet dynamic parameters:

Additionally, you will need to pass the following parameters to vttablet. Since
these parameters could change over time, you will probably need a bootstrap
script to populate or update them when you start vttablet:

 * The `role_id` we obtained before from the approle we created:
   **`-db-credentials-vault-roleid 1ad616e9-7498-b7c3-742a-6ee962489629`**
 * Provide a file to read the `secret_id` we obtained earlier and saved
   to a file also called `secret_id`:
   **`-db-credentials-vault-role-secretidfile $PWD/secret_id`**

### Log messages on vttablet startup

Upon starting vttablet with the appropriate options, as above, and running
the instance for a few minutes, you should see auth-related messages such
as the following:

```sh
$ grep credentials vttablet.INFO
I0131 18:29:03.128263  240018 credentials.go:241] Vault client status: Token ready
I0131 18:34:03.456725  240018 credentials.go:241] Vault client status: token renewed
```

**Annotated:**

 * The first line reflects that vttablet has:
 
   * successfully connected to Vault
   * successfully authenticated to Vault using the `role_id` and `secret_id`
   * successfully obtained a token
   Implicitly, vttablet also fetched the JSON value for the MySQL user/passwords
   from Vault, or an error would be seen.
   
 * The last line reflects that, 5 minutes after obtaining the intial token,
 as configured via `-db-credentials-vault-ttl`, vttablet renewed the initial
 token with the Vault server.  Again, implicitly the JSON value for the
 MySQL user/passwords was also refreshed from the Vault server at the same
 time.

### Changing/adding users and passwords

The JSON value in Vault that contains the MySQL users and passwords can be
changed at any time, and would get picked up by the vttablet servers the next
time they refresh from Vault, as determined by the `-db-credentials-vault-ttl`
value.

If you want to have the values updated more promptly, you can restart vttablet,
or more gracefully, you can send a SIGHUP signal to the vttablet process (e.g. 
via `kill -HUP`).  The SIGHUP process will (among other things like reopening
log files), cause the vttablet server to refresh its credentials from Vault.

Note that because of the way connection pools work in vttablet, it might be
an extended period of time after the credentials are refreshed until they
start to actually be used, since the current connections in the pools are not
actively torn down and refreshed (e.g. logged into MySQL again). Using default
vttablet options, for example, this might take up to 30 minutes, since that is
the default value for `-queryserver-config-idle-timeout`.  As a result, if
need to do a "hard" rotation of your app passwords, you will probably want to
restart your vttablet instances.  However, because of the disruption that this
would cause, a common strategy for password rotation in this case is to create
an alternate set of usernames and passwords on the MySQL side, roll that out
everywhere, then start to recycle your vttablet instances to use the new
users and password over a long period of time.  Finally, when you are confident
that the old MySQL users are no longer in use, you can drop or lock them
on the MySQL side.

## What happens if Vault goes down?

If Vault is down at the time vttablet is started, it will not be able to fetch
any credentials.  As a result, vttablet will start, but not have any valid
credentials available to log into MySQL. This means that no requests to this
vttablet instance that results in a MySQL query would succeed.
This is obviously undesireable, and you should take to avoid this scenario.
We suggest:

  * Monitoring the vttablet logs for errors related to this.  The error will be
    something like:  `Error in vault client initialization, will retry:` and
    then some detail about the type of error.  If an error occurs, vttablet
    will start retrying against Vault every second until it successfully
    obtains credentials.
  * Note that if you provide the wrong usernames or passwords in the JSON
    blob in Vault, and vttablet fetches this blob, it will continue retrying
    using the incorrect password to log into MySQL. Even if you fix the
    blob in Vault, you will have to either:
    * wait for the vttablet credentials TTL to expire
    * or restart vttablet
    * or send a SIGHUP signal to vttablet to reload the credentials
  * Make sure your Vault infrastructure is configured for HA, potentially
    across multiple datacenters or availability zones, if necessary.  If this is
    not possible, you may want to reconsider if you really want to use Vault to
    store your vttablet to MySQL credentials.
  * If you **do** have problems with your Vault servers, **DO NOT** restart your
    vttablet instances.  If vttablet is already running, and has obtained
    credentials from the Vault server, it will retain those credentials, at
    least until the next TTL expiration.  If this TTL is sufficiently long,
    you this **should** (although, depending in the timing, this is not
    guaranteed) have some time to repair your Vault serving infrastructure
    until it causes vttablet to be unable to login to MySQL.


