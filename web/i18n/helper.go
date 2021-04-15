// SPDX-License-Identifier: MIT

// Package i18n wraps around github.com/nicksnyder/go-i18n mostly so that we don't have to deal with i18n.LocalizeConfig struct literals everywhere.
package i18n

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"go.mindeco.de/http/render"
	"golang.org/x/text/language"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

const LanguageCookieName = "gossbroom-language"

type Helper struct {
	bundle      *i18n.Bundle
	languages   map[string]string
	cookieStore *sessions.CookieStore
}

func New(r repo.Interface) (*Helper, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	cookieCodec, err := web.LoadOrCreateCookieSecrets(r)
	if err != nil {
		return nil, err
	}

	cookieStore := &sessions.CookieStore{
		Codecs: cookieCodec,
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 2 * 60 * 60, // two hours in seconds  // TODO: configure
		},
	}

	// parse toml files and add them to the bundle
	walkFn := func(path string, info os.FileInfo, rs io.Reader, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, "toml") {
			return nil
		}

		mfb, err := ioutil.ReadAll(rs)
		if err != nil {
			return err
		}
		_, err = bundle.ParseMessageFileBytes(mfb, path)
		if err != nil {
			return fmt.Errorf("i18n: failed to parse file %s: %w", path, err)
		}
		fmt.Println("loaded", path)
		return nil
	}

	// walk the embedded defaults
	err = fs.WalkDir(Defaults, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		r, err := Defaults.Open(path)
		if err != nil {
			return err
		}

		err = walkFn(path, info, r, err)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("i18n: failed to iterate localizations: %w", err)
	}

	// walk the local filesystem for overrides and additions
	err = filepath.Walk(r.GetPath("i18n"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()

		err = walkFn(path, info, r, err)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("i18n: failed to iterate localizations: %w", err)
	}

	// create a mapping of language tags to the translated language names
	langmap := listLanguages(bundle)
	return &Helper{bundle: bundle, languages: langmap, cookieStore: cookieStore}, nil
}

func listLanguages(bundle *i18n.Bundle) map[string]string {
	langmap := make(map[string]string)

	for _, langTag := range bundle.LanguageTags() {
		var l Localizer
		l.loc = i18n.NewLocalizer(bundle, langTag.String())

		msg, err := l.loc.Localize(&i18n.LocalizeConfig{
			MessageID: "LanguageName",
		})
		if err != nil {
			msg = langTag.String()
		}

		langmap[langTag.String()] = msg
	}

	return langmap
}

// ListLanguages returns a mapping between the room's translated languages.
// The keys are language tags (as strings) and the values are the name of the language tag, as translated in the original language.
// For example: en -> English, sv -> Svenska, de -> Deutsch
func (h Helper) ListLanguages() map[string]string {
	return h.languages
}

type Localizer struct {
	loc *i18n.Localizer
}

func (h Helper) newLocalizer(lang string, accept ...string) *Localizer {
	var langs = []string{lang}
	langs = append(langs, accept...)
	var l Localizer
	l.loc = i18n.NewLocalizer(h.bundle, langs...)
	return &l
}

// FromRequest returns a new Localizer for the passed helper,
// using form value 'lang' and Accept-Language http header from the passed request.
// If a language cookie is detected, then it takes precedence over the form value & Accept-Lanuage header.
func (h Helper) FromRequest(r *http.Request) *Localizer {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")

	session, err := h.cookieStore.Get(r, LanguageCookieName)
	if err != nil {
		return h.newLocalizer(lang, accept)
	}

	prevCookie := session.Values["lang"]
	if prevCookie != nil {
		return h.newLocalizer(prevCookie.(string), lang, accept)
	}

	return h.newLocalizer(lang, accept)
}

func (h Helper) GetRenderFuncs() []render.Option {
	var opts = []render.Option{
		render.InjectTemplateFunc("i18npl", func(r *http.Request) interface{} {
			loc := h.FromRequest(r)
			return loc.LocalizePlurals
		}),

		render.InjectTemplateFunc("i18n", func(r *http.Request) interface{} {
			loc := h.FromRequest(r)
			return loc.LocalizeSimple
		}),
	}
	return opts
}

func (l Localizer) LocalizeSimple(messageID string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizeWithData(messageID string, tplData map[string]string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: tplData,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePlurals(messageID string, pluralCount int) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: pluralCount,
		TemplateData: map[string]int{
			"Count": pluralCount,
		},
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePluralsWithData(messageID string, pluralCount int, tplData map[string]string) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  pluralCount,
		TemplateData: tplData,
	})
	if err == nil {
		return msg
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}
