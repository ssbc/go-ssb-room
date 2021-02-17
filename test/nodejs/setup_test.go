// SPDX-License-Identifier: MIT

// Package nodejs_test contains test scenarios and helpers to run interoparability tests against the javascript implementation.
package nodejs_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"

	"golang.org/x/sync/errgroup"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"go.cryptoscope.co/netwrap"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/testutils"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
)

func init() {
	err := os.RemoveAll("testrun")
	if err != nil {
		fmt.Println("failed to clean testrun dir")
		panic(err)
	}
}

type testSession struct {
	t *testing.T

	info log.Logger

	repo string

	keySHS []byte

	done errgroup.Group

	ctx    context.Context
	cancel context.CancelFunc
}

// TODO: restrucuture so that we can test both (default and random net keys) with the same Code

// rolls random values for secret-handshake app-key and HMAC
func newRandomSession(t *testing.T) *testSession {
	appKey := make([]byte, 32)
	rand.Read(appKey)
	return newSession(t, appKey)
}

// if appKey is nil, the default value is used
// if hmac is nil, the object string is signed instead
func newSession(t *testing.T, appKey []byte) *testSession {
	repo := filepath.Join("testrun", t.Name())
	err := os.RemoveAll(repo)
	if err != nil {
		t.Errorf("remove testrun folder (%s) for this test: %s", repo, err)
	}

	ts := &testSession{
		info:   testutils.NewRelativeTimeLogger(nil),
		repo:   repo,
		t:      t,
		keySHS: appKey,
	}

	// todo: hook into deadline
	ts.ctx, ts.cancel = context.WithCancel(context.Background())

	return ts
}

func (ts *testSession) startGoServer(al admindb.AllowListService, opts ...roomsrv.Option) *roomsrv.Server {
	r := require.New(ts.t)

	// prepend defaults
	opts = append([]roomsrv.Option{
		roomsrv.WithLogger(ts.info),
		roomsrv.WithListenAddr("localhost:0"),
		roomsrv.WithRepoPath(ts.repo),
		roomsrv.WithContext(ts.ctx),
	}, opts...)

	if ts.keySHS != nil {
		opts = append(opts, roomsrv.WithAppKey(ts.keySHS))
	}

	opts = append(opts,
		roomsrv.WithPostSecureConnWrapper(func(conn net.Conn) (net.Conn, error) {
			return debug.WrapDump(filepath.Join("testrun", ts.t.Name(), "muxdump"), conn)
		}),
	)

	srv, err := roomsrv.New(al, opts...)
	r.NoError(err, "failed to init tees a server")
	ts.t.Logf("go server: %s", srv.Whoami().Ref())
	ts.t.Cleanup(func() {
		ts.t.Log("bot close:", srv.Close())
	})

	ts.done.Go(func() error {
		err := srv.Network.Serve(ts.ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			err = fmt.Errorf("go server exited: %w", err)
			ts.t.Log(err)
			return err
		}
		return nil
	})

	return srv
}

var jsBotCnt = 0

// returns the jsbots pubkey
func (ts *testSession) startJSClient(name, testScript string, peerAddr net.Addr, peerRef refs.FeedRef) refs.FeedRef {
	ts.t.Log("starting client", name)
	r := require.New(ts.t)
	cmd := exec.CommandContext(ts.ctx, "node", "../../../sbot_client.js")
	cmd.Stderr = os.Stderr
	outrc, err := cmd.StdoutPipe()
	r.NoError(err)

	if name == "" {
		name = fmt.Sprintf("jsbot-%d", jsBotCnt)
	}
	jsBotCnt++

	// copy test scripts (maybe later with templates if we need to)
	cmd.Dir = filepath.Join("testrun", ts.t.Name(), name)
	os.MkdirAll(cmd.Dir, 0700)
	err = exec.Command("cp", "-r", "testscripts", cmd.Dir).Run()
	r.NoError(err)

	env := []string{
		"TEST_NAME=" + name,
		"TEST_REPO=" + cmd.Dir,
		"TEST_PEERADDR=" + netwrap.GetAddr(peerAddr, "tcp").String(),
		"TEST_PEERREF=" + peerRef.Ref(),
		"TEST_SESSIONSCRIPT=" + testScript,
		// "DEBUG=ssb:room:tunnel:*",
	}

	if ts.keySHS != nil {
		env = append(env, "TEST_APPKEY="+base64.StdEncoding.EncodeToString(ts.keySHS))
	}

	cmd.Env = env
	r.NoError(cmd.Start(), "failed to init test js-sbot")

	ts.done.Go(func() error {
		err := cmd.Wait()
		if err != nil {
			ts.t.Logf("node server %s: exited with %s", name, err)
		}
		return nil
	})
	ts.t.Cleanup(func() {
		cmd.Process.Kill()
	})

	pubScanner := bufio.NewScanner(outrc) // TODO muxrpc comms?
	r.True(pubScanner.Scan(), "multiple lines of output from js - expected #1 to be %s pubkey/id", name)

	jsBotRef, err := refs.ParseFeedRef(pubScanner.Text())
	r.NoError(err, "failed to get %s key from JS process")
	ts.t.Logf("JS %s:%d %s", name, jsBotCnt, jsBotRef.Ref())
	return *jsBotRef
}

func (ts *testSession) startJSBotAsServer(name, testScriptFileName string) (*refs.FeedRef, int) {
	r := require.New(ts.t)
	cmd := exec.CommandContext(ts.ctx, "node", "../../../sbot_serv.js")
	cmd.Stderr = os.Stderr
	outrc, err := cmd.StdoutPipe()
	r.NoError(err)

	if name == "" {
		name = fmt.Sprintf("jsbot-%d", jsBotCnt)
	}
	jsBotCnt++

	// copy test scripts (maybe later with templates if we need to)
	cmd.Dir = filepath.Join("testrun", ts.t.Name(), name)
	os.MkdirAll(cmd.Dir, 0700)
	err = exec.Command("cp", "-r", "testscripts", cmd.Dir).Run()
	r.NoError(err)

	var port = 1024 + mrand.Intn(23000)

	env := []string{
		"TEST_NAME=jsbot-" + name,
		"TEST_REPO=" + cmd.Dir,
		fmt.Sprintf("TEST_PORT=%d", port),
		"TEST_SESSIONSCRIPT=" + testScriptFileName,
		// "DEBUG=ssb:room:tunnel:*",
	}
	if ts.keySHS != nil {
		env = append(env, "TEST_APPKEY="+base64.StdEncoding.EncodeToString(ts.keySHS))
	}
	cmd.Env = env

	r.NoError(cmd.Start(), "failed to init test js-sbot")

	ts.done.Go(func() error {
		err := cmd.Wait()
		if err != nil {
			ts.t.Logf("node server %s: exited with %s", name, err)
		}
		return nil
	})
	ts.t.Cleanup(func() {
		cmd.Process.Kill()
	})

	pubScanner := bufio.NewScanner(outrc) // TODO muxrpc comms?
	r.True(pubScanner.Scan(), "multiple lines of output from js - expected #1 to be %s pubkey/id", name)

	srvRef, err := refs.ParseFeedRef(pubScanner.Text())
	r.NoError(err, "failed to get srvRef key from JS process")
	ts.t.Logf("JS %s: %s port: %d", name, srvRef.Ref(), port)
	return srvRef, port
}

func (ts *testSession) wait() {
	ts.cancel()

	assert.NoError(ts.t, ts.done.Wait())
}
