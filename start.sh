#!/bin/sh

[[ -f ".env" ]] && source .env
./cmd/server/server -https-domain "${HTTPS_DOMAIN}" -repo "${REPO:-~/.ssb-go-room-secrets}" -aliases-as-subdomains "${ALIASES_AS_SUBDOMAINS}"
