// SPDX-License-Identifier: MIT

// +build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
)

func main() {
	err := vfsgen.Generate(i18n.Defaults, vfsgen.Options{
		PackageName:  "i18n",
		BuildTags:    "!dev",
		VariableName: "Defaults",
	})
	if err != nil {
		log.Fatalln(err)
	}

}
