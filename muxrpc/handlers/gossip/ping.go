package gossip

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"go.cryptoscope.co/muxrpc/v2"
	"go.mindeco.de/encodedTime"
)

func Ping(ctx context.Context, req *muxrpc.Request, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	type arg struct {
		Timeout int
	}

	var args []arg
	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return err
	}

	var timeout = time.Minute * 5
	if len(args) == 1 {
		timeout = time.Minute * time.Duration(args[0].Timeout/(60*1000))
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		peerSnk.SetEncoding(muxrpc.TypeJSON)
		enc := json.NewEncoder(peerSnk)

		tick := time.NewTicker(time.Second)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}

			var pong = encodedTime.Millisecs(time.Now())
			err = enc.Encode(pong)
			if err != nil {
				return
			}
		}
	}()

	for peerSrc.Next(ctx) {
		var ping encodedTime.Millisecs
		err := peerSrc.Reader(func(rd io.Reader) error {
			return json.NewDecoder(rd).Decode(&ping)
		})
		if err != nil {
			return err
		}

		// when := time.Time(ping)
		// fmt.Printf("got ping: %s - age: %s\n", when.String(), time.Since(when))
	}

	return nil
}
