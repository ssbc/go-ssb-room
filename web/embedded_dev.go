// SPDX-License-Identifier: MIT

// +build dev

/*
This is the development version of the templates, where they are read directly from the local filesystem.

to use this pass '-tags dev' to your go build or test commands.
*/

package web

import (
	"embed"

	"go.mindeco.de/goutils"
)

const Production = false

// absolute path of where this package is located
var pkgDir = goutils.MustLocatePackage("github.com/ssb-ngi-pointer/go-ssb-room/web")

//go:embed templates/*
var Templates embed.FS

// var Templates http.FileSystem = http.Dir(filepath.Join(pkgDir, "templates"))

//go:embed assets/*
var Assets embed.FS

// var Assets http.FileSystem = http.Dir(filepath.Join(pkgDir, "assets"))
