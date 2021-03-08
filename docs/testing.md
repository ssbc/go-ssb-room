# Testing
* Fill the fake route
* Write your test (preferably getting the route from the router)

## Structure
Most routes are focused on administrating the room server. Tasks such as adding new users, editing notices (like the Welcome page or Code of Conduct). These are routes that require authentication, and they live at `web/handlers/admin`.

Routes that are to be visited by all users can be found in `web/handlers`.

### Places to write tests
* `web/handlers` covers site-functionality usable by all
* `web/handlers/admin` covers admin-only functionality
* `admindb/sqlite` covers tests that are using default data as opposed to a given tests's mockdata

## Goquery
The frontend tests, the tests that check for the presence of various elements on served pages,
use the module [`goquery`](https://github.com/PuerkitoBio/goquery) for querying the returned
HTML.

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


## Functions
#### `web/handlers/admin:newSession(*testing.T)`
Creates a testing session with admin capabilities mocked.
More concretely, the following is mocked:
* databases
    * `ts.AllowListDB`
    * `ts.PinnedDB`
    * `ts.NoticeDB`
* `ts.RoomState`
    * I don't know what this is
* `ts.Router`
    * I don't really know what this does
* `ts.Mux`
* `ts.Client`
    * Testing facility used for performing mocked HTTP requests:  
    `ts.Client.GetHTML(url string)`

## Filling the fakedb
This means prefilling a mock route with the data you expect. Note: This is in
opposition to using an entire pre-filled sqlite database of fake data.

Thus, there is no command you run first to generate your fake database, but
functions you have to call in a kind of pre-test setup, inside each testing
block you are authoring. 

> [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) generates a bunch of methods for each function, so you have
> XXXXReturns,  XXXCallCount XXXArgsForCall(i) etc
>
> _cryptix_

That is, for a function `GetUID` there is a corresponding mock-filling function
`GetUIDReturns`.


## Example test
```
package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoticeShow(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	testNotice := admindb.Notice{
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
