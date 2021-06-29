module github.com/ssb-ngi-pointer/go-ssb-room/v2

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/PuerkitoBio/goquery v1.5.0
	github.com/cryptix/front v0.0.0-20210629121817-246a2cb32d0c // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/friendsofgo/errors v0.9.2
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gorilla/csrf v1.7.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gorilla/websocket v1.4.2
	github.com/mattevans/pwned-passwords v0.3.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/maxbrunsfeld/counterfeiter/v6 v6.3.0
	github.com/nicksnyder/go-i18n/v2 v2.1.2
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stretchr/testify v1.7.0
	github.com/throttled/throttled/v2 v2.7.1
	github.com/unrolled/secure v1.0.8
	github.com/vcraescu/go-paginator/v2 v2.0.0
	github.com/volatiletech/sqlboiler/v4 v4.5.0
	github.com/volatiletech/strmangle v0.0.1
	go.cryptoscope.co/muxrpc/v2 v2.0.6
	go.cryptoscope.co/netwrap v0.1.1
	go.cryptoscope.co/nocomment v0.0.0-20210520094614-fb744e81f810
	go.cryptoscope.co/secretstream v1.2.8
	go.mindeco.de v1.12.0
	go.mindeco.de/ssb-refs v0.2.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/text v0.3.5
	golang.org/x/tools v0.1.1
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

// https://github.com/rubenv/sql-migrate/pull/189
// and using branch 'drop-other-drivers' for less dependency pollution (oracaldb and the like)
replace github.com/rubenv/sql-migrate => github.com/cryptix/go-sql-migrate v0.0.0-20210521142015-a3e4d9974764
