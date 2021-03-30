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
		login    string
		pubKey   *refs.FeedRef
		role     roomdb.Role = roomdb.RoleAdmin
		repoPath string
	)

	flag.StringVar(&login, "login", "", "username (used when logging into the room's web ui)")
	flag.Func("key", "the public key of the user, format: @<base64-encoded public-key>.ed25519", func(val string) error {
		if len(val) == 0 {
			return fmt.Errorf("the public key is required. if you are just testing things out, generate one by running 'cmd/insert-user/generate-fake-id.sh'\n")
		}
		key, err := refs.ParseFeedRef(val)
		if err != nil {
			return fmt.Errorf("%s\n", err)
		}
		pubKey = key
		return nil
	})
	flag.StringVar(&repoPath, "repo", filepath.Join(u.HomeDir, ".ssb-go-room"), "[optional] where the locally stored files of the room is located")
	flag.Func("role", "[optional] which role the new member should have (values: mod[erator], admin, or member. default is admin)", func(val string) error {
		switch strings.ToLower(val) {
		case "admin":
			role = roomdb.RoleAdmin
		case "mod":
			fallthrough
		case "moderator":
			role = roomdb.RoleModerator
		case "member":
			role = roomdb.RoleMember
		default:
			return fmt.Errorf("unknown member role: %q", val)
		}

		return nil
	})
	flag.Parse()

	/* we require at least 5 arguments: <executable> + -name <val> + -key <val> */
	/*                                  1              2     3       4    5     */
	if len(os.Args) < 5 {
		cliMissingArguments("please provide the default arguments -name and -key")
	}

	if login == "" {
		cliMissingArguments("please provide a username with -login <username>")
	}

	if pubKey == nil {
		cliMissingArguments("please provide a public key with -key")
	}

	r := repo.New(repoPath)
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
		fmt.Fprintln(os.Stderr, "Passwords didn't match")
		os.Exit(1)
		return
	}

	ctx := context.Background()
	mid, err := db.Members.Add(ctx, *pubKey, role)
	check(err)

	err = db.AuthFallback.Create(ctx, mid, login, bytePassword)
	check(err)

	fmt.Fprintf(os.Stderr, "Created member %s (%s) with ID %d\n", login, role, mid)
}

func cliMissingArguments(message string) {
	executable := strings.TrimPrefix(os.Args[0], "./")
	fmt.Fprintf(os.Stderr, "%s: %s\nusage:%s -name <user-name> -key <@<base64-encoded public key>.ed25519> <optional flags>\n", executable, message, executable)
	flag.Usage()
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
