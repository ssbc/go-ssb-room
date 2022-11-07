# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

# create a user to run the server as
adduser --system --home /var/lib/go-ssb-room go-ssb-room
chown go-ssb-room /var/lib/go-ssb-room

# welcome message
cat <<EOF
> Welcome !

go-ssb-room has been installed as a systemd service.

It will store it's files (roomdb and cookie secrets) under /var/lib/go-ssb-room.
This is also where you would put custom translations.

For more configuration background see /usr/share/go-ssb-room/README.md
or visit the code repo at https://github.com/ssbc/go-ssb-room/tree/master/docs

Like outlined in that document, we highly encourage using nginx with certbot for TLS termination.
We also supply an example config for this. You can find it under /usr/share/go-ssb-room/nginx-example.conf

> Important

Before you start using room server via the systemd service, you need to at least change the https domain in the systemd service.

Edit /etc/systemd/system/go-ssb-room.service and then run this command to reflect the changes:

sudo systemctl daemon-reload

> Running the room server:

To start/stop go-ssb-room:

sudo systemctl start go-ssb-room
sudo systemctl stop go-ssb-room

To enable/disable go-ssb-room starting automatically on boot:

sudo systemctl enable go-ssb-room
sudo systemctl disable go-ssb-room

To reload go-ssb-room:

sudo systemctl restart go-ssb-room

To view go-ssb-room logs:

sudo journalctl -f -u go-ssb-room

EOF
