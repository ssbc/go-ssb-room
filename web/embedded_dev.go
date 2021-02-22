// SPDX-License-Identifier: MIT

// +build dev

package web

import (
	"net/http"
	"path/filepath"

	"go.mindeco.de/goutils"
)

const Production = false

// absolute path of where this package is located
var pkgDir = goutils.MustLocatePackage("github.com/ssb-ngi-pointer/go-ssb-room/web")

var Templates = http.Dir(filepath.Join(pkgDir, "templates"))

var Assets = http.Dir(filepath.Join(pkgDir, "assets"))
