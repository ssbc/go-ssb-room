// SPDX-License-Identifier: MIT

package multicloser

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type Closer struct {
	cs []io.Closer
	l  sync.Mutex
}

func (mc *Closer) Add(c io.Closer) {
	mc.l.Lock()
	defer mc.l.Unlock()

	mc.cs = append(mc.cs, c)
}

var _ io.Closer = (*Closer)(nil)

func (mc *Closer) Close() error {
	mc.l.Lock()
	defer mc.l.Unlock()

	var (
		hasErrs bool
		errs    []error
	)

	for i, c := range mc.cs {
		if cerr := c.Close(); cerr != nil {
			cerr = fmt.Errorf("Closer: c%d failed: %w", i, cerr)
			errs = append(errs, cerr)
			hasErrs = true
		}
	}

	if !hasErrs {
		return nil
	}

	return errList{errs: errs}
}

type errList struct {
	errs []error
}

func (el errList) Error() string {
	var str strings.Builder

	if n := len(el.errs); n > 0 {
		fmt.Fprintf(&str, "multiple errors(%d): ", n)
	}
	for i, err := range el.errs {
		fmt.Fprintf(&str, "(%d): ", i)
		str.WriteString(err.Error() + " - ")
	}

	return str.String()
}
