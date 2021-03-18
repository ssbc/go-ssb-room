// SPDX-License-Identifier: MIT

// insert-user is a utility to create a new member and password
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite"
)

func main() {
	u, err := user.Current()
	check(err)

	var (
		repoPath string
		role     roomdb.Role
	)

	flag.StringVar(&repoPath, "repo", filepath.Join(u.HomeDir, ".ssb-go-room"), "where the repo of the room is located")
	flag.Func("role", "which role the new member should have (ie moderator, admin, member. defaults to admin)", func(val string) error {
		if val == "" {
			role = roomdb.RoleAdmin
			return nil
		}

		switch strings.ToLower(val) {
		case "admin":
			role = roomdb.RoleAdmin
		case "moderator":
			role = roomdb.RoleAdmin
		case "member":
			role = roomdb.RoleMember

		default:
			return fmt.Errorf("unknown member role: %q", val)
		}

		return nil
	})

	flag.Parse()

	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <user-name> <@theirPublicKey.ed25519>\n", os.Args[0])
		flag.Usage()
		os.Exit(1)
		return
	}

	r := repo.New(repoPath)
	db, err := sqlite.Open(r)
	check(err)
	defer db.Close()

	pubKey, err := refs.ParseFeedRef(os.Args[1])

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
	mid, err := db.Members.Add(ctx, os.Args[1], *pubKey, role)
	check(err)

	err = db.AuthFallback.Create(ctx, mid, os.Args[1], bytePassword)
	check(err)

	fmt.Fprintln(os.Stderr, "created member with ID", mid)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
