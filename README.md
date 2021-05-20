# Go-SSB Room
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fssb-ngi-pointer%2Fgo-ssb-room.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fssb-ngi-pointer%2Fgo-ssb-room?ref=badge_shield)


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

## Installation

See the [releases page](https://github.com/ssb-ngi-pointer/go-ssb-room/releases) for packaged linux releases.

We currently only distributed pre-packaged releases for debian-compatible distributions. See [Issue #79](https://github.com/ssb-ngi-pointer/go-ssb-room/issues/79) for the details. If this doesn't work for you, we ask you to read the Development notes and build from source.

After running `sudo dpkg -i go-ssb-room_v1.2.3_Linux_x86_64.deb` pay special attention to the [postinstall notes](./docs/debian/postinstall.sh) for how to configure the systemd file and webserver.

## Development

For an in-depth walkthrough, see the [development.md](./docs/development.md) in the `docs` folder of this repository.

## Authors

* [cryptix](https://github.com/cryptix) (`@p13zSAiOpguI9nsawkGijsnMfWmFd5rlUNpzekEE+vI=.ed25519`)
* [staltz](https://github.com/staltz)
* [cblgh](https://github.com/cblgh)

## License

MIT


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fssb-ngi-pointer%2Fgo-ssb-room.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fssb-ngi-pointer%2Fgo-ssb-room?ref=badge_large)