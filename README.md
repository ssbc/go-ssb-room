# Go-SSB Rooms

This repository contains code for a [Secure-Scuttlebutt Room v2](github.com/ssb-ngi-pointer/rooms2/) server writen in Go.

It not only includes the secret-handshake+boxstream setup and muxrpc handlers for tunneling connections but also a fully embedded http/html interface for administering the room.

## Features

* [x] Rooms v1 (`tunnel.connect`, `tunnel.endpoints`, etc.)
* [ ] Sign-in with SSB
* [x] Simple whitelisting
    Currently via `.ssb-go-rooms/authorized_keys`.
    To be replaced with a authorization list on the dashboard.
* [ ] Alias managment

## Development

The basics just need a recent version of [Go](https://golang.org). v1.14 and onward should be sufficient.

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

If you are working on the html templates or assets for them, build the server with `go build -tags dev`.
This way it won't use the assets that are embedded in the binary but read them directly from the local filesystem.

Once you are done with your changes run `go generate` in package web to update them.

## Testing

### Rooms

The folder `tests/nodejs` contains tests against the JavaScript implementation. To run them, install node and npm and run the following:

```bash
cd tests/nodejs
npm ci
go test
```

### Web Dashboard

The folders `web/handlers` contain the HTTP handlers for the dashboard. Each subfolder comes with unit tests for the specific area (like `auth`, `news`, etc.). Simply run `go test` in one of them or run `go test ./web/...` in the root of the repo to test them all.

## Authors

* [cryptix](https://github.com/cryptix) (`@p13zSAiOpguI9nsawkGijsnMfWmFd5rlUNpzekEE+vI=.ed25519`)

## License

MIT