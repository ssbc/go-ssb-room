// SPDX-License-Identifier: MIT

package roomsrv

import (
	"context"
	"fmt"
	"net"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"go.cryptoscope.co/netwrap"
)

type Option func(srv *Server) error

// WithRepoPath changes where the replication database and blobs are stored.
func WithRepoPath(path string) Option {
	return func(s *Server) error {
		s.repoPath = path
		return nil
	}
}

// WithAppKey changes the appkey (aka secret-handshake network cap).
// See https://ssbc.github.io/scuttlebutt-protocol-guide/#handshake for more.
func WithAppKey(k []byte) Option {
	return func(s *Server) error {
		if n := len(k); n != 32 {
			return fmt.Errorf("appKey: need 32 bytes got %d", n)
		}
		s.appKey = k
		return nil
	}
}

// WithNamedKeyPair changes from the default `secret` file, useful for testing.
func WithNamedKeyPair(name string) Option {
	return func(s *Server) error {
		r := repo.New(s.repoPath)
		var err error
		s.keyPair, err = repo.LoadKeyPair(r, name)
		if err != nil {
			return fmt.Errorf("loading named key-pair %q failed: %w", name, err)
		}
		return nil
	}
}

// WithJSONKeyPair expectes a JSON-string as blob and calls ssb.ParseKeyPair on it.
// This is useful if you dont't want to place the keypair on the filesystem.
func WithJSONKeyPair(blob string) Option {
	return func(s *Server) error {
		var err error
		s.keyPair, err = keys.ParseKeyPair(strings.NewReader(blob))
		if err != nil {
			return fmt.Errorf("JSON KeyPair decode failed: %w", err)
		}
		return nil
	}
}

// WithKeyPair exepect a initialized ssb.KeyPair. Useful for testing.
func WithKeyPair(kp *keys.KeyPair) Option {
	return func(s *Server) error {
		s.keyPair = kp
		return nil
	}
}

// WithLogger changes the info/warn/debug loging output.
func WithLogger(log kitlog.Logger) Option {
	return func(s *Server) error {
		s.logger = log
		return nil
	}
}

// WithContext changes the context that is context.Background() by default.
// Handy to setup cancelation against a interup signal like ctrl+c.
// Canceling the context also shuts down indexing. If no context is passed sbot.Shutdown() can be used.
func WithContext(ctx context.Context) Option {
	return func(s *Server) error {
		s.rootCtx, s.Shutdown = context.WithCancel(ctx)
		return nil
	}
}

// TODO: remove all this network stuff and make them options on network

// WithDialer changes the function that is used to dial remote peers.
// This could be a sock5 connection builder to support tor proxying to hidden services.
func WithDialer(dial netwrap.Dialer) Option {
	return func(s *Server) error {
		s.dialer = dial
		return nil
	}
}

// WithNetworkConnTracker changes the connection tracker. See network.NewLastWinsTracker and network.NewAcceptAllTracker.
func WithNetworkConnTracker(ct network.ConnTracker) Option {
	return func(s *Server) error {
		s.networkConnTracker = ct
		return nil
	}
}

// WithListenAddr changes the muxrpc listener address. By default it listens to ':8008'.
func WithListenAddr(addr string) Option {
	return func(s *Server) error {
		var err error
		s.listenAddr, err = net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to parse tcp listen addr: %w", err)
		}
		return nil
	}
}

// WithPreSecureConnWrapper wrapps the connection after it is encrypted.
// Usefull for debugging and measuring traffic.
func WithPreSecureConnWrapper(cw netwrap.ConnWrapper) Option {
	return func(s *Server) error {
		s.preSecureWrappers = append(s.preSecureWrappers, cw)
		return nil
	}
}

// WithPostSecureConnWrapper wrapps the connection before it is encrypted.
// Usefull to insepct the muxrpc frames before they go into boxstream.
func WithPostSecureConnWrapper(cw netwrap.ConnWrapper) Option {
	return func(s *Server) error {
		s.postSecureWrappers = append(s.postSecureWrappers, cw)
		return nil
	}
}
