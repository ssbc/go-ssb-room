// SPDX-License-Identifier: MIT

package testutils

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log/term"
)

func NewRelativeTimeLogger(w io.Writer) log.Logger {
	if w == nil {
		w = log.NewSyncWriter(os.Stderr)
	}

	var rtl relTimeLogger
	rtl.start = time.Now()

	//	mainLog := log.NewLogfmtLogger(w)
	mainLog := term.NewColorLogger(w, log.NewLogfmtLogger, colorFn)
	return log.With(mainLog, "t", log.Valuer(rtl.diffTime))
}

func colorFn(keyvals ...interface{}) term.FgBgColor {
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok && key == "level" {
			lvl, ok := keyvals[i+1].(level.Value)
			if !ok {
				fmt.Printf("%d: %v %T\n", i+1, lvl, keyvals[i+1])
				continue
			}

			var c term.FgBgColor
			level := lvl.String()
			switch level {
			case "error":
				c.Fg = term.Red
			case "warn":
				c.Fg = term.Brown
			case "debug":
				c.Fg = term.Gray
			case "info":
				c.Fg = term.Green
			default:
				panic("unhandled level:" + level)
			}
			return c
		}
	}
	return term.FgBgColor{}
}

type relTimeLogger struct {
	sync.Mutex

	start time.Time
}

func (rtl *relTimeLogger) diffTime() interface{} {
	rtl.Lock()
	defer rtl.Unlock()
	newStart := time.Now()
	since := newStart.Sub(rtl.start)
	return since
}
