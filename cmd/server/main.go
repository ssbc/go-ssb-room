// SPDX-License-Identifier: MIT

// go-roomsrv hosts the database and p2p server for replication.
// It supplies various flags to contol options.
// See 'go-roomsrv -h' for a list and their usage.
package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	// debug
	"net/http"
	_ "net/http/pprof"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2/debug"

	"go.mindeco.de/ssb-rooms/roomsrv"
	mksrv "go.mindeco.de/ssb-rooms/roomsrv"
	"go.mindeco.de/ssb-rooms/web/handlers"
)

var (
	// flags
	flagDisableUNIXSock bool

	listenAddrShsMux string
	listenAddrHTTP   string

	listenAddrDebug string
	logToFile       string
	repoDir         string

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

	flag.StringVar(&listenAddrShsMux, "lismux", ":8008", "address to listen on for secret-handshake+muxrpc")
	flag.StringVar(&listenAddrHTTP, "lishttp", ":3000", "address to listen on for HTTP requests")

	flag.BoolVar(&flagDisableUNIXSock, "nounixsock", false, "disable the UNIX socket RPC interface")

	flag.StringVar(&repoDir, "repo", filepath.Join(u.HomeDir, ".ssb-go-room"), "where to put the log and indexes")

	flag.StringVar(&listenAddrDebug, "dbg", "localhost:6078", "listen addr for metrics and pprof HTTP server")
	flag.StringVar(&logToFile, "logs", "", "where to write debug output to (default is just stderr)")

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
		log = kitlog.NewJSONLogger(kitlog.NewSyncWriter(logFile))
	} else {
		log = kitlog.NewLogfmtLogger(os.Stderr)
	}
}

func runroomsrv() error {
	initFlags()

	if flagPrintVersion {
		log.Log("version", Version, "build", Build)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ak, err := base64.StdEncoding.DecodeString(appKey)
	if err != nil {
		return fmt.Errorf("application key: %w", err)
	}

	opts := []roomsrv.Option{
		roomsrv.WithLogger(log),
		roomsrv.WithAppKey(ak),
		roomsrv.WithRepoPath(repoDir),
		roomsrv.WithListenAddr(listenAddrShsMux),
		roomsrv.WithUNIXSocket(!flagDisableUNIXSock),
	}

	if logToFile != "" {
		opts = append(opts, roomsrv.WithPostSecureConnWrapper(func(conn net.Conn) (net.Conn, error) {
			parts := strings.Split(conn.RemoteAddr().String(), "|")

			if len(parts) != 2 {
				return conn, nil
			}

			muxrpcDumpDir := filepath.Join(
				repoDir,
				logToFile,
				parts[1], // key first
				parts[0],
			)

			return debug.WrapDump(muxrpcDumpDir, conn)
		}))
	}

	if listenAddrDebug != "" {
		go func() {
			// http.Handle("/metrics", promhttp.Handler())
			level.Debug(log).Log("starting", "metrics", "addr", listenAddrDebug)
			err := http.ListenAndServe(listenAddrDebug, nil)
			checkAndLog(err)
		}()
	}

	// create the shs+muxrpc server
	roomsrv, err := mksrv.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to instantiate ssb server: %w", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		level.Warn(log).Log("event", "killed", "msg", "received signal, shutting down", "signal", sig.String())
		cancel()
		roomsrv.Shutdown()
		time.Sleep(2 * time.Second)

		err := roomsrv.Close()
		checkAndLog(err)

		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// setup web dashboard handlers
	dashboardH, err := handlers.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTPdashboard handler: %w", err)
	}

	// open the HTTP listener
	httpLis, err := net.Listen("tcp", listenAddrHTTP)
	if err != nil {
		return fmt.Errorf("failed to open listener for HTTPdashboard: %w", err)
	}

	level.Info(log).Log(
		"event", "serving",
		"ID", roomsrv.Whoami().Ref(),
		"shsmuxaddr", listenAddrShsMux,
		"httpaddr", listenAddrHTTP,
		"version", Version,
		"build", Build,
	)

	// start serving http connections
	go func() {
		err = http.Serve(httpLis, dashboardH)
		if err != nil {
			level.Error(log).Log("event", "http serve failed", "err", err)
		}
	}()

	// start serving shs+muxrpc connections
	for {
		// Note: This is where the serving starts ;)
		err = roomsrv.Network.Serve(ctx)
		if err != nil {
			level.Warn(log).Log("event", "roomsrv node.Serve returned", "err", err)
		}

		time.Sleep(1 * time.Second)
		select {
		case <-ctx.Done():
			err := roomsrv.Close()
			return err
		default:
		}
	}
}

func main() {
	if err := runroomsrv(); err != nil {
		fmt.Fprintf(os.Stderr, "go-ssb-tunnel: %s\n", err)
		os.Exit(1)
	}
}
