package i18ntesting

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
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
