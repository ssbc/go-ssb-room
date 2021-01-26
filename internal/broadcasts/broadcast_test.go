// SPDX-License-Identifier: MIT

package broadcasts

import (
	"errors"
	"fmt"
)

type testPrinter struct{}

func (tp testPrinter) Update(rc RoomChange) error {
	fmt.Println("test:", rc.Op, rc.Who.Ref())
	return nil
}

func (tp testPrinter) Close() error { return nil }

func ExampleBroadcast() {
	sink, bcast := NewRoomChanger()
	defer sink.Close()

	var p1, p2 testPrinter

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)
	defer closeSink()

	var rc RoomChange
	rc.Who.Algo = "dummy"
	rc.Who.ID = []byte{0, 0, 0, 0}
	rc.Op = "joined"
	sink.Update(rc)

	// Output:
	// test: joined @AAAAAA==.dummy
	// test: joined @AAAAAA==.dummy
}

func ExampleBroadcastCanceled() {
	sink, bcast := NewRoomChanger()
	defer sink.Close()

	var p1, p2 testPrinter

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)

	var rc RoomChange
	rc.Who.Algo = "dummy"
	rc.Who.ID = []byte{0, 0, 0, 0}
	rc.Op = "joined"

	closeSink()

	sink.Update(rc)

	// Output:
	// test: joined @AAAAAA==.dummy
}

type erroringPrinter struct{}

func (tp erroringPrinter) Update(rc RoomChange) error {
	fmt.Println("failed:", rc.Op, rc.Who.Ref())
	return errors.New("nope")
}

func (tp erroringPrinter) Close() error { return nil }

func ExampleBroadcastOneErrs() {
	sink, bcast := NewRoomChanger()
	defer sink.Close()

	var p1 testPrinter
	var p2 erroringPrinter

	closeSink := bcast.Register(p1)
	defer closeSink()
	closeSink = bcast.Register(p2)
	defer closeSink()

	var rc RoomChange
	rc.Who.Algo = "dummy"
	rc.Who.ID = []byte{0, 0, 0, 0}
	rc.Op = "joined"

	sink.Update(rc)

	rc.Op = "left"
	sink.Update(rc)

	// Output:
	// test: joined @AAAAAA==.dummy
	// failed: joined @AAAAAA==.dummy
	// test: left @AAAAAA==.dummy
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

		sink, bcast := NewRoomChanger()

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
