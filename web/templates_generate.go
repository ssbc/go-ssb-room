// +build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/ssb-ngi-pointer/gossb-rooms/web"
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
}
