// SPDX-License-Identifier: MIT

package news

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var (
	testMux    *http.ServeMux
	testClient *tester.Tester
	testRouter = router.News(nil)
)

func setup(t *testing.T) {

	testFuncs := web.TemplateFuncs(testRouter)
	testFuncs["i18n"] = func(msgID string) string { return msgID }
	testFuncs["is_logged_in"] = func() *admindb.User { return nil }

	log, _ := logtest.KitLogger("feed", t)
	r, err := render.New(web.Templates,
		render.SetLogger(log),
		render.BaseTemplates("/base.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "/error.tmpl")...),
		render.FuncMap(testFuncs),
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: render init failed"))
	}
	testMux = http.NewServeMux()
	testMux.Handle("/", Handler(testRouter, r))
	testClient = tester.New(testMux, t)
}

func teardown() {
	testMux = nil
	testClient = nil
}
