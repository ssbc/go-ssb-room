// SPDX-License-Identifier: MIT

// Package nodejs_test contains test scenarios and helpers to run interoparability tests against the javascript implementation.
package nodejs_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"go.cryptoscope.co/netwrap"
	refs "go.mindeco.de/ssb-refs"

	"go.mindeco.de/ssb-rooms/internal/maybemod/testutils"
	"go.mindeco.de/ssb-rooms/roomsrv"
)

func init() {
	err := os.RemoveAll("testrun")
	if err != nil {
		fmt.Println("failed to clean testrun dir")
		panic(err)
	}
}

func writeFile(t *testing.T, data string) string {
	r := require.New(t)
	f, err := ioutil.TempFile("testrun/"+t.Name(), "*.js")
	r.NoError(err)
	_, err = fmt.Fprintf(f, "%s", data)
	r.NoError(err)
	err = f.Close()
	r.NoError(err)
	return f.Name()
}

type testSession struct {
	t *testing.T

	info log.Logger

	repo string

	keySHS, keyHMAC []byte

	// since we can't pass *testing.T to other goroutines, we use this to collect errors from background taskts
	backgroundErrs []<-chan error

	gobot *roomsrv.Server

	done errgroup.Group
	// doneJS, doneGo <-chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

// TODO: restrucuture so that we can test both (default and random net keys) with the same Code

// rolls random values for secret-handshake app-key and HMAC
func newRandomSession(t *testing.T) *testSession {
	appKey := make([]byte, 32)
	rand.Read(appKey)
	hmacKey := make([]byte, 32)
	rand.Read(hmacKey)
	return newSession(t, appKey, hmacKey)
}

// if appKey is nil, the default value is used
// if hmac is nil, the object string is signed instead
func newSession(t *testing.T, appKey, hmacKey []byte) *testSession {
	repo := filepath.Join("testrun", t.Name())
	err := os.RemoveAll(repo)
	if err != nil {
		t.Errorf("remove testrun folder (%s) for this test: %s", repo, err)
	}

	ts := &testSession{
		info:    testutils.NewRelativeTimeLogger(nil),
		repo:    repo,
		t:       t,
		keySHS:  appKey,
		keyHMAC: hmacKey,
	}

	// todo: hook into deadline
	ts.ctx, ts.cancel = context.WithCancel(context.Background())

	return ts
}

func (ts *testSession) startGoServer(opts ...roomsrv.Option) {
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

	srv, err := roomsrv.New(opts...)
	r.NoError(err, "failed to init tees a server")
	ts.t.Logf("go server: %s", srv.Whoami())
	ts.t.Cleanup(func() {
		srv.Close()
	})

	ts.done.Go(func() error {
		err := srv.Network.Serve(ts.ctx)
		if err != nil {
			err = fmt.Errorf("node serve exited: %w", err)
			ts.t.Log(err)
			return err
		}
		return nil
	})

	ts.gobot = srv

	// TODO: make muxrpc client and connect to whoami for _ready_ ?
	return
}

var jsBotCnt = 0

func (ts *testSession) startJSBot(jsbefore, jsafter string) refs.FeedRef {
	return ts.startJSBotWithName("", jsbefore, jsafter)
}

// returns the jsbots pubkey
func (ts *testSession) startJSBotWithName(name, jsbefore, jsafter string) refs.FeedRef {
	ts.t.Log("starting client", name)
	r := require.New(ts.t)
	cmd := exec.CommandContext(ts.ctx, "node", "./sbot_client.js")
	cmd.Stderr = os.Stderr

	outrc, err := cmd.StdoutPipe()
	r.NoError(err)

	if name == "" {
		name = fmt.Sprint(ts.t.Name(), jsBotCnt)
	}
	jsBotCnt++
	env := []string{
		"TEST_NAME=" + name,
		"TEST_BOB=" + ts.gobot.Whoami().Ref(),
		"TEST_GOADDR=" + netwrap.GetAddr(ts.gobot.Network.GetListenAddr(), "tcp").String(),
		"TEST_BEFORE=" + writeFile(ts.t, jsbefore),
		"TEST_AFTER=" + writeFile(ts.t, jsafter),
	}

	if ts.keySHS != nil {
		env = append(env, "TEST_APPKEY="+base64.StdEncoding.EncodeToString(ts.keySHS))
	}
	if ts.keyHMAC != nil {
		env = append(env, "TEST_HMACKEY="+base64.StdEncoding.EncodeToString(ts.keyHMAC))
	}
	cmd.Env = env
	r.NoError(cmd.Start(), "failed to init test js-sbot")

	ts.done.Go(cmd.Wait)
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
	ts.t.Log("starting srv", name)
	r := require.New(ts.t)
	cmd := exec.CommandContext(ts.ctx, "node", "./sbot_serv.js")
	cmd.Stderr = os.Stderr

	outrc, err := cmd.StdoutPipe()
	r.NoError(err)

	if name == "" {
		name = fmt.Sprintf("jsbot-%d", jsBotCnt)
	}
	jsBotCnt++

	var port = 1024 + mrand.Intn(23000)

	env := []string{
		"TEST_NAME=" + filepath.Join(ts.t.Name(), "jsbot-"+name),
		"TEST_BOB=" + ts.gobot.Whoami().Ref(),
		fmt.Sprintf("TEST_PORT=%d", port),
		"TEST_BEFORE=" + testScriptFileName,
	}
	// if ts.keySHS != nil {
	// 	env = append(env, "TEST_APPKEY="+base64.StdEncoding.EncodeToString(ts.keySHS))
	// }
	cmd.Env = env

	r.NoError(cmd.Start(), "failed to init test js-sbot")

	ts.done.Go(cmd.Wait)
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
	closeErrc := make(chan error)

	go func() {
		time.Sleep(15 * time.Second) // would be nice to get -test.timeout for this

		ts.gobot.Shutdown()
		closeErrc <- ts.gobot.Close()
		close(closeErrc)
	}()

	for err := range testutils.MergeErrorChans(append(ts.backgroundErrs, closeErrc)...) {
		require.NoError(ts.t, err)
	}

	require.NoError(ts.t, ts.done.Wait())
}
