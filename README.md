# Go-SSB Room

This repository contains code for a [Secure-Scuttlebutt Room v2](https://github.com/ssb-ngi-pointer/rooms2/) server written in Go.

It not only includes the secret-handshake+boxstream setup and muxrpc handlers for tunneling connections but also a fully embedded http/html interface for administering the room.

## Features

* [x] Rooms v1 (`tunnel.connect`, `tunnel.endpoints`, etc.)
* [ ] Sign-in with SSB
* [x] Simple allow-listing
    Currently via `.ssb-go-rooms/authorized_keys`.
    To be replaced with a authorization list on the dashboard.
* [ ] Alias managment

## Development

To get started, you need a recent version of [Go](https://golang.org). v1.16 and onward should be sufficient.

To build the server and see a list of it's options, run the following:

```bash
$ cd cmd/server
$ go build
$ ./server -h
Usage of ./server:
  -dbg string
    	listen addr for metrics and pprof HTTP server (default "localhost:6078")
  -lishttp string
    	address to listen on for HTTP requests (default ":3000")
  -lismux string
    	address to listen on for secret-handshake+muxrpc (default ":8008")
  -nounixsock
    	disable the UNIX socket RPC interface
  -logs string
    	where to write debug output to (default is just stderr)
  -repo string
    	where to put the log and indexes (default "/home/cryptix/.ssb-go-room")
  -shscap string
    	secret-handshake app-key (or capability) (default "1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
  -version
    	print version number and build date
```

If you want to view the development server in your browser:
```sh
cd cmd/server && go build -tags dev && ./server
# and visit http://localhost:3000
```

This can be useful if you are working on:
* the sqlite migrations, 
* html templates, 
* or website assets

This way the build won't use the assets embedded in the binary, but instead read them directly from the local filesystem.

Once you are done with your changes run and want to update the embedded assets:
```sh
go generate
```
**Note**: you need to run generate in each changed package.

## Tooling
### Mocks

[`counterfeiter`](https://github.com/maxbrunsfeld/counterfeiter) enables generating mocks for defined interfaces. To update the mocks, run `go generate` in package admindb.
* TODO: setup tool as dependency (no manual install)

### Database schema

This project uses [sql-migrate](https://github.com/rubenv/sql-migrate) to upgrade the sqlite database when necessary.

To upgrade, create a new file in `admindb/sqlite/migrations` with your changes. 

**Note**: similar to the web assets, you need to use `go test -tags dev` to test them. Afterwards run, `go generate` to embed the assets in the code and thus the resulting server binary.

### No ORM

We use [sqlboiler](github.com/volatiletech/sqlboiler) to generate type-safe Go code directly from SQL statements and table definitions. This approach suits the programming language much more then classical ORM approaches, which usually rely havily on reflection for (un)packing structs.

To generate them run the following commands. This will populate `admindb/sqlite/models`:
* (TODO: automate this with `go generate`)

```bash
# also included as generate_models.sh
cd admindb/sqlite
go test
sqlboiler sqlite3 --wipe
```

The generated package `admindb/sqlite/models` is then used to implemente the custom logic of the different services in `admindb/sqlite`.

Aside: I would have used `sqlc` since it's a bit more minimal and uses hand written SQL queries instead of generic query builders but it [currently doesn't support sqlite](https://github.com/kyleconroy/sqlc/issues/161).

### Development user creation

`cmd/insert-user` contains code to create a fallback user. Build it and point it to your database with a username:

```bash
cd cmd/insert-user
go build
./insert-user $HOME/.ssb-go-room/roomdb my-user
```

Then repeat your password twice and you are all set for development.

## Testing
### Rooms

The folder `tests/nodejs` contains tests against the JavaScript implementation. To run them, install node and npm and run the following:

```bash
cd tests/nodejs
npm ci
go test
```

### Web Dashboard

The folder `web/handlers` contains the HTTP handlers for the dashboard. Each subfolder comes with unit tests for the specific area (like `auth`, `news`, etc.). Simply run `go test` in one of them or run `go test ./web/...` in the root of the repo to test them all.

## Authors

* [cryptix](https://github.com/cryptix) (`@p13zSAiOpguI9nsawkGijsnMfWmFd5rlUNpzekEE+vI=.ed25519`)

## License

MIT
