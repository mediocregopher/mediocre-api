# mediocre-api/prefab/rest/user

An internal endpoint for interacting with users, e.g. creating, disabling,
resetting passwords, setting arbitrary properties, etc...

user can be backed by either a single redis instance or a redis cluster.
Multiple user processes can run against a single instance or cluster safely.

## Endpoints

Errors are returned as strings in the body (not json-encoded), with a non-200
response code. All bodies of 200 responses, if there is a body at all, will be
json encoded.

The `_asUser` GET argument can be used to indicate the call is being made on
behalf of an authenticated user. This may be required for some calls (e.g. POST
to `/<username>`), or augment other calls (e.g. GET to `/<username>`).

-----

```
POST /new-user

{
    "Username": "Username to be created",
    "Email": "Email of user",
    "Password": "Password user should be created with"
}
```

Creates a new user with the given username/email/password. If you do not wish to
use a username and only want emails you may use the email in the `Username`
field.

May return `400 user exists` if the username is taken

-----

```
GET /<username>
```

Returns

```
{
    "Name":"The username of the user",
    "Create":"Time string of when the user was created"
}
```

or, if authed as the user being requested, this:

```
{
    "Name":"The username of the user",
    "Create":"Time string of when the user was created",
    "Email":"The user's primary email",
    "Modified":"Time string of the last time any field was modified",
    "Disabled": false // Whether or not the account is disabled
}
```

This may return `404 user not found`

-----

```
POST /<username>

{
    "Editable field":"New value"
}
```

Used to modify one or more fields which are `Editable`. Must be authd as the
user in order to call. A 200 with no body is returned if successful.

On failure this may return:

* `404 user not found`
* `400 could not authenticate user`
* `400 unknown field <field>`
* `400 field <field> not editable`

-----

```
POST /<username>/password

{
    "OldPassword":"Old password",
    "NewPassword":"New password"
}
```

Used to modify the user's active password. Must be authd as the user in order to
call. A 200 with no body is returned if successful.

On failure this may return:

* `404 user not found`
* `400 could not authenticate user`
* `400 user account is disabled`

-----

```
POST /<username>/auth

{
    "Password":"User password"
}
```

Used to confirm that the given password is the correct one for the user. A 200
with no body is returned if the password is correct.

On failure this may return:

* `404 user not found`
* `400 user account is disabled`
* `400 could not authenticate user`


## Build and Use

To build (from the root of the mediocre-api project)

    go build ./prefab/rest/user

To use:

    ./user

Use `--help` or `-h` to see more available options.
