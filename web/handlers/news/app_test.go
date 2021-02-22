// SPDX-License-Identifier: MIT

package news

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

type testSession struct {
	Mux    *http.ServeMux
	Client *tester.Tester
	Router *mux.Router
}

func newSession(t *testing.T) *testSession {
	var ts testSession
	ts.Router = router.News(nil)

	testFuncs := web.TemplateFuncs(ts.Router)
	testFuncs["i18n"] = func(msgID string) string { return msgID }
	testFuncs["is_logged_in"] = func() *admindb.User { return nil }

	log, _ := logtest.KitLogger("feed", t)
	r, err := render.New(web.Templates,
		render.SetLogger(log),
		render.BaseTemplates("base.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "error.tmpl")...),
		render.FuncMap(testFuncs),
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: render init failed"))
	}
	ts.Mux = http.NewServeMux()
	ts.Mux.Handle("/", Handler(ts.Router, r))
	ts.Client = tester.New(ts.Mux, t)

	return &ts
}
