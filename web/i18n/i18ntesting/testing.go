// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package i18ntesting

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
)

// justTheKeys auto generates from the defaults a list of Label = "Label"
// must keep order of input intact
// (at least all the globals before starting with nested plurals)
// also replaces 'one' and 'other' in plurals
func justTheKeys(t *testing.T) []byte {
	f, err := i18n.Defaults.Open("active.en.toml")
	if err != nil {
		t.Fatal(err)
	}
	justAMap := make(map[string]interface{})
	md, err := toml.DecodeReader(f, &justAMap)
	if err != nil {
		t.Fatal(err)
	}

	var buf = &bytes.Buffer{}

	// if we don't produce the same order as the input
	// (in go maps are ALWAYS random access when ranged over)
	// nested keys (such as plural form) will mess up the global level...
	for _, k := range md.Keys() {
		key := k.String()
		val, has := justAMap[key]
		if !has {
			// fmt.Println("i18n test warning:", key, "not unmarshaled")
			continue
		}

		switch tv := val.(type) {

		case string:
			fmt.Fprintf(buf, "%s = \"%s\"\n", key, key)

		case map[string]interface{}:
			// fmt.Println("i18n test warning: custom map for ", key)

			fmt.Fprintf(buf, "\n[%s]\n", key)
			// replace "one" and "other" keys
			// with  Label and LabelPlural
			tv["one"] = key + "Singular"
			tv["other"] = key + "Plural"
			toml.NewEncoder(buf).Encode(tv)
			fmt.Fprintln(buf)

		default:
			t.Fatalf("unhandled toml structure under %s: %T\n", key, val)
		}
	}

	return buf.Bytes()
}

func WriteReplacement(t *testing.T) {
	r := repo.New(filepath.Join("testrun", t.Name()))

	testOverride := filepath.Join(r.GetPath("i18n"), "active.en.toml")
	t.Log(testOverride)
	os.MkdirAll(filepath.Dir(testOverride), 0700)

	content := justTheKeys(t)

	err := ioutil.WriteFile(testOverride, content, 0700)
	if err != nil {
		t.Fatal(err)
	}
}
