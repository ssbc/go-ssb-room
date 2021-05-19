# Go-SSB Room

This repository contains code for a [Secure Scuttlebutt](https://ssb.nz) [Room (v1+v2) server](https://github.com/ssb-ngi-pointer/rooms2), written in Go.

It includes:
* secret-handshake+boxstream network transport, sometimes referred to as SHS, using [secretstream](https://github.com/cryptoscope/secretstream)
* muxrpc handlers for tunneling connections
* a fully embedded HTTP server & HTML frontend, for administering the room

![](./docs/screenshot.png)

## Features

* [x] Rooms v1 (`tunnel.connect`, `tunnel.endpoints`, etc.)
* [x] User management (allow- & denylisting + moderator & administrator roles), all administered via the web dashboard
* [x] Multiple [privacy modes](https://ssb-ngi-pointer.github.io/rooms2/#privacy-modes)
* [x] [Sign-in with SSB](https://ssb-ngi-pointer.github.io/ssb-http-auth-spec/)
* [x] Alias management

## Getting started
For an architecture and instructions on setting up a webserver to use with `go-ssb-room`, [read the documentation](./docs).

## Development

(_TODO: move this section to contributing document_ See: https://github.com/ssb-ngi-pointer/go-ssb-room/issues/206 )

To get started, you need a recent version of [Go](https://golang.org). v1.16 and onward should be sufficient.

Also, if you want to develop the CSS and HTML on the website, you need Node.js v14 in order to compile Tailwind.

To build the server and see a list of its options:

```bash
cd cmd/server
go build
./server -h
 
Usage of ./server:
  -aliases-as-subdomains
    	needs to be disabled if a wildcard certificate for the room is not available. (default true)
  -dbg string
    	listen addr for metrics and pprof HTTP server (default "localhost:6078")
  -https-domain string
    	which domain to use for TLS and AllowedHosts checks
  -lishttp string
    	address to listen on for HTTP requests (default ":3000")
  -lismux string
    	address to listen on for secret-handshake+muxrpc (default ":8008")
  -logs string
    	where to write debug output to (default is just stderr)
  -mode value
    	the privacy mode (values: open, community, restricted) determining room access controls
  -nounixsock
    	disable the UNIX socket RPC interface
  -repo string
    	where to put the log and indexes (default "~/.ssb-go-room")
  -shscap string
    	secret-handshake app-key (or capability) (default "1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
  -version
    	print version number and build date

```

If you want to view the development server in your browser:
```sh
# change to the root of the project (e.g. cd go-ssb-room) and generate the frontend's styling; requires npm
go generate -tags dev ./...
# now let's build & run the development server
cd cmd/server && go build -tags dev && ./server
# and visit http://localhost:3000
```

This can be useful if you are working on:
* the sqlite migrations, 
* html templates, 
* styling elements using [tailwind](https://tailwindcss.com/docs/)
  * _if you don't run generate with `-tags dev`, the bundled css will only contain the tailwind classes found in *.tmpl at the time of generation!_
* or website templates or assets like JavaScript files, the favicon or other images that are used inside the HTML.

This way, the build won't use the assets embedded in the binary, but instead read them directly from the local filesystem.

Once you are done with your changes and want to update the embedded assets to the production versions:
```sh
# cd to the root of the folder, and then run go generate
go generate ./...
# now build the server without the development mode
cd cmd/server && go build && ./server -htts-domain my.room.example
```


## Tooling
### Mocks

[`counterfeiter`](https://github.com/maxbrunsfeld/counterfeiter) enables generating mocks for defined interfaces. To update the mocks, run `go generate` in package roomdb.

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
# optional step: run a script to generate a valid ssb id @<pubkey>.ed25519, useful for trying things out quickly
./generate-fake-id.sh   
./insert-user -login <username> -key <@pubkey.ed25519>
```
Then repeat your password twice and you are all set for development.

Run `insert-user` without any flags to see all the options.

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
* [staltz](https://github.com/staltz)
* [cblgh](https://github.com/cblgh)

## License

MIT
