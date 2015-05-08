# mediocre-api/prefab/rest

This is a "prefabricated" REST api which uses the other mediocre-api packages as
a backend. It's actually broken up into multiple parts, each intended to run
completely separately from each other.

* shield - The endpoint which browsers and other clients are intended to hit. It
  implements rate-limiting as well as user authentication. Assuming a client
  passes those checks it will forward the request on to the appropriate backing
  REST api. For example, all requests to `/user/*` will get forwarded to the
  user api process.

* user - Manages everything related to user accounts, e.g. creation,
  modification, disabling, password changing, etc...
