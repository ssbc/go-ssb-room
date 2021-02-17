// SPDX-License-Identifier: MIT

// Package i18n wraps around github.com/nicksnyder/go-i18n mostly so that we don't have to deal with i18n.LocalizeConfig struct literals everywhere.
package i18n

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/shurcooL/httpfs/vfsutil"
	"golang.org/x/text/language"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

//go:generate go run -tags=dev defaults_generate.go

type Helper struct {
	bundle *i18n.Bundle
}

func New(r repo.Interface) (*Helper, error) {

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// parse toml files and add them to the bundle
	walkFn := func(path string, info os.FileInfo, rs io.ReadSeeker, err error) error {
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
	err := vfsutil.WalkFiles(Defaults, "/", walkFn)
	if err != nil { // && !os.IsNotExist(err) {
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

		rs, err := os.Open(path)
		if err != nil {
			return err
		}
		defer rs.Close()

		err = walkFn(path, info, rs, err)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("i18n: failed to iterate localizations: %w", err)
	}

	return &Helper{bundle: bundle}, nil
}

type Localizer struct {
	loc *i18n.Localizer
}

func (h Helper) NewLocalizer(lang string, accept ...string) *Localizer {
	var langs = []string{lang}
	langs = append(langs, accept...)
	var l Localizer
	l.loc = i18n.NewLocalizer(h.bundle, langs...)
	return &l
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

// LocalizerFromRequest returns a new Localizer for the passed helper,
// using form value 'lang' and Accept-Language http header from the passed request.
// TODO: user settings/cookie values?
func LocalizerFromRequest(helper *Helper, r *http.Request) *Localizer {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")
	return helper.NewLocalizer(lang, accept)
}
