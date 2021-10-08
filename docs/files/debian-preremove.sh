# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

systemctl stop go-ssb-room
systemctl disable go-ssb-room
# TODO: we might want to have a proper config file so users dont need to tweak this file, then we can also remove and upgrade it properly
# rm /etc/systemd/system/go-ssb-room.service
# systemctl daemon-reload
systemctl reset-failed
