// SPDX-License-Identifier: MIT

package alias

import (
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	kitlog "github.com/go-kit/kit/log"

	refs "go.mindeco.de/ssb-refs"
)

func New(log kitlog.Logger, self refs.FeedRef, aliasDB roomdb.AliasService) Handler {
	var h Handler
	h.self = self
	h.logger = log
	h.db = aliasDB

	return h
}
