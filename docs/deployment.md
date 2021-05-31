# Getting Started
There are two paths to starting your own room: creating a build from source, or downloading one
of the premade releases.

## Premade builds

See the [releases page](https://github.com/ssb-ngi-pointer/go-ssb-room/releases) for packaged linux releases.

We currently only distributed pre-packaged releases for Debian-compatible distributions.
See [Issue #79](https://github.com/ssb-ngi-pointer/go-ssb-room/issues/79) for the details.
If this doesn't work for you, read the "Creating a build" section below.

After running `sudo dpkg -i go-ssb-room_v1.2.3_Linux_x86_64.deb` pay special attention to the
[postinstall notes](./files/debian-postinstall.sh) for how to configure the systemd file and webserver.

## Creating a build

* [Download Go](https://golang.org/doc/install) & [set up your Go environment](https://golang.org/doc/install#install). You will need at least Go v1.16.
* Download the repository `git clone git@github.com:ssb-ngi-pointer/go-ssb-room.git && cd go-ssb-room`
* [Follow the development instructions](./development.md)
* You should now have a working go-ssb-room binary! Read the HTTP Hosting section below and admin
  user sections below, for more instructions on the last mile.

# HTTP Hosting

We currently assume a standard HTTPS server in front of go-ssb-room to facilitate TLS
termination and certificate management. This should be possible with most modern HTTP servers
since it's a pretty standard practice, known as [reverse
proxying](https://en.wikipedia.org/wiki/Reverse_proxy).

Two bits of rationale:

1) People usually want to have more than one site on their server. Put differently, we could
have [LetsEncrypt](https://letsencrypt.org/) inside the go-ssb-room server but it would have to
listen on port :443—blocking the use of other domains on the same IP.  2) Listening on :443 can
be pretty annoying (you might need root privileges or similar capabilities).

go-ssb-room needs three headers to function properly, which need to be forwarded by the
webserver.

* `X-Forwarded-Host` as which domain name the room is running (enforce strict TLS checking)
* `X-Forwarded-Proto` to ensure that TLS is used (and redirect if necessary)
* `X-Forwarded-For` the remote TCP/IP address of the client accessing the room (used for rate
  limiting)

[example-nginx.conf](./files/example-nginx.conf) contains an [nginx](https://nginx.org) config that
we use for [hermies.club](https://hermies.club). To get a wildcard TLS certificate you can
follow the steps in [this
article](https://medium.com/@alitou/getting-a-wildcard-ssl-certificate-using-certbot-and-deploy-on-nginx-15b8ffa34157),
which uses the [certbot](https://certbot.eff.org/) utility.

# First Admin user

To manage your now working server, you need an initial admin user. For this you can use the "insert-user" utility included with go-ssb-room.
In a new terminal window navigate to the insert-user utility folder and compile the GO-based utility into an executable your computer can use

```
cd cmd/insert-user
go build
```

A new executable file should be created called "insert-user"
Execute the `./insert-user -h` command to get a full list of custom options (optional location of the repo & SQLite database and user role). follow the instructions given in the output you receive.

example (with custom repo location, only needed if you setup your with a custom repo):

```
./insert-user -repo "/ssb-go-room-secrets" "@Bp5Z5TQKv6E/Y+QZn/3LiDWMPi63EP8MHsXZ4tiIb2w=.ed25519"
```

You can now login in the web-front-end using these credentials