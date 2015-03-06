# mediocre-api

**THIS PROJECT IS STILL A WORK IN PROGRESS, AND THE API IS LIKELY TO CHANGE A
LOT FOR A WHILE**

This is a collection of packages, each independent of the other, but which
together can be used to form the skeleton of an api. It is presumed to be backed
by redis, and allows for redis cluster.

## Parts

There are multiple parts to this package, each of which can be used more or less
independently (although most assume you are using the [auth](/auth) package for
user authentication). Each part has its own README explaining how to use it

- [auth](/auth) - Wrapper around `net/http` providing rate limiting and user
  auth on arbitrary requests (in leu of sessions)

- [user](/user) - User creation/modification/authentication. Also provides a
  basic REST api which can be used and built on
