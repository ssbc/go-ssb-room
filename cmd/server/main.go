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
	_ "github.com/mattn/go-sqlite3"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/unrolled/secure"
	"go.cryptoscope.co/muxrpc/v2/debug"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
	mksrv "github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/handlers"
)

var (
	// flags
	flagDisableUNIXSock bool

	listenAddrShsMux string
	listenAddrHTTP   string

	httpsDomain string

	aliasesAsSubdomains bool

	listenAddrDebug string
	logToFile       string
	repoDir         string

	privacyMode = roomdb.ModeUnknown

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

	flag.StringVar(&httpsDomain, "https-domain", "", "which domain to use for TLS and AllowedHosts checks")

	flag.BoolVar(&flagPrintVersion, "version", false, "print version number and build date")

	flag.Func("mode", "the privacy mode (values: open, community, restricted) determining room access controls", func(val string) error {
		pm := roomdb.ParsePrivacyMode(val)
		err := pm.IsValid()
		if err != nil {
			return fmt.Errorf("%s, valid values are open, community, restricted", err)
		}
		privacyMode = pm
		return nil
	})

	flag.BoolVar(&aliasesAsSubdomains, "aliases-as-subdomains", true, "needs to be disabled if a wildcard certificate for the room is not available. (stub until we have the admin/settings page)")

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

	if httpsDomain == "" {
		if !development {
			return fmt.Errorf("https-domain can't be empty. See '%s -h' for a full list of options", os.Args[0])
		}
		httpsDomain = "localhost"
	}

	// validate listen addresses to bail out on invalid flag input before doing anything else
	_, muxrpcPortStr, err := net.SplitHostPort(listenAddrShsMux)
	if err != nil {
		return fmt.Errorf("invalid muxrpc listener: %w", err)
	}

	_, err = net.LookupPort("tcp", muxrpcPortStr)
	if err != nil {
		return fmt.Errorf("invalid tcp port for muxrpc listener: %w", err)
	}

	_, portHTTPStr, err := net.SplitHostPort(listenAddrHTTP)
	if err != nil {
		return fmt.Errorf("invalid http listener: %w", err)
	}

	portHTTP, err := net.LookupPort("tcp", portHTTPStr)
	if err != nil {
		return fmt.Errorf("invalid tcp port for muxrpc listener: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ak, err := base64.StdEncoding.DecodeString(appKey)
	if err != nil {
		return fmt.Errorf("secret-handshake appkey is invalid base64: %w", err)
	}

	opts := []roomsrv.Option{
		roomsrv.WithLogger(log),
		roomsrv.WithAppKey(ak),
		roomsrv.WithRepoPath(repoDir),
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

	r := repo.New(repoDir)

	keyPair, err := repo.DefaultKeyPair(r)
	checkAndLog(err)
	opts = append(opts, roomsrv.WithKeyPair(keyPair))

	networkInfo := network.ServerEndpointDetails{
		Development: development,

		Domain:    httpsDomain,
		PortHTTPS: uint(portHTTP),

		RoomID: keyPair.Feed,

		ListenAddressMUXRPC: listenAddrShsMux,

		UseSubdomainForAliases: aliasesAsSubdomains,
	}

	// open the sqlite version of the roomdb
	db, err := sqlite.Open(r)
	if err != nil {
		return fmt.Errorf("failed to initiate database: %w", err)
	}

	bridge := signinwithssb.NewSignalBridge()
	// the privacy mode flag was passed => update it in the database
	if privacyMode != roomdb.ModeUnknown {
		db.Config.SetPrivacyMode(ctx, privacyMode)
	}

	// create the shs+muxrpc server
	roomsrv, err := mksrv.New(
		db.Members,
		db.DeniedKeys,
		db.Aliases,
		db.AuthWithSSB,
		bridge,
		db.Config,
		networkInfo,
		opts...)
	if err != nil {
		return fmt.Errorf("failed to instantiate ssb server: %w", err)
	}

	// open the HTTP listener
	httpLis, err := net.Listen("tcp", listenAddrHTTP)
	if err != nil {
		return fmt.Errorf("failed to open listener for HTTPdashboard: %w", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		level.Warn(log).Log("event", "killed", "msg", "received signal, shutting down", "signal", sig.String())
		cancel()
		roomsrv.Shutdown()

		httpLis.Close()
		time.Sleep(2 * time.Second)

		err := roomsrv.Close()
		checkAndLog(err)

		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// setup web dashboard handlers
	webHandler, err := handlers.New(
		kitlog.With(log, "package", "web"),
		repo.New(repoDir),
		networkInfo,
		roomsrv.StateManager,
		roomsrv.Network,
		bridge,
		handlers.Databases{
			Aliases:       db.Aliases,
			AuthFallback:  db.AuthFallback,
			AuthWithSSB:   db.AuthWithSSB,
			Config:        db.Config,
			DeniedKeys:    db.DeniedKeys,
			Invites:       db.Invites,
			Notices:       db.Notices,
			Members:       db.Members,
			PinnedNotices: db.PinnedNotices,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTPdashboard handler: %w", err)
	}

	// setup CSP and HTTPS redirects
	secureMiddleware := secure.New(secure.Options{
		IsDevelopment: development,

		AllowedHosts: []string{
			// the normal domain
			httpsDomain,
			// the domain but as a wildcard match with *. infront
			`*\.` + strings.Replace(httpsDomain, ".", `\.`, -1),
		},

		// for the wildcard matching
		AllowedHostsAreRegex: true,

		// TLS stuff
		SSLRedirect: true,
		SSLHost:     httpsDomain,

		// Important for reverse-proxy setups (when nginx or similar does the TLS termination)
		SSLProxyHeaders:   map[string]string{"X-Forwarded-Proto": "https"},
		HostsProxyHeaders: []string{"X-Forwarded-Host"},

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
		STSSeconds: 2592000, // 30 days in seconds (TODO configure?)
		STSPreload: false,   // don't submit to googles list service (TODO configure?)
		// TODO configure (could be needed in special setups where the room is a subdomain of a site)
		STSIncludeSubdomains: false,

		// See for more https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
		// helpful: https://report-uri.com/home/generate
		ContentSecurityPolicy: "default-src 'self'; img-src 'self' data:", // enforce no external content

		BrowserXssFilter: true,
		FrameDeny:        true,
		//ContentTypeNosniff: true, // TODO: fix Content-Type headers served from assets
	})

	// HTTP rate limiter
	throttleStore, err := memstore.New(65536) // 64k different combinations of limitByPathAndAddr
	if err != nil {
		return fmt.Errorf("failed to init HTTP rate limiter store: %w", err)
	}
	quota := throttled.RateQuota{
		MaxRate:  throttled.PerSec(5), // different requests per second per VaryBy
		MaxBurst: 25,
	}
	limiter, err := throttled.NewGCRARateLimiter(throttleStore, quota)
	if err != nil {
		return fmt.Errorf("failed to init HTTP rate limiter: %w", err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: limiter,
		VaryBy:      limitByPathAndAddr{},
	}

	// wrap dashboard/alias/invite handler in ratlimiter and security middleware
	var httpHandler http.Handler
	httpHandler = httpRateLimiter.RateLimit(webHandler)
	httpHandler = secureMiddleware.Handler(httpHandler)
	httpHandler = roomsrv.Network.WebsockHandler(httpHandler)

	// all init was successfull
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
		srv := http.Server{
			Addr: httpLis.Addr().String(),

			// Good practice to set timeouts to avoid Slowloris attacks.
			// Keep in mind that the SSE stuff for "sign-in with ssb" can take a moment, thou
			ReadHeaderTimeout: time.Second * 15,
			WriteTimeout:      time.Minute * 3,
			IdleTimeout:       time.Minute * 3,

			Handler: httpHandler,
		}

		err = srv.Serve(httpLis)
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
		fmt.Fprintf(os.Stderr, "go-ssb-room: %s\n", err)
		os.Exit(1)
	}
}

type limitByPathAndAddr struct{}

func (limitByPathAndAddr) Key(r *http.Request) string {
	var k strings.Builder

	k.WriteString(r.URL.Path)
	k.WriteString("\n")

	remoteIP := r.Header.Get("X-Forwarded-For")
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	k.WriteString(remoteIP)

	return k.String()
}
