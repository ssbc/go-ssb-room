package go_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"go.cryptoscope.co/secretstream"
	"go.mindeco.de/encodedTime"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	tunserv "github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/tunnel/server"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

func TestWebsocketDialing(t *testing.T) {
	r := require.New(t)

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// create the roomsrv
	session := makeNamedTestBot(t, "server", ctx, nil)
	server := session.srv

	// open a TCP listener for HTTP
	l, err := net.Listen("tcp4", "localhost:0")
	r.NoError(err)

	var fh = failHandler{t: t}

	// serve the websocket handler
	handler := server.Network.WebsockHandler(fh)
	go http.Serve(l, handler)

	// create a fresh keypair for the client
	client, err := keys.NewKeyPair(nil)
	r.NoError(err)

	// add it as a memeber
	memberID, err := server.Members.Add(ctx, client.Feed, roomdb.RoleMember)
	r.NoError(err)
	t.Log("client member:", memberID)

	// construct the websocket http address
	var wsURL url.URL
	wsURL.Scheme = "ws"
	wsURL.Host = l.Addr().String()
	wsURL.Path = "/"

	// create a websocket connection
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), nil)
	r.NoError(err)
	t.Log(resp.Status)
	r.Equal(http.StatusSwitchingProtocols, resp.StatusCode)

	// default app key for the secret-handshake connection
	ak, err := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
	r.NoError(err)

	// create a shs client to authenticate and encrypt the connection
	clientSHS, err := secretstream.NewClient(client.Pair, ak)
	r.NoError(err)

	// the returned wrapper: func(net.Conn) (net.Conn, error)
	// returns a new connection that went through shs and does boxstream
	connAuther := clientSHS.ConnWrapper(server.Whoami().PubKey())

	// turn a websocket conn into a net.Conn
	wrappedConn := network.NewWebsockConn(conn)

	authedConn, err := connAuther(wrappedConn)
	r.NoError(err)

	var muxMock muxrpc.FakeHandler

	debugConn := debug.Dump(filepath.Join(testPath, "client"), authedConn)
	pkr := muxrpc.NewPacker(debugConn)

	wsEndpoint := muxrpc.Handle(pkr, &muxMock, muxrpc.WithContext(ctx))

	srv := wsEndpoint.(muxrpc.Server)
	go func() {
		err = srv.Serve()
		r.NoError(err)
		t.Log("mux server error:", err)
	}()

	// check we are talking to a room
	var meta tunserv.MetadataReply
	err = wsEndpoint.Async(ctx, &meta, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "isRoom"})
	r.NoError(err)
	r.Equal("server", meta.Name)
	r.True(meta.Membership, "not a member?")

	// open the gossip.ping channel
	src, snk, err := wsEndpoint.Duplex(ctx, muxrpc.TypeJSON, muxrpc.Method{"gossip", "ping"})
	r.NoError(err)

	pingTS := encodedTime.NewMillisecs(time.Now().Unix())
	err = json.NewEncoder(snk).Encode(pingTS)
	r.NoError(err)

	r.True(src.Next(ctx))
	pong, err := src.Bytes()
	r.NoError(err)

	var pongTS encodedTime.Millisecs
	err = json.Unmarshal(pong, &pongTS)
	r.NoError(err)
	r.False(time.Time(pongTS).IsZero())
}

type failHandler struct {
	t *testing.T
}

func (fh failHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fh.t.Error("next handler called", req.URL.String())
}
