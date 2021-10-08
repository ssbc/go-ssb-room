// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// +build !dev

package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

// Production can be used to determain different aspects at compile time (like hot template reloading)
const Production = true

var (
	Assets    http.FileSystem
	Templates http.FileSystem
)

// correct the paths by stripping their prefixes
func init() {
	var err error

	prefixedAssets, err := fs.Sub(embedAssets, "assets")
	if err != nil {
		log.Fatal(err)
	}
	Assets = http.FS(prefixedAssets)

	prefixedTemplates, err := fs.Sub(embedTemplates, "templates")
	if err != nil {
		log.Fatal(err)
	}
	Templates = http.FS(prefixedTemplates)
}

//go:embed templates/*
var embedTemplates embed.FS

//go:embed assets/*
var embedAssets embed.FS
