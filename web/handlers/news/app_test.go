package news

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/gossb-rooms/web"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

var (
	testMux    *http.ServeMux
	testClient *tester.Tester
	testRouter = router.News(nil)

	testAssets = http.Dir("../../templates")
)

func setup(t *testing.T) {
	log, _ := logtest.KitLogger("feed", t)
	r, err := render.New(testAssets, //TODO: embedd web.Assets,
		render.SetLogger(log),
		render.BaseTemplates("/testing/base.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "/error.tmpl")...),
		render.FuncMap(web.TemplateFuncs(testRouter)),
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
