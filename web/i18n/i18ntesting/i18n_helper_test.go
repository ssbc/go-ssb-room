package i18ntesting

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
)

func TestListAllLanguages(t *testing.T) {
	r := repo.New(filepath.Join("testrun", t.Name()))
	a := assert.New(t)
	helper, err := i18n.New(r)
	a.NoError(err)
	t.Log(helper)
	langmap := helper.ListLanguages()
	a.Equal(langmap["en"], "English")
}
