package web

import "net/http"

// TODO: put this behind +build dev and embedd the assets for deployment

// FIXME: this expects running the server from cmd/server
var Assets = http.Dir("../../web/templates")
