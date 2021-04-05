// SPDX-License-Identifier: MIT

package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/websocket"
	"go.cryptoscope.co/muxrpc/v2"
)

// WebsockHandler returns a "middleware" like thing that is able to upgrade a websocket request to a muxrpc connection and authenticate using shs.
// It calls the next handler if it fails to upgrade the connection to websocket.
// However, it will error on the request and not call the passed handler
// if the websocket upgrade is successfull.
func (n *node) WebsockHandler(next http.Handler) http.Handler {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024 * 4,
		WriteBufferSize: 1024 * 4,

		CheckOrigin: func(_ *http.Request) bool {
			return true
		},

		// 99% of the traffic will be ciphertext which is impossible to distingish from randomness and thus also hard to compress
		EnableCompression: false,

		// if upgrading fails, just call the next handler and ignore the error
		Error: func(w http.ResponseWriter, req *http.Request, _ int, _ error) {
			next.ServeHTTP(w, req)
		},
	}

	var wsh websocketHandelr
	wsh.next = next
	wsh.upgrader = &upgrader
	wsh.muxnetwork = n

	return wsh
}

type websocketHandelr struct {
	next http.Handler

	muxnetwork *node

	upgrader *websocket.Upgrader
}

func (wsh websocketHandelr) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	remoteAddrStr := req.Header.Get("X-Forwarded-For")
	if remoteAddrStr == "" {
		remoteAddrStr = req.RemoteAddr
	}

	remoteAddr, err := net.ResolveTCPAddr("tcp", remoteAddrStr)
	if err != nil {
		wsh.next.ServeHTTP(w, req)
		return
	}

	wsConn, err := wsh.upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}

	errLog := level.Error(wsh.muxnetwork.log)
	errLog = kitlog.With(errLog, "remote", remoteAddrStr)

	var wc net.Conn
	wc = NewWebsockConn(wsConn)

	cw := wsh.muxnetwork.secretServer.ConnWrapper()
	wc, err = cw(wc)
	if err != nil {
		errLog.Log("warning", "failed to authenticate", "err", err, "remote", remoteAddr)
		wsConn.Close()
		return
	}

	// debugging copy of all muxrpc frames
	// can be handy for reversing applications
	// feed, err := GetFeedRefFromAddr(wc.RemoteAddr())
	// if err != nil {
	// 	errLog.Log("warning", "failed to get feed after auth", "err", err, "remote", remoteAddr)
	// 	wsConn.Close()
	// 	return
	// }
	// dumpPath := filepath.Join("webmux", base64.URLEncoding.EncodeToString(feed.ID[:10]), remoteAddrStr)
	// wc, err = debug.WrapDump(dumpPath, wc)
	// if err != nil {
	// 	errLog.Log("warning", "failed wrap", "err", err, "remote", remoteAddr)
	// 	wsConn.Close()
	// 	return
	// }

	pkr := muxrpc.NewPacker(wc)

	h, err := wsh.muxnetwork.opts.MakeHandler(wc)
	if err != nil {
		err = fmt.Errorf("websocket make handler failed: %w", err)
		errLog.Log("warn", err)
		wsConn.Close()
		return
	}

	edp := muxrpc.Handle(pkr, h,
		muxrpc.WithContext(req.Context()),
		muxrpc.WithRemoteAddr(wc.RemoteAddr()))

	srv := edp.(muxrpc.Server)
	if err := srv.Serve(); err != nil {
		errLog.Log("conn", "serve exited", "err", err, "peer", remoteAddr)
	}
	wsConn.Close()
}

// WebsockConn emulates a normal net.Conn from a websocket connection
type WebsockConn struct {
	r   io.Reader
	wsc *websocket.Conn
}

func NewWebsockConn(wsc *websocket.Conn) *WebsockConn {
	return &WebsockConn{
		wsc: wsc,
	}
}

func (conn *WebsockConn) Read(data []byte) (int, error) {
	if conn.r == nil {
		if err := conn.renewReader(); err != nil {
			return -1, err
		}

	}

	n, err := conn.r.Read(data)
	if err == io.EOF {
		if err := conn.renewReader(); err != nil {
			return -1, err
		}
		n, err = conn.Read(data)
		if err != nil {
			err = fmt.Errorf("wsConn: failed to read after renew(): %w", err)
			return -1, err
		}
		return n, nil
	}

	if err != nil {
		return -1, fmt.Errorf("wsConn: read failed: %w", err)
	}

	return n, nil
}

func (conn *WebsockConn) renewReader() error {
	mt, r, err := conn.wsc.NextReader()
	if err != nil {
		err = fmt.Errorf("wsConn: failed to get reader: %w", err)
		return err
	}

	if mt != websocket.BinaryMessage {
		return fmt.Errorf("wsConn: not binary message: %v", mt)

	}

	conn.r = r
	return nil
}

func (conn *WebsockConn) Write(data []byte) (int, error) {
	writeCloser, err := conn.wsc.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return -1, fmt.Errorf("wsConn: failed to create Reader: %w", err)
	}

	n, err := io.Copy(writeCloser, bytes.NewReader(data))
	if err != nil {
		return -1, fmt.Errorf("wsConn: failed to copy data: %w", err)
	}

	return int(n), writeCloser.Close()
}

func (conn *WebsockConn) Close() error {
	return conn.wsc.Close()
}

func (conn *WebsockConn) LocalAddr() net.Addr  { return conn.wsc.LocalAddr() }
func (conn *WebsockConn) RemoteAddr() net.Addr { return conn.wsc.RemoteAddr() }

func (conn *WebsockConn) SetDeadline(t time.Time) error {
	rErr := conn.wsc.SetReadDeadline(t)
	wErr := conn.wsc.SetWriteDeadline(t)

	var err error

	if rErr != nil {
		err = fmt.Errorf("websock conn: failed to set read deadline: %w", rErr)
	}

	if wErr != nil {
		wErr = fmt.Errorf("websock conn: failed to set read deadline: %w", wErr)
		if err != nil {
			err = fmt.Errorf("both faild: %w and %s", err, wErr)
		} else {
			err = wErr
		}
	}

	return err
}
func (conn *WebsockConn) SetReadDeadline(t time.Time) error {
	return conn.wsc.SetReadDeadline(t)
}
func (conn *WebsockConn) SetWriteDeadline(t time.Time) error {
	return conn.wsc.SetWriteDeadline(t)
}
