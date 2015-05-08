# mediocre-api/prefab/rest/shield

Shield is the external endpoint which is used to access internal APIs. It
handles rate-limiting requests as well as user authentication. See the
[auth](/auth) README for more on how those two work.

Once a request is successfully processed it is forwarded on to an appropriate
backend api to actually handle the request (in most cases). The api the request
is forwarded to depends on the endpoint the request was trying to hit. E.g. all
requests starting with `/user` will get forwarded to the user api.

## `_asUser`

When shield processes a request and sees a valid user token identifying the user
making the request, it will append the `_asUser` GET parameter to the request
before forwarding it off. The backend apis will expect this parameter and use it
appropriately.

For example, a properly authenticated request to `/user/toaster` by the user
`strudel` will get turned into `/user/toaster?_asUser=strudel` before being
forwarded onto the user api.

## Builtin

Most requests to shield are forwarded onward, but these are handled directly by
it:

-----

```
GET /sheild/token
```

Used to retrieve a new API token. Returns:

```
{
    "Token": "API token"
}
```

This may return `420 chill bro...` if the IP is rate-limited

## Build and Use

To build (from the root of the mediocre-api project)

    go build ./prefab/rest/shield

To use:

    ./shield --secret=somesecret

If requests are being load balanced across multiple instances of shield all of
those instances must have the same secret.

Use `--help` or `-h` to see more available options.
