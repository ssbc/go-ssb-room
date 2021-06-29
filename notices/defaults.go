package notices

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/cryptix/front"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
)

//go:embed defaults/*.md
var Defaults embed.FS

type frontMatter struct {
	Title    string
	Notice   roomdb.PinnedNoticeName
	Language string
}

type NoticesMap map[roomdb.PinnedNoticeName]roomdb.PinnedNotice

func AllDefaults() (NoticesMap, error) {
	var notices = make(NoticesMap)

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error from walk %q: %w", path, err)
		}

		if d.IsDir() {
			return nil
		}

		r, err := Defaults.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open: %q: %w", path, err)
		}

		m := front.NewMatter("+++")

		var front frontMatter
		body, err := m.JSONViaPointer(r, &front)
		if err != nil {
			return err
		}

		n, has := notices[front.Notice]
		if !has {
			n = roomdb.PinnedNotice{
				Name: front.Notice,
			}
		}

		n.Notices = append(n.Notices, roomdb.Notice{
			Title:    front.Title,
			Content:  body,
			Language: front.Language,
		})

		notices[front.Notice] = n

		return nil
	}

	err := fs.WalkDir(Defaults, ".", walkFn)
	if err != nil {
		return nil, err
	}

	return notices, nil
}
