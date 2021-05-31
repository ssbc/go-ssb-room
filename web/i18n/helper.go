// SPDX-License-Identifier: MIT

// Package i18n wraps around github.com/nicksnyder/go-i18n mostly so that we don't have to deal with i18n.LocalizeConfig struct literals everywhere.
package i18n

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	"go.mindeco.de/http/render"
	"golang.org/x/text/language"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
)

const LanguageCookieName = "gossbroom-language"

type TagTranslation struct {
	Tag         string
	Translation string
}

type Helper struct {
	bundle      *i18n.Bundle
	languages   []TagTranslation
	cookieStore *sessions.CookieStore
	config      roomdb.RoomConfig
}

func New(r repo.Interface, config roomdb.RoomConfig) (*Helper, error) {
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
	return &Helper{bundle: bundle, languages: langmap, cookieStore: cookieStore, config: config}, nil
}

func listLanguages(bundle *i18n.Bundle) []TagTranslation {
	languageTags := bundle.LanguageTags()
	tags := make([]string, 0, len(languageTags))
	langslice := make([]TagTranslation, 0, len(languageTags))

	// convert from i18n language tags to a slice of strings
	for _, langTag := range languageTags {
		tags = append(tags, langTag.String())
	}
	// sort the slice of language tag strings
	sort.Strings(tags)

	// now that we have a known order, construct a TagTranslation slice mapping language tags to their translations
	for _, langTag := range tags {
		var l Localizer
		l.loc = i18n.NewLocalizer(bundle, langTag)

		msg, err := l.loc.Localize(&i18n.LocalizeConfig{
			MessageID: "LanguageName",
		})
		if err != nil {
			msg = langTag
		}

		langslice = append(langslice, TagTranslation{Tag: langTag, Translation: msg})
	}

	return langslice
}

// ListLanguages returns a slice of the room's translated languages.
// The entries of the slice are of the type TagTranslation, consisting of the fields Tag and Translation.
// Each Tag fields is a language tag (as strings), and the field Translation is the corresponding translated language
// name of that language tag.
// Example: {Tag: en, Translation: English}, {Tag: sv, Translation: Svenska} {Tag: de, Translation: Deutsch}
func (h Helper) ListLanguages() []TagTranslation {
	return h.languages
}

func (h Helper) ChooseTranslation(requestedTag string) string {
	for _, entry := range h.languages {
		if entry.Tag == requestedTag {
			return entry.Translation
		}
	}
	return requestedTag
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

	defaultLang, err := h.config.GetDefaultLanguage(r.Context())

	// if we don't have a default language set, then fallback to whatever we have left :^)
	if err != nil {
		return h.newLocalizer(lang, accept)
	}

	// if we don't have a user cookie set, then use the room's default language setting
	return h.newLocalizer(defaultLang, accept)
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

		render.InjectTemplateFunc("i18nWithData", func(r *http.Request) interface{} {
			loc := h.FromRequest(r)
			return loc.LocalizeWithData
		}),
	}
	return opts
}

func (l Localizer) LocalizeSimple(messageID string) template.HTML {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err == nil {
		return template.HTML(msg)
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizeWithData(messageID string, labelsAndData ...string) template.HTML {
	n := len(labelsAndData)
	if n%2 != 0 {
		panic(fmt.Errorf("expected an even amount of labels and data. got %d", n))
	}

	tplData := make(map[string]string, n/2)
	for i := 0; i < n; i += 2 {
		key := labelsAndData[i]
		data := labelsAndData[i+1]
		tplData[key] = data
	}

	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: tplData,
	})
	if err == nil {
		return template.HTML(msg)
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePlurals(messageID string, pluralCount int) template.HTML {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: pluralCount,
		TemplateData: map[string]int{
			"Count": pluralCount,
		},
	})
	if err == nil {
		return template.HTML(msg)
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}

func (l Localizer) LocalizePluralsWithData(messageID string, pluralCount int, tplData map[string]string) template.HTML {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  pluralCount,
		TemplateData: tplData,
	})
	if err == nil {
		return template.HTML(msg)
	}

	panic(fmt.Sprintf("i18n/error: failed to localize label %s: %s", messageID, err))
}
