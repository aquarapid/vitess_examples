# Password rotation with Vitess

In Vitess, there are two main sets of username/password credentials
to worry about when it comes to password rotation:

  * `vtgate` credentials:  i.e. the credentials used by application clients
  when connecting to `vtgate`
  * `vttablet` credentials: i.e. the credentials used by `vttablet` to
  connect to the underlying MySQL instances (whether they are local MySQL
  server instances or external instances like RDS or CloudSQL).  This
  often uses the standard Vitess/vttablet MySQL users like `vt_dba`, 
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

