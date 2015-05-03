# mediocre-api/room

[![GoDoc](https://godoc.org/github.com/mediocregopher/mediocre-api/room?status.svg)](https://godoc.org/github.com/mediocregopher/mediocre-api/room)

This package provides basic functionality for managing "rooms" of users

A room is a place where multiple users gather together to interact in some way.
They have the following qualities:

* A single unique string identifies the room

* Rooms are ephemeral, they are not explicitely created nor explicitely
  destroyed

* A user can join a room and leave a room. They cannot be in a room twice at
  the same time. Users must periodically "check in" to a room to confirm they
  are still in it

* A user is identified by an arbitrary string

* Any user can retrieve a list of users currently in a room


