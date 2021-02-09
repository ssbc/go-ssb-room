// SPDX-License-Identifier: MIT

package roomsrv

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/gossb-rooms/internal/netwraputil"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/repo"
)

// WithUNIXSocket enables listening for muxrpc connections on a unix socket files ($repo/socket).
// This socket is not encrypted or authenticated since access to it is mediated by filesystem ownership.
func WithUNIXSocket(yes bool) Option {
	return func(s *Server) error {
		s.loadUnixSock = yes
		return nil
	}
}

func (s *Server) initUnixSock() error {
	// this races because sbot might not be done with init yet
	// TODO: refactor network peer code and make unixsock implement that (those will be inited late anyway)
	if s.keyPair == nil {
		return fmt.Errorf("sbot/unixsock: keypair is nil. please use unixSocket with LateOption")
	}
	spoofWrapper := netwraputil.SpoofRemoteAddress(s.keyPair.Feed.ID)

	r := repo.New(s.repoPath)
	sockPath := r.GetPath("socket")

	// local clients (not using network package because we don't want conn limiting or advertising)
	c, err := net.Dial("unix", sockPath)
	if err == nil {
		c.Close()
		return fmt.Errorf("sbot: repo already in use, socket accepted connection")
	}
	os.Remove(sockPath)
	os.MkdirAll(filepath.Dir(sockPath), 0700)

	uxLis, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}
	s.closers.Add(uxLis)

	go func() {

		for {
			c, err := uxLis.Accept()
			if err != nil {
				if nerr, ok := err.(*net.OpError); ok {
					if nerr.Err.Error() == "use of closed network connection" {
						return
					}
				}

				level.Warn(s.logger).Log("event", "unix sock accept failed", "err", err)
				continue
			}

			wc, err := spoofWrapper(c)
			if err != nil {
				c.Close()
				continue
			}
			for _, w := range s.postSecureWrappers {
				var err error
				wc, err = w(wc)
				if err != nil {
					level.Warn(s.logger).Log("err", err)
					c.Close()
					continue
				}
			}

			go func(conn net.Conn) {
				defer conn.Close()

				pkr := muxrpc.NewPacker(conn)

				h, err := s.master.MakeHandler(conn)
				if err != nil {
					level.Warn(s.logger).Log("event", "unix sock make handler", "err", err)
					return
				}

				edp := muxrpc.Handle(pkr, h,
					muxrpc.WithContext(s.rootCtx),
					muxrpc.WithLogger(kitlog.NewNopLogger()),
				)

				srv := edp.(muxrpc.Server)
				if err := srv.Serve(); err != nil {
					level.Warn(s.logger).Log("conn", "serve exited", "err", err, "peer", conn.RemoteAddr())
				}
				edp.Terminate()

			}(wc)
		}
	}()
	return nil

}
