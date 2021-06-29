package notices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/notices"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
)

func TestDefaults(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	d, err := notices.AllDefaults()
	r.NoError(err)

	var wantNames = []struct {
		Name  roomdb.PinnedNoticeName
		Count int
	}{
		{roomdb.NoticeDescription, 1},
		{roomdb.NoticeNews, 1},
		{roomdb.NoticeCodeOfConduct, 1},
		{roomdb.NoticePrivacyPolicy, 1},
	}
	r.Len(d, len(wantNames))

	for _, want := range wantNames {
		notices, has := d[want.Name]
		a.True(has, "expected %s in defaults", want.Name)
		a.Len(notices.Notices, want.Count)
	}
}
