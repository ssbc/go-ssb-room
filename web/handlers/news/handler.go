// SPDX-License-Identifier: MIT

package news

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

type Post struct {
	ID         int
	Name, Text string
}

var db = []Post{
	Post{
		ID:   0,
		Name: "Hello",
		Text: "lot's of stuff",
	},
	Post{
		ID:   1,
		Name: "Testing",
		Text: "yeeeeaaaahhhh...",
	},
	Post{
		ID:   2,
		Name: "WAT",
		Text: "i have only a partial idea of what i'm doing",
	},
}

func showOverview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	return map[string]interface{}{
		"AllPosts": db,
	}, nil
}

func showPost(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	i, err := strconv.Atoi(req.URL.Query().Get("id"))
	if err != nil {
		return nil, fmt.Errorf("argument parsing failed: %w", err)
	}
	if i < 0 || i >= len(db) {
		return nil, errors.New("db: not found")
	}
	return map[string]interface{}{
		"Post": db[i],
	}, nil
}
