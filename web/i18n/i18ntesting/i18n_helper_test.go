package i18ntesting

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
)

func TestListLanguages(t *testing.T) {
	configDB := new(mockdb.FakeRoomConfig)
	configDB.GetDefaultLanguageReturns("en", nil)
	r := repo.New(filepath.Join("testrun", t.Name()))
	a := assert.New(t)
	helper, err := i18n.New(r, configDB)
	a.NoError(err)
	t.Log(helper)
	translation := helper.ChooseTranslation("en")
	a.Equal(translation, "English")
}
