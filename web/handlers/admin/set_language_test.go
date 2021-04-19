package admin

import (
	"net/http"
  "strings"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)


/* can't test English atm due to web/i18n/i18ntesting/testing.go:justTheKeys, which generates translations that are just
* translationLabel = "translationLabel"
*/
// func TestLanguageSetDefaultLanguageEnglish(t *testing.T) {
// 	ts := newSession(t)
// 	a := assert.New(t)
//
// 	ts.ConfigDB.GetDefaultLanguageReturns("en", nil)
//
// 	u := ts.URLTo(router.AdminSettings)
// 	html, resp := ts.Client.GetHTML(u)
// 	a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
//
//   fmt.Println(html.Html())
// 	summaryElement := html.Find("#language-summary")
//   summaryText := strings.TrimSpace(summaryElement.Text())
//   a.Equal("English", summaryText, "summary language should display english translation of language name")
// }

func TestLanguageSetDefaultLanguage(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	ts.ConfigDB.GetDefaultLanguageReturns("de", nil)

	u := ts.URLTo(router.AdminSettings)
	html, resp := ts.Client.GetHTML(u)
	a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")

	summaryElement := html.Find("#language-summary")
  summaryText := strings.TrimSpace(summaryElement.Text())
  a.Equal("Deutsch", summaryText, "summary language should display german translation of language name")
}
