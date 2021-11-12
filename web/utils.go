// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package web

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	ua "github.com/mileusna/useragent"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

// TemplateFuncs returns a map of template functions
func TemplateFuncs(m *mux.Router, netInfo network.ServerEndpointDetails) template.FuncMap {
	return template.FuncMap{
		"human_time": func(when time.Time) string {
			return humanize.Time(when)
		},
		"urlTo": NewURLTo(m, netInfo),
		"inc":   func(i int) int { return i + 1 },
	}
}

type URLMaker func(string, ...interface{}) *url.URL

// NewURLTo returns a template helper function for a router.
// It is usually called with one parameter, the route name, which should be defined in the router package.
// If it's called with more then one, it has a to be a pair of two values. (1, 3, 5, 7, etc.)
// The first value of such a pair is the placeholder name in the router (i.e. in '/our/routes/{id:[0-9]+}/test' it would be id )
// and the 2nd value is the actual value that should be put in place of the placeholder.
func NewURLTo(appRouter *mux.Router, netInfo network.ServerEndpointDetails) URLMaker {
	l := logging.Logger("helper.URLTo") // TOOD: inject in a scoped way
	return func(routeName string, ps ...interface{}) *url.URL {
		route := appRouter.Get(routeName)
		if route == nil {
			// TODO: https://github.com/ssb-ngi-pointer/go-ssb-room/issues/35 for a
			// for reference, see https://github.com/ssb-ngi-pointer/go-ssb-room/pull/64
			// level.Warn(l).Log("msg", "no such route", "route", routeName, "params", fmt.Sprintf("%v", ps))
			return &url.URL{}
		}

		if len(ps)%2 != 0 {
			level.Warn(l).Log("msg", "expected even number of params (name-value pairs)", "route", routeName, "params", fmt.Sprintf("%v", ps))
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
			case refs.FeedRef:
				params = append(params, v.Ref())
			case roomdb.PinnedNoticeName:
				params = append(params, string(v))
			default:
				level.Error(l).Log("msg", "invalid param type", "param", fmt.Sprintf("%T", p), "route", routeName)
				return &url.URL{}
			}
		}

		// named vars in routes don't work because we cant use the mux.router with middleware correctly
		u, err := route.URLPath()
		if err != nil {
			level.Error(l).Log("msg", "failed to create URL",
				"route", routeName,
				"params", params,
				"error", err)
			return &url.URL{}
		}

		urlVals := u.Query()
		n := len(params)
		for i := 0; i < n; i += 2 {
			key, value := strings.ToLower(params[i]), params[i+1]
			urlVals.Set(key, value)
		}

		u.RawQuery = urlVals.Encode()
		u.Host = netInfo.Domain
		u.Scheme = "https"
		if netInfo.Development {
			u.Scheme = "http"
			u.Host += fmt.Sprintf(":%d", netInfo.PortHTTPS)
		}

		return u
	}
}

// LoadOrCreateCookieSecrets either parses the bytes from $repo/web/cookie-secret or creates a new file with suitable keys in it
func LoadOrCreateCookieSecrets(repo repo.Interface) ([]securecookie.Codec, error) {
	secretPath := repo.GetPath("web", "cookie-secret")
	err := os.MkdirAll(filepath.Dir(secretPath), 0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create folder for cookie secret: %w", err)
	}

	// load the existing data
	secrets, err := ioutil.ReadFile(secretPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load cookie secrets: %w", err)
		}

		// create new keys, save them and return the codec
		hashKey := securecookie.GenerateRandomKey(32)
		blockKey := securecookie.GenerateRandomKey(32)

		data := append(hashKey, blockKey...)
		err = ioutil.WriteFile(secretPath, data, 0600)
		if err != nil {
			return nil, err
		}
		sc := securecookie.CodecsFromPairs(hashKey, blockKey)
		return sc, nil
	}

	// secrets should contain multiple of 64byte (to enable key rotation as supported by gorilla)
	if n := len(secrets); n%64 != 0 {
		return nil, fmt.Errorf("expected multiple of 64bytes in cookie secret file but got: %d", n)
	}

	// range over the secrets []byte in chunks of 64 bytes
	// and slice it into 32byte pairs
	var pairs [][]byte

	// the increment/next part (which usually is i++)
	// is the multiple comma assigment (a,b = b+1,a-1)
	// so chunk is the next 64 bytes and then it slices of the first 64 bytes of secrets for the next iteration
	for chunk := secrets[:64]; len(secrets) >= 64; chunk, secrets = secrets[:64], secrets[64:] {
		pairs = append(pairs,
			chunk[0:32],  // hash key
			chunk[32:64], // block key
		)
	}

	sc := securecookie.CodecsFromPairs(pairs...)
	return sc, nil
}

const csrfKeyLength = 32

// LoadOrCreateCSRFSecret either loads the bytes from $repo/web/csrf-secret or creates a new file with suitable keys in it
func LoadOrCreateCSRFSecret(repo repo.Interface) ([]byte, error) {
	secretPath := repo.GetPath("web", "csrf-secret")
	err := os.MkdirAll(filepath.Dir(secretPath), 0700)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create folder for csrf secret: %w", err)
	}

	// load the existing data
	secret, err := ioutil.ReadFile(secretPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load csrf secrets: %w", err)
		}

		// create a new key, save and return it
		freshKey := securecookie.GenerateRandomKey(csrfKeyLength)

		err = ioutil.WriteFile(secretPath, freshKey, 0600)
		if err != nil {
			return nil, err
		}

		return freshKey, nil
	}

	if n := len(secret); csrfKeyLength != n {
		return nil, fmt.Errorf("expected %d bytes csrf secert but got %d", csrfKeyLength, n)
	}

	return secret, nil
}

// Transforms the SSB URI into a string suitable for an <a href> and
// takes into account quirks from some browsers.
func StringifySSBURI(uri *url.URL, userAgent string) string {
	browser := ua.Parse(userAgent)

	// Special treatment for Android Chrome for issue #135
	// https://github.com/ssb-ngi-pointer/go-ssb-room/issues/135
	if browser.IsAndroid() && browser.IsChrome() {
		href := url.URL{
			Scheme:   "intent",
			Opaque:   fmt.Sprintf("//%v", uri.Opaque),
			RawQuery: uri.RawQuery,
			Fragment: fmt.Sprintf("Intent;scheme=%v;end;", uri.Scheme),
		}
		return href.String()
	} else {
		return uri.String()
	}
}
