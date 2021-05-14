// Package webassert contains test helpers to the check the rooms web pages for certain aspects.
package webassert

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mindeco.de/http/tester"
)

type LocalizedElement struct {
	Selector, Label string
}

// Localized checks that a certain selector has a certain label.
// This relies on the fact that the test code doesn't return a localized version but just the labels
func Localized(t *testing.T, html *goquery.Document, elems []LocalizedElement) {
	a := assert.New(t)
	for i, pair := range elems {
		a.Equal(pair.Label, html.Find(pair.Selector).Text(), "localized pair %d failed (selector: %s)", i+1, pair.Selector)
	}
}

// CSRFTokenPresent checks a CSRF token is in side the passed selection (ususally a form).
// The function returns a url.Values map with the token, which can be used to craft further requests.
func CSRFTokenPresent(t *testing.T, sel *goquery.Selection) url.Values {
	a := assert.New(t)
	csrfField := sel.Find("input[name='gorilla.csrf.Token']")
	a.EqualValues(1, csrfField.Length(), "no csrf-token input tag")
	tipe, ok := csrfField.Attr("type")
	a.True(ok, "csrf input has a type")
	a.Equal("hidden", tipe, "wrong type on csrf field")
	val, ok := csrfField.Attr("value")
	a.True(ok, "should have a value")
	return url.Values{
		"gorilla.csrf.Token": []string{val},
	}
}

type FormElement struct {
	Tag, Name, Value, Type, Placeholder string
}

// ElementsInForm checks a list of defined elements. It tries to find them by input[name=$name]
// and then proceeds with asserting their value, type or placeholder (if the fields in FormElement are not "")
func ElementsInForm(t *testing.T, form *goquery.Selection, elems []FormElement) {
	a := assert.New(t)
	for _, e := range elems {
		// empty Tag defaults to <input>
		if e.Tag == "" {
			e.Tag = "input"
		}

		elementSelector := form.Find(fmt.Sprintf("%s[name=%s]", e.Tag, e.Name))
		ok := a.Equal(1, elementSelector.Length(), "expected to find element with name %s", e.Name)
		if !ok {
			continue
		}

		if e.Value != "" {
			value, has := elementSelector.Attr("value")
			a.True(has, "expected value attribute input[name=%s]", e.Name)
			a.Equal(e.Value, value, "wrong value attribute on input[name=%s]", e.Name)
		}

		if e.Type != "" {
			tipe, has := elementSelector.Attr("type")
			a.True(has, "expected type attribute input[name=%s]", e.Name)
			a.Equal(e.Type, tipe, "wrong type attribute on input[name=%s]", e.Name)
		}

		if e.Placeholder != "" {
			tipe, has := elementSelector.Attr("placeholder")
			a.True(has, "expected placeholder attribute input[name=%s]", e.Name)
			a.Equal(e.Placeholder, tipe, "wrong placeholder attribute on input[name=%s]", e.Name)
		}
	}
}

func HasFlashMessages(t *testing.T, client *tester.Tester, url *url.URL, labels ...string) {
	a := assert.New(t)

	doc, resp := client.GetHTML(url)
	a.Equal(http.StatusOK, resp.Code)
	flashes := doc.Find("#flashes-list").Children()
	require.Equal(t, len(labels), flashes.Length(), "number of labels is wrong")

	flashes.Each(func(idx int, sel *goquery.Selection) {
		a.Equal(labels[idx], sel.Text(), "wrong text on label %d", idx)
	})
}
