<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# Getting Started

There are two paths to starting your own room: creating a build from source, or downloading one
of the premade releases.

## Premade builds

See the [releases page](https://github.com/ssbc/go-ssb-room/releases) for packaged linux releases.

We currently only distributed pre-packaged releases for Debian-compatible distributions.
See [Issue #79](https://github.com/ssbc/go-ssb-room/issues/79) for the details.
If this doesn't work for you, read the "Creating a build" section below.

After running `sudo dpkg -i go-ssb-room_v1.2.3_linux_amd64.deb` pay special attention to the
[postinstall notes](./files/debian-postinstall.sh) for how to configure the systemd file and webserver.

## Creating a build

* [Download Go](https://golang.org/doc/install) & [set up your Go environment](https://golang.org/doc/install#install). You will need at least Go v1.17.
* Download the repository `git clone git@github.com:ssbc/go-ssb-room.git && cd go-ssb-room`
* [Follow the development instructions](./development.md)
* You should now have a working go-ssb-room binary! Read the HTTP Hosting section below and admin
  user sections below, for more instructions on the last mile.

# Docker & Docker-compose

This project includes a docker-compose.yml file as well as a Docker file. Using
it should be fairly straight forward.

Start off by making a copy of `.env_example` called `.env` and insert your
website domain there.

For example:
```env
# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: Unlicense

HTTPS_DOMAIN=yourserver.your-domain.com
ALIASES_AS_SUBDOMAINS=true
# uncomment variable  if you want to store data in a custom directory (required for default docker-compose setup)
REPO=/ssb-go-room-secrets
```
- `HTTPS_DOMAIN`: This is the domain, where the server can be found
- `ALIASES_AS_SUBDOMAINS`: If a user gets an alias on this server, the user will get his own subdomain. For example, the user chooses the alias 'bob', so the subdomain will be 'bob.yourserver.your-domain.com'. The user will be reachable under this domain.
- `REPO`: This is the folder, where the docker container has his volume, to permanently store data.

With that done execute

```bash
docker-compose build room
```

Followed by

```bash
docker-compose up -d
```

That will start the SSB server and let it run in the background.

Your database, secrets and other things will be synchronized to a folder in your
project called "docker-secrets".

After starting your server for the first time you need to enter your running
server to insert your first user (your docker-compose up should be active). First of all, you need to copy your SSB-ID. You can find that, by your account. It's the long text with the @ at the beginning. This account will become the administrator of the server. Then you need to get into the running docker container.

You can do this by:

```bash
docker-compose exec room sh
```

Then inside the docker container:

```bash
/app/cmd/insert-user/insert-user -repo /ssb-go-room-secrets @your-own-ssb-public-key
```

Fill in your password and then exit your instance by typing `exit`. The password will not be shown, that's normal.

You should setup Nginx or HTTPS load-balancing outside the docker-compose
instance. The port `8008` for the SSB protocol should be streamed to the domain from the instance. This can be done with (for example) Nginx. More about that in ![HTTP Hosting](#http-hosting).

# HTTP Hosting

We currently assume a standard HTTPS server in front of go-ssb-room to facilitate TLS
termination and certificate management. This should be possible with most modern HTTP servers
since it's a pretty standard practice, known as [reverse
proxying](https://en.wikipedia.org/wiki/Reverse_proxy).

Two bits of rationale:

1. People usually want to have more than one site on their server. Put differently, we could
have [LetsEncrypt](https://letsencrypt.org/) inside the go-ssb-room server but it would have to
listen on port :443—blocking the use of other domains on the same IP.
2. Listening on :443 can be pretty annoying (you might need root privileges or similar capabilities).

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

For example, to get a wildcard SSL certificate for `hermies.club`, we typically run

```bash
certbot certonly --manual --server https://acme-v02.api.letsencrypt.org/directory \
  --preferred-challenges dns-01 \
  -d 'hermies.club' -d '*.hermies.club'
```

(Replace `hermies.club` with your room's domain, of course)

`certbot` will tell you to update TXT DNS records with the key `_acme-challenge.hermies.club` but be
carefully with your DNS provider because you may have to input just `_acme-challenge` since the rest
is often added automatically by your provider.

When the process is complete with `certbot`, pay attention to where the certificate has been placed
in the filesystem. If it's at `/etc/letsencrypt/live/hermies.club`, it's correct, otherwise you may
need to rename it e.g. `hermies.club-0001` to `hermies.club`.

The example nginx configuration uses prebuilt Diffie-Hellman parameters.  You can generate these
with the following command:

```bash
openssl dhparam -out /etc/letsencrypt/ssl-dhparams.pem 2048
```

Then restart your server, e.g. `systemctl restart nginx`.

If you see such errors as the following when setting up your deployment:

```bash
nginx: [emerg] unknown "connection_upgrade" variable
```

You may need to configure `$connection_upgrade` in your
`/etc/nginx/nginx.conf`. See [this
article](https://futurestud.io/tutorials/nginx-how-to-fix-unknown-connection_upgrade-variable)
for more.

## Enable TCP ports

For your room to fully work the following **TCP** ports need to be allowed:

- 80 (HTTP)
- 443 (HTTPS)
- 8008 (SSB)

### Example

Using a Debian-compatible distribution with `ufw`, execute the commands below:

```bash
sudo ufw allow http
sudo ufw allow https
sudo ufw allow 8008/tcp
```


# First Admin user

To manage your now working server, you need an initial admin user. For this you can use the "insert-user" utility included with go-ssb-room.

If you installed the Debian package, you will first need to install Go to build the "insert-user" utility.  You can do this via:

```bash
sudo apt-get install golang-go
```

(**WARNING**: please check that `golang-go` is >= 1.17 and if not, you may need to use the [official installation documentation](https://go.dev/dl/) instead. `go-ssb-room` requires at least Go 1.17.)

In a new terminal window navigate to the insert-user utility folder and compile the GO-based utility into an executable your computer can use

```bash
cd cmd/insert-user
go build
```

A new executable file should be created called "insert-user"
Execute the `./insert-user -h` command to get a full list of custom options (optional location of the repo & SQLite database and user role). follow the instructions given in the output you receive.

example (with custom repo location, only needed if you setup your with a custom repo):

```bash
./insert-user -repo "/ssb-go-room-secrets" "@Bp5Z5TQKv6E/Y+QZn/3LiDWMPi63EP8MHsXZ4tiIb2w=.ed25519"
```

Or if you installed go-ssb-room using the Debian package:

```bash
sudo ./insert-user -repo "/var/lib/go-ssb-room" "@Bp5Z5TQKv6E/Y+QZn/3LiDWMPi63EP8MHsXZ4tiIb2w=.ed25519"
```

It will ask you to create a password to access the web-front-end.  You can now login in the web-front-end using these credentials.

# Troubleshooting

## Building the docker image

If the build fails with the error message similar to 
```
note: module requires Go 1.18
ERROR: Service 'room' failed to build: The command '/bin/sh -c cd /app/cmd/server && go build &&     cd /app/cmd/insert-user && go build' returned a non-zero code: 2
```
just increase in the `Dockerfile` the version like shown here:

Change:
```Dockerfile
FROM golang:1.17-alpine
```

To Go version that the module requires (like described in the error message):
```Dockerfile
FROM golang:1.18-alpine
```

And run the build command again.