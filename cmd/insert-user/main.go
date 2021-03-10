// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <repo-location> <user-name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "repo-location: default is $HOME/.ssb-go-room\n")
		os.Exit(1)
		return
	}

	r := repo.New(os.Args[1])
	db, err := sqlite.Open(r)
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

	ctx := context.Background()
	uid, err := db.AuthFallback.Create(ctx, os.Args[2], bytePassword)
	check(err)

	fmt.Fprintln(os.Stderr, "created user with ID", uid)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
