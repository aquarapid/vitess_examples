# Negative ACLs

In Vitess' ACL config authorization mechanism, described
[here](https://vitess.io/docs/user-guides/configuration-advanced/authorization/)
it is not generally possible to achieve the following elegantly:

 * Assume a database with the tables `t1`, `t2` and `t3`, and the database
 (`vtgate`) users `regular` and `privileged`.
 * Give read and write access to only tables `t1` and `t2` to user `regular`.
 * **Only** give user `privileged` access to read or write table `t3`.

In a simple case like the above, we can just construct an ACL config
with two ACLs, and enumerate all the necessary table names in each ACL
(`t1` and `t2` in the first ACL;  `t3` in the second ACL), and achieve the
goal. Let's call this type of configuration "completely specified".
However, everytime a non-privileged table is added to the schema, the
ACL config needs to be updated to add the table name to the config, or
user `regular` will not have access to it.  For schemas with large numbers
of tables, and that change frequently, this can be a burden.

In general, it is not possible to express a "negative" target ACL in Vitess'
ACL config syntax, e.g.:  `Give this user access to all tables except these
specific ones`. It is, however, possible to express an ACL config that is
equivalent to the above "completely specified" ACL config, but somewhat
easier to manage, even for large numbers of tables.

Consider the following more realistic example:
  * Your schema has a 100+ tables.
  * You regularly add new tables.
  * You have a special set of tables called `secret` and `supersecret`
    that you only want a specific `vtgate` user called `super` to have
    access to.
  * You have 3 other users:
    * `readonly` for read access to all tables, except `secret` and
      `supersecret`
    * `readwrite` for read and write access to all tables, except `secret`
       and `supersecret`
    * `dba` for read, write and admin access to all tables, except
      `secret` and `supersecret`.
  * You only a few other tables that start with the letter `s`, called
    `s1`, `s2`, `s3`.
  * We assume you do not use table names with upper case or other
    characters.

The idea of our configuration is that we construct access to the
non-sensitive data using wildcards of table names for each letter
of the alphabet.  We then only need to specify table names fully for
the letter of the alphabet that our "special" tables start
with.  This still requires us to specify a list of table names, but
only for the letters of the alphabet that the "special" tables start
with. 

Here is the ACL config that satisfies our requirements:

```json
{
  "table_groups": [
    {
      "name": "acl1",
      "table_names_or_prefixes": ["a%", "b%", "c%", "d%", "e%", "f%", "g%", "h%", "i%", "j%", "k%", "l%", "m%", "n%", "o%", "p%", "q%", "r%", "t%", "u%", "v%", "w%", "x%", "y%", "z%", "s1", "s2", "s3"],
      "readers": ["readonly", "readwrite", "dba"],
      "writers": ["readwrite", "dba"],
      "admins": ["dba"]
    },
    {
      "name": "acl2",
      "table_names_or_prefixes": ["secret", "supersecret"],
      "readers": ["super"],
      "writers": ["super"],
      "admins": ["super"]
    }
  ]
}
```

Now, with the above ACL config, you only need to update the ACL config
if you add a new table that starts with the letter `s`.

