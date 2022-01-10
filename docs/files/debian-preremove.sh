#!/bin/sh

# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

set -e

if [ -d /run/systemd/system ] && [ "$1" = remove ]; then
	deb-systemd-invoke stop ssb-pub.service >/dev/null || true
fi
