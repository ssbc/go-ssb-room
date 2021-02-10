// SPDX-License-Identifier: MIT

// +build dev

/*
This is the development version of the templates, where they are read directly from the local filesystem.

to use this pass '-tags dev' to your go build or test commands.
*/

package i18n

import (
	"net/http"
	"path/filepath"

	"go.mindeco.de/goutils"
)

// absolute path of where this package is located
var pkgDir = goutils.MustLocatePackage("github.com/ssb-ngi-pointer/go-ssb-room/web/i18n")

var Defaults http.FileSystem = http.Dir(filepath.Join(pkgDir, "defaults"))
