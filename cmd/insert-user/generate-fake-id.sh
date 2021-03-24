#!/bin/bash
id=$(dd if=/dev/urandom bs=1 count=32 2>/dev/null | base64 -w0)
echo "@${id}.ed25519"
