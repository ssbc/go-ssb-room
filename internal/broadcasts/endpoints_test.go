// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package broadcasts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type testPrinter struct {
	w io.Writer
}

func (tp testPrinter) Update(members []string) error {
	fmt.Fprintf(tp.w, "test: %d\n", len(members))
	return nil
}

func (tp testPrinter) Close() error { return nil }

func ExampleBroadcast() {
	sink, bcast := NewEndpointsEmitter()
	defer sink.Close()

	var p1, p2 testPrinter
	p1.w = os.Stdout
	p2.w = os.Stdout

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)
	defer closeSink()

	sink.Update([]string{"whoop", "whoop"})

	// Output:
	// test: 2
	// test: 2
}

func ExampleBroadcastCanceled() {
	sink, bcast := NewEndpointsEmitter()
	defer sink.Close()

	var p1, p2 testPrinter
	p1.w = os.Stdout
	p2.w = os.Stdout

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)

	closeSink() // p2 never prints

	sink.Update([]string{"hi"})

	// Output:
	// test: 1
}

type erroringPrinter struct {
	w io.Writer
}

func (tp erroringPrinter) Update(m []string) error {
	fmt.Fprintf(tp.w, "failed: %d\n", len(m))
	// time.Sleep(1 * time.Second)
	return errors.New("nope")
}

func (tp erroringPrinter) Close() error { return nil }

func TestBroadcastOneErrs(t *testing.T) {
	var buf = &bytes.Buffer{}

	sink, bcast := NewEndpointsEmitter()
	defer sink.Close()

	var p1 testPrinter
	p1.w = buf

	var p2 erroringPrinter
	p2.w = buf

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)
	defer closeSink()

	sink.Update([]string{"run1"})

	sink.Update([]string{"run", "2"})

	output := buf.String()

	expectedContains := []string{
		"test: 1\n",
		"failed: 1\n",
		"test: 2\n",
	}

	for i, exp := range expectedContains {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %d in ouput but it didn't", i+1)
			t.Log(output)
		}
	}

	lines := strings.Split(output, "\n")
	if n := len(lines); n != 4 { // 3 and one empty one
		t.Errorf("expected 4 lines of output but got %d", n)
		t.Log(output)
	}
}

/*
type expectedEOSErr struct{ v interface{} }

func (err expectedEOSErr) Error() string {
	return fmt.Sprintf("expected end of stream but got %q", err.v)
}

func (err expectedEOSErr) IsExpectedEOS() bool {
	return true
}

func IsExpectedEOS(err error) bool {
	_, ok := err.(expectedEOSErr)
	return ok
}

func TestBroadcast(t *testing.T) {
	type testcase struct {
		rx, tx []interface{}
	}

	test := func(tc testcase) {
		if tc.rx == nil {
			tc.rx = tc.tx
		}

		sink, bcast := NewEndpointsEmitter()

		mkSink := func() Sink {
			var (
				closed bool
				i      int
			)

			return FuncSink(func(ctx context.Context, v interface{}, err error) error {
				if err != nil {
					if err != (EOS{}) {
						t.Log("closed with non-EOF error:", err)
					}

					if closed {
						return fmt.Errorf("sink already closed")
					}

					if i != len(tc.rx) {
						return fmt.Errorf("early close at i=%v", i)
					}

					closed = true
					return nil
				}

				if i >= len(tc.rx) {
					return expectedEOSErr{v}
				}

				if v != tc.rx[i] {
					return fmt.Errorf("expected value %v but got %v", tc.rx[i], v)
				}

				i++

				return nil
			})
		}

		cancelReg1 := bcast.Register(mkSink())
		cancelReg2 := bcast.Register(mkSink())

		defer cancelReg1()
		defer cancelReg2()

		for j, v := range tc.tx {
			err := sink.Pour(context.TODO(), v)

			if len(tc.tx) == len(tc.rx) {
				if err != nil {
					t.Errorf("expected nil error but got %#v", err)
				}
			} else if len(tc.tx) > len(tc.rx) {
				if j >= len(tc.rx) {
					merr, ok := err.(*multierror.Error)
					if ok {
						for _, err := range merr.Errors {
							if !IsExpectedEOS(err) {
								t.Errorf("expected an expectedEOS error, but got %v", err)
							}
						}
					} else {
						if !IsExpectedEOS(err) {
							t.Errorf("expected an expectedEOS error, but got %v", err)
						}
					}
				} else {
					if err != nil {
						t.Errorf("expected nil error but got %#v", err)
					}
				}
			}
		}

		err := sink.Close()
		if err != nil {
			t.Errorf("expected nil error but got %s", err)
		}
	}

	cases := []testcase{
		{tx: []interface{}{1, 2, 3}},
		{tx: []interface{}{}},
		{tx: []interface{}{nil, 0, ""}},
		{tx: []interface{}{nil, 0, ""}, rx: []interface{}{nil, 0}},
	}

	for _, tc := range cases {
		test(tc)
	}

}
*/
