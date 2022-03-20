# Password rotation with Vitess

**NOTE: The following assumes you are connecting to `vtgate` via the
MySQL protocol, not gRPC.**

In Vitess, there are two main sets of username/password credentials
to worry about when it comes to password rotation:

  * `vtgate` credentials:  i.e. the credentials used by application clients
  when connecting to `vtgate`;  usually configured via parameters to the
  MySQL client connector for your application language.
  * `vttablet` credentials: i.e. the credentials used by `vttablet` to
  connect to the underlying MySQL instances (whether they are local MySQL
  server instances or external instances like RDS or CloudSQL).  This
  often uses the standard Vitess/`vttablet` MySQL users like `vt_dba`,
  `vt_app`, etc.


## `vtgate` password rotation

Password rotation in a large environment without causing downtime can be
tricky.  This is because it is hard to coordinate changing the password
used across a large number of application endpoints used to access the
database at the same time the database password changes.  Fortunately,
Vitess has some specific features to help with that.

Specifically, `vtgate` supports:

  * Allowing more than one password for a given user.  This makes it 
  possible to have two different application clients use the same user
  with different passwords to access their data via `vtgate`.
  * Reloading the password store for `vtgate` containing usernames and
  passwords without having to restart `vtgate` (i.e. no downtime)

The combination of these two features makes password rotation for
application access to `vtgate` relatively trivial then:

  * Suppose we have an application that uses a username `A` to access
  Vitess with password `X`.
  * We update all the static auth files for the `vtgate` instances
  in our install to add the new (rotated) password of `Y` to user `A`.
  * We have all the `vtgate` instances reload the auth file
  (either by sending the appropriate signal to the `vtgate` process,
  or waiting until the periodic auth file reload has occurred, if it was
  configured).
  * At this point, both password `X` and password `Y` are valid passwords
  for user `A`.  Existing clients using user `A` thus will continue
  working.
  * Now we start updating the application(s) using user `A` to use password
  `Y` instead of password `X`.  This can be done incrementally, since both
  passwords are valid for user `A`.
  * Once we are confident that all applications have been updated, we can
  update the static auth files for the `vtgate` instances to remove
  password `X` as a valid password for user `A`.  We will also need to
  arrange for the reloading of the auth files, as before.
  * At this point, the password rotation for user `A` is complete.

We can, of course, rotate the passwords for multiple users at the same time
in the same fashion.


## `vttablet` password rotation

Password rotation for `vttablet`, i.e. rotation of the passwords used by
`vttablet` to connect to the underlying MySQL server(s), has to follow a more
conventional route than `vtgate` password rotation, since we are limited
by the authentication features of MySQL Server.  The following strategy,
based on the common "dual users" pattern, has been used by Vitess users with
success without having to use a "stop-the-world" strategy.  Note that whatever
strategy you follow, a `vttablet` restart is mandatory at some point during
the process, since dynamic updating of the various MySQL users and passwords
used by `vttablet` is not possible.


 * To start with, `vttablet` uses a set of MySQL users, say `A`, `B` and `C`
 with respective passwords `X`, `Y` and `Z`.
 * We now want to change the passwords for all these users. To achieve this,
 we create **new** users with the same permissions/grants as `A`, `B` and `C`.
 These users are `AA`, `BB` and `CC`.  We assign our new/rotated passwords
 to these users, say `XX`, `YY` and `ZZ`.
 * We validate manually or otherwise that the new users are:
   * Active
   * Has the new passwords we want
   * Have the correct permissions
 * Now, we update the `vttablet` configurations to use the new usernames
 (`AA`, `BB` and `CC` instead of `A`, `B` and `C`) and passwords
 (`XX`, `YY` and `ZZ` instead of `X`, `Y` and `Z`).
 * During a maintainance window, we restart the `vttablet`s one-by-one,
 validating that they are using the new usernames, and are still functioning
 as before.  Depending on your situation, you may also perform planned
 reparents as you restart primary shards, to minimize write unavailability.
 * After the restarts are complete, Vitess is now using the new set of
 users and passwords, and your password rotation is complete. You can
 now validate no clients are connecting to MySQL server using the old
 usernames anymore, and then delete the old users from MySQL.
