// SPDX-License-Identifier: MIT

// +build ignore

package main

import (
	"log"
	"os/exec"

	"github.com/shurcooL/vfsgen"

	"github.com/ssb-ngi-pointer/go-ssb-room/web"
)

func main() {
	err := vfsgen.Generate(web.Templates, vfsgen.Options{
		PackageName:  "web",
		BuildTags:    "!dev",
		VariableName: "Templates",
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = vfsgen.Generate(web.Assets, vfsgen.Options{
		PackageName:  "web",
		BuildTags:    "!dev",
		VariableName: "Assets",
	})
	if err != nil {
		log.Fatalln(err)
	}

	// nasty hack to strip duplicate type information
	// https://github.com/shurcooL/vfsgen/issues/23
	err = exec.Command("sed", "-i", "/^type vfsgen۰FS/,$d", "assets_vfsdata.go").Run()
	if err != nil {
		log.Fatalln(err)
	}

	// clean up the unused imports
	err = exec.Command("goimports", "-w", ".").Run()
	if err != nil {
		log.Fatalln(err)
	}
}
