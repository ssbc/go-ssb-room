// SPDX-License-Identifier: MIT

// +build dev

package i18n

import (
	"io/fs"
	"os"
	"path/filepath"

	"go.mindeco.de/goutils"
)

var Defaults fs.FS = os.DirFS(defaultsPath)

var (
	pkgDir       = goutils.MustLocatePackage("github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n")
	defaultsPath = filepath.Join(pkgDir, "defaults")
)
