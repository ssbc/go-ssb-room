// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package gossip

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/ssbc/go-muxrpc/v2"
	"go.mindeco.de/encodedTime"
)

// Ping implements the server side of gossip.ping.
// it's idea is mentioned here https://github.com/ssbc/ssb-gossip/#ping-duplex
// and implemented by https://github.com/dominictarr/pull-ping/
func Ping(ctx context.Context, req *muxrpc.Request, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	type arg struct {
		// The only argument is the delay between two pings.
		// the Javascript code calls this "timeout", tho.
		Delay int `json:"timeout"`
	}

	var args []arg
	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return err
	}

	// var timeout = time.Minute * 5
	// if len(args) == 1 {
	// 	timeout = time.Minute * time.Duration(args[0].Timeout/(60*1000))
	// }

	// return sillyPingPong(ctx, peerSrc, peerSnk)
	return actualPingPong(ctx, peerSrc, peerSnk)
}

// actually just read and write whenever...
func sillyPingPong(ctx context.Context, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	var (
		sendErr    = make(chan error)
		receiveErr = make(chan error)
	)

	go func() {
		peerSnk.SetEncoding(muxrpc.TypeJSON)
		enc := json.NewEncoder(peerSnk)

		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()

		defer close(sendErr)

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}

			var pong = encodedTime.Millisecs(time.Now())

			if err := enc.Encode(pong); err != nil {
				sendErr <- err
				return
			}
		}
	}()

	go func() {
		defer close(receiveErr)

		for peerSrc.Next(ctx) {
			var ping encodedTime.Millisecs
			err := peerSrc.Reader(func(rd io.Reader) error {
				return json.NewDecoder(rd).Decode(&ping)
			})
			if err != nil {
				receiveErr <- err
				return
			}

		}

		return
	}()

	select {
	case e := <-sendErr:
		return e
	case e := <-receiveErr:
		return e
	case <-ctx.Done():
		return nil
	}
}

// this is how it should work, i think, but it leads to disconnects...
// From the code it's hard to see but the client sends a timestamp in milliseconds (Date.now() in javascript/json)
// and the other side responds with it's own timestamp.
func actualPingPong(ctx context.Context, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	peerSnk.SetEncoding(muxrpc.TypeJSON)
	enc := json.NewEncoder(peerSnk)

	for peerSrc.Next(ctx) {
		var ping encodedTime.Millisecs
		err := peerSrc.Reader(func(rd io.Reader) error {
			return json.NewDecoder(rd).Decode(&ping)
		})
		if err != nil {
			return err
		}

		//when := time.Time(ping)
		//fmt.Printf("got ping: %s - age: %s\n", when.String(), time.Since(when))

		pong := encodedTime.Millisecs(time.Now())
		err = enc.Encode(pong)
		if err != nil {
			return err
		}

		// time.Sleep(timeout)
	}

	return peerSrc.Err()
}
