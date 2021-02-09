// SPDX-License-Identifier: MIT

// +build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/sqlite"
)

func main() {
	err := vfsgen.Generate(sqlite.Migrations, vfsgen.Options{
		PackageName:  "sqlite",
		BuildTags:    "!dev",
		VariableName: "Migrations",
	})
	if err != nil {
		log.Fatalln(err)
	}

}
