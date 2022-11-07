// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/router"
)

func TestSettingsOverview(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	/* First: make sure everything renders correctly for admins */
	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}

	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeCommunity, nil)
	ts.ConfigDB.GetDefaultLanguageReturns("en", nil)

	settingsURL := ts.URLTo(router.AdminSettings)

	html, resp := ts.Client.GetHTML(settingsURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	// the privacy mode form & its summary/details container should exist
	privacyFormContainer := html.Find("#change-privacy")
	a.Equal(1, privacyFormContainer.Length())
	a.Equal(1, privacyFormContainer.Find("summary").Length())
	// chosen privacy mode is ModeCommunity (english translation will only be the name of the label, due to testing suite is set up atm)
	a.Equal("ModeCommunity", strings.TrimSpace(privacyFormContainer.Find("summary").Text()))
	// details-dropdown should have two forms, one for each of the other two privacy modes
	// that can be selected (ModeOpen, ModeRestricted)
	a.Equal(2, privacyFormContainer.Find("form").Length())
	// and one span, showing the selected mode
	a.Equal(1, privacyFormContainer.Find("#selected-mode").Length())
	inputs := privacyFormContainer.Find("input")
	// verify none of the privacy mode container's inputs are disabled
	inputs.Each(func(i int, el *goquery.Selection) {
		_, exists := el.Attr("disabled")
		a.False(exists)
	})

	// verify that the change language form exists & is enabled
	languageFormContainer := html.Find("#change-language-container")
	a.Equal(1, languageFormContainer.Length())
	a.Equal(1, languageFormContainer.Find("summary").Length())
	// (english translation will only be the name of the label, due to testing suite is set up atm)
	a.Equal("LanguageName", strings.TrimSpace(languageFormContainer.Find("summary").Text()))

	testDisabledBehaviour := func() {
		settingsURL := ts.URLTo(router.AdminSettings)
		html, resp := ts.Client.GetHTML(settingsURL)
		a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

		// we do not have the summary/details hack if the forms are hidden
		privacyFormContainer := html.Find("#change-privacy")
		a.Equal(0, privacyFormContainer.Length())
		// the should still be the parent container, however
		privacyContainer := html.Find("#privacy-mode-container")
		a.Equal(1, privacyContainer.Length())
		// there should only be one input in the privacy mode container now
		inputs := privacyContainer.Find("input")
		a.Equal(1, inputs.Length())
		// the input should be disabled
		_, disabled := inputs.Attr("disabled")
		a.True(disabled)

		// next, verify that the change language setting is disabled
		languageContainer := html.Find("#change-language-container")
		a.Equal(1, languageContainer.Length())
		// there should only be one input in the language mode container now
		inputs = languageContainer.Find("input")
		a.Equal(1, inputs.Length())
		// the input should be disabled
		_, disabled = inputs.Attr("disabled")
		a.True(disabled)
	}

	/* Now: verify that moderators cannot make room settings changes */
	ts.User = roomdb.Member{
		ID:   7331,
		Role: roomdb.RoleModerator,
	}
	testDisabledBehaviour()

	/* Finally: verify that members cannot make room settings changes */
	ts.User = roomdb.Member{
		ID:   9001,
		Role: roomdb.RoleMember,
	}
	testDisabledBehaviour()
}
