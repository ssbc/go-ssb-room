// Package i18n wraps around github.com/nicksnyder/go-i18n mostly so that we don't have to deal with i18n.LocalizeConfig struct literals everywhere.
package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.mindeco.de/ssb-rooms/internal/repo"
	"golang.org/x/text/language"
)

type Helper struct {
	bundle *i18n.Bundle
}

func New(r repo.Interface) (*Helper, error) {

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// TODO: could additionally embedd the defaults together with the html assets and templates

	err := filepath.Walk(r.GetPath("i18n"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, "toml") {
			return nil
		}

		_, err = bundle.LoadMessageFile(path)
		if err != nil {
			return fmt.Errorf("i18n: failed to parse file %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
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

	// TODO: could panic() and let the http recovery handle this?
	// might also be easier to catch in testing
	return fmt.Sprintf("i18n/error: failed to localize %s: %s", messageID, err)
}

func (l Localizer) LocalizePlurals(messageID string, pluralCount int) string {
	msg, err := l.loc.Localize(&i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: pluralCount,
	})
	if err == nil {
		return msg
	}

	// TODO: could panic() and let the http recovery handle this?
	// might also be easier to catch in testing
	return fmt.Sprintf("i18n/error: failed to localize %s: %s", messageID, err)
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

	// TODO: could panic() and let the http recovery handle this?
	// might also be easier to catch in testing
	return fmt.Sprintf("i18n/error: failed to localize %s: %s", messageID, err)
}
