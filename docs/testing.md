<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# Testing
`go-ssb-room` has a variety of tests to ensure that functionality that once worked, keeps
working (it does not regress.) These tests are scattered around the repositories modules, but
they are always contained in a file ending with `_test.go`—a compiler-enforced naming
convention for Golang tests.

## Running all tests

You'll need to have Node.js v14 installed before running this.

```
go test ./...
```

This command can be run at the root of the repository.

## Structure
Most routes are focused on administrating the room server. Tasks such as adding new users,
editing notices (like the _Welcome_ or _Code of Conduct_ pages). These are routes that require
elevated privileges to perform actions, and they live in `web/handlers/admin`.

Routes that are to be visited by all users can be found in `web/handlers`.

### Places to write tests
* `web/handlers` covers site-functionality usable by all
* `web/handlers/admin` covers admin-only functionality
* `roomdb/sqlite` covers tests that are using default data as opposed to a given tests's mockdata

## Goquery
The frontend tests—tests that check for the presence of various elements on served pages—use
the module [`goquery`](https://github.com/PuerkitoBio/goquery) for querying the returned HTML.

## Snippets

#### Print the raw html of the corresponding page

```
html, _ := ts.Client.GetHTML(url)
fmt.Println(html.Html())
```

#### Find and print the `title` element of a page

```
html, _ := ts.Client.GetHTML(url)
title := html.Find("title")
// print the title string
fmt.Println(title.Text())
```

## Filling the mockdb

`go-ssb-room` uses database mocks for performing tests against the backend database logic. This
means prefilling a route with the data you expect to be returned when the route is queried.
This type of testing is an alternative to using an entire pre-filled sqlite database of test
data.

As such, there is no command you run first to generate your fake database, but
functions you have to call in a kind of pre-test setup, inside each testing
block you are authoring.

> [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) generates a bunch of methods for each function, so you have
> XXXXReturns,  XXXCallCount XXXArgsForCall(i) etc
>
> _cryptix_

That is, for a function `GetUID` there is a corresponding mock-filling function
`GetUIDReturns`.

The following examples show more concretely what mocking the data looks like.

**Having the List() function return a static list of three items:**

```go
// go-ssb-room/web/handlers/admin/allow_list_test.go:113
lst := roomdb.ListEntries{
	{ID: 1, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
	{ID: 2, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
	{ID: 3, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
}
ts.MembersDB.ListReturns(lst, nil)
```

**Checking how often RemoveID was called and with what arguments:**

```go
// go-ssb-room/web/handlers/admin/allow_list_test.go:210
 a.Equal(1, ts.MembersDB.RemoveIDCallCount())
 _, theID := ts.MembersDB.RemoveIDArgsForCall(0)
 a.EqualValues(666, theID)
```

## Example test

```go
package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoticeShow(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	testNotice := roomdb.Notice{
		ID:    123,
		Title: "foo",
	}
	ts.NoticeDB.GetByIDReturns(testNotice, nil)

	html, resp := ts.Client.GetHTML("/notice/show?id=123")
	a.Equal(http.StatusOK, resp.Code)

	r.Equal("foo", html.Find("title").Text())
	fmt.Println(html.Text())
}
```

## Muxrpc room functionality

### Go

The folder `tests/nodejs` contains basic tests of the client<>server functionality for the different aspects of a room.

```bash
cd muxrpc/test/go
go test
```

### JavaScript

The folder `tests/nodejs` contains tests for the JavaScript implementation. To run them, install node and npm and run the following:

```bash
cd muxrpc/test/nodejs
npm ci
go test
```
