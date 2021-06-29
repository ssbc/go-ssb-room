package notices

import (
	"context"
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

func InitDefaults(pinnedDB roomdb.PinnedNoticesService, noticesDB roomdb.NoticesService) error {
	ctx := context.Background()

	existingList, err := pinnedDB.List(ctx)
	if err != nil {
		return err
	}

	if len(existingList) != 0 {
		return fmt.Errorf("expected an empty notices list")
	}

	def, err := AllDefaults()
	if err != nil {
		return err
	}

	for name, notices := range def {
		for _, notice := range notices.Notices {
			err = noticesDB.Save(ctx, &notice)
			if err != nil {
				return fmt.Errorf("InitDefaults: failed to save notice for %s: %w", name, err)
			}

			err = pinnedDB.Set(ctx, name, notice.ID)
			if err != nil {
				return fmt.Errorf("InitDefaults: failed to set pin for %s: %w", name, err)
			}
		}
	}

	return nil
}
