module github.com/ssb-ngi-pointer/go-ssb-room

go 1.15

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/PuerkitoBio/goquery v1.5.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-kit/kit v0.10.0
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gorilla/websocket v1.4.2
	github.com/keks/nocomment v0.0.0-20181007001506-30c6dcb4a472
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/nicksnyder/go-i18n/v2 v2.1.2
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/volatiletech/sqlboiler/v4 v4.4.0
	github.com/volatiletech/strmangle v0.0.1
	go.cryptoscope.co/muxrpc/v2 v2.0.0-20210202162901-fe642d405dc6
	go.cryptoscope.co/netwrap v0.1.1
	go.cryptoscope.co/secretstream v1.2.2
	go.mindeco.de v1.7.3-0.20210216124941-2da08c6b0080
	go.mindeco.de/ssb-refs v0.1.1-0.20210108133850-cf1f44fea870
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/text v0.3.5
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

exclude go.cryptoscope.co/ssb v0.0.0-20201207161753-31d0f24b7a79

// We need our internal/extra25519 since agl pulled his repo recently.
// Issue: https://github.com/cryptoscope/ssb/issues/44
// Ours uses a fork of x/crypto where edwards25519 is not an internal package,
// This seemed like the easiest change to port agl's extra25519 to use x/crypto
// Background: https://github.com/agl/ed25519/issues/27#issuecomment-591073699
// The branch in use: https://github.com/cryptix/golang_x_crypto/tree/non-internal-edwards
replace golang.org/x/crypto => github.com/cryptix/golang_x_crypto v0.0.0-20200924101112-886946aabeb8
