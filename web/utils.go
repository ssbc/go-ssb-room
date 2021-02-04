package web

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"go.mindeco.de/logging"
)

func NewURLTo(appRouter *mux.Router) func(string, ...interface{}) *url.URL {
	l := logging.Logger("helper.URLTo") // TOOD: inject in a scoped way
	return func(routeName string, ps ...interface{}) *url.URL {
		route := appRouter.Get(routeName)
		if route == nil {
			level.Warn(l).Log("msg", "no such route", "route", routeName, "params", ps)
			return &url.URL{}
		}

		var params []string
		for _, p := range ps {
			switch v := p.(type) {
			case string:
				params = append(params, v)
			case int:
				params = append(params, strconv.Itoa(v))
			case int64:
				params = append(params, strconv.FormatInt(v, 10))

			default:
				level.Error(l).Log("msg", "invalid param type", "param", p, "route", routeName)
				logging.CheckFatal(errors.New("invalid param"))
			}
		}

		u, err := route.URLPath(params...)
		if err != nil {
			level.Error(l).Log("msg", "failed to create URL",
				"route", routeName,
				"params", params,
				"error", err)
			return &url.URL{}
		}
		return u
	}
}
