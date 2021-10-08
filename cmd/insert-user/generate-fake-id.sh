#!/bin/bash

# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

id=$(dd if=/dev/urandom bs=1 count=32 2>/dev/null | base64 -w0)
echo "@${id}.ed25519"
