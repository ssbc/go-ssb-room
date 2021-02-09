// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"fmt"
	"os"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb/sqlite"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <file.db> <user-name>\n", os.Args[0])
		os.Exit(1)
		return
	}

	db, err := sqlite.Open(os.Args[1])
	check(err)
	defer db.Close()

	fmt.Fprintln(os.Stderr, "Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	check(err)

	fmt.Fprintln(os.Stderr, "Repeat Password: ")
	bytePasswordRepeat, err := terminal.ReadPassword(int(syscall.Stdin))
	check(err)

	if !bytes.Equal(bytePassword, bytePasswordRepeat) {
		fmt.Fprintln(os.Stderr, "passwords didn't match")
		os.Exit(1)
		return
	}

	err = db.AuthFallback.Create(os.Args[2], bytePassword)
	check(err)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
