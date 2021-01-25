// SPDX-License-Identifier: MIT

// go-tunsrv hosts the database and p2p server for replication.
// It supplies various flags to contol options.
// See 'go-tunsrv -h' for a list and their usage.
package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	// debug
	"net/http"
	_ "net/http/pprof"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

var (
	// flags
	flagDisableUNIXSock bool

	listenAddr string
	debugAddr  string
	logToFile  string
	repoDir    string

	// helper
	log kitlog.Logger

	// juicy bits
	appKey string
)

// Version and Build are set by ldflags
var (
	Version = "snapshot"
	Build   = ""

	flagPrintVersion bool
)

func checkFatal(err error) {
	checkAndLog(err)
	if err != nil {
		os.Exit(1)
	}
}

func checkAndLog(err error) {
	if err != nil {
		level.Error(log).Log("event", "fatal error", "err", err)
	}
}

func initFlags() {
	u, err := user.Current()
	checkFatal(err)

	flag.StringVar(&appKey, "shscap", "1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=", "secret-handshake app-key (or capability)")

	flag.StringVar(&listenAddr, "l", ":8008", "address to listen on")

	flag.BoolVar(&flagDisableUNIXSock, "nounixsock", false, "disable the UNIX socket RPC interface")

	flag.StringVar(&repoDir, "repo", filepath.Join(u.HomeDir, ".ssb-go"), "where to put the log and indexes")

	flag.StringVar(&debugAddr, "dbg", "localhost:6078", "listen addr for metrics and pprof HTTP server")
	flag.StringVar(&logToFile, "path", "", "where to write debug output to (otherwise just stderr)")

	flag.BoolVar(&flagPrintVersion, "version", false, "print version number and build date")

	flag.Parse()

	if logToFile != "" {
		logDir := filepath.Join(repoDir, logToFile)
		os.MkdirAll(logDir, 0700) // nearly everything is a log here so..
		logFileName := fmt.Sprintf("%s-%s.log",
			filepath.Base(os.Args[0]),
			time.Now().Format("2006-01-02_15-04"))
		logFile, err := os.Create(filepath.Join(logDir, logFileName))
		if err != nil {
			panic(err) // logging not ready yet...
		}
		log = kitlog.NewJSONLogger(logFile)
	} else {
		log = kitlog.NewJSONLogger(os.Stderr)
	}
}

func runtunsrv() error {
	// DEBUGGING
	// runtime.SetMutexProfileFraction(1)
	// runtime.SetBlockProfileRate(1)
	// DEBUGGING

	initFlags()

	if flagPrintVersion {
		log.Log("version", Version, "build", Build)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	ak, err := base64.StdEncoding.DecodeString(appKey)
	if err != nil {
		return errors.Wrap(err, "application key")
	}

	if !flagDisableUNIXSock {
		// opts = append(opts, mktunsrv.LateOption(mktunsrv.WithUNIXSocket()))
	}

	// if dbgLogDir != "" {
	// 	opts = append(opts, mktunsrv.WithPostSecureConnWrapper(func(conn net.Conn) (net.Conn, error) {
	// 		parts := strings.Split(conn.RemoteAddr().String(), "|")

	// 		if len(parts) != 2 {
	// 			return conn, nil
	// 		}

	// 		muxrpcDumpDir := filepath.Join(
	// 			repoDir,
	// 			dbgLogDir,
	// 			parts[1], // key first
	// 			parts[0],
	// 		)

	// 		return debug.WrapDump(muxrpcDumpDir, conn)
	// 	}))
	// }

	if debugAddr != "" {
		go func() {
			// http.Handle("/metrics", promhttp.Handler())
			log.Log("starting", "metrics", "addr", debugAddr)
			err := http.ListenAndServe(debugAddr, nil)
			checkAndLog(err)
		}()
	}

	tunsrv, err := makeTunnelServer.New(opts...)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate ssb server")
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		level.Warn(log).Log("event", "killed", "msg", "received signal, shutting down", "signal", sig.String())
		cancel()
		tunsrv.Shutdown()
		time.Sleep(2 * time.Second)

		err := tunsrv.Close()
		checkAndLog(err)

		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	level.Info(log).Log("event", "serving", "ID", id.Ref(), "addr", listenAddr, "version", Version, "build", Build)
	for {
		// Note: This is where the serving starts ;)
		err = tunsrv.Network.Serve(ctx)
		if err != nil {
			level.Warn(log).Log("event", "tunsrv node.Serve returned", "err", err)
		}

		time.Sleep(1 * time.Second)
		select {
		case <-ctx.Done():
			err := tunsrv.Close()
			return err
		default:
		}
	}
}

func main() {
	if err := runtunsrv(); err != nil {
		fmt.Fprintf(os.Stderr, "go-ssb-tunnel: %s\n", err)
		os.Exit(1)
	}
}
