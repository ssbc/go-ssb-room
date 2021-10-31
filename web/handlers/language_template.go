// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import "html/template"

type changeLanguageTemplateData struct {
	PostRoute    string
	CSRFElement  template.HTML
	LangTag      string
	RedirectPage string
	Translation  string
	ClassList    string
}

var changeLanguageTemplate = template.Must(template.New("changeLanguageForm").Parse(`
<form
  action="{{ .PostRoute }}"
  method="POST"
>
  {{ .CSRFElement }}
  <input type="hidden" name="lang" value="{{ .LangTag }}">
  <input type="hidden" name="page" value="{{ .RedirectPage }}">
  <input
    type="submit"
    value="{{ .Translation }}"
    class="{{ .ClassList }}"
  />
</form>`))
