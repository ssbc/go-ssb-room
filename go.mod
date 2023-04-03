// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: Unlicense

module github.com/ssbc/go-ssb-room/v2

go 1.16

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/BurntSushi/toml v1.2.1
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/dustin/go-humanize v1.0.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/gorilla/csrf v1.7.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattevans/pwned-passwords v0.6.0
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/maxbrunsfeld/counterfeiter/v6 v6.5.0
	github.com/mileusna/useragent v1.2.1
	github.com/nicksnyder/go-i18n/v2 v2.2.1
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v1.2.0
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/ssbc/go-muxrpc/v2 v2.0.14-0.20221111190521-10382533750c
	github.com/ssbc/go-netwrap v0.1.5-0.20221019160355-cd323bb2e29d
	github.com/ssbc/go-secretstream v1.2.11-0.20221111164233-4b41f899f844
	github.com/ssbc/go-ssb-refs v0.5.2
	github.com/stretchr/testify v1.8.1
	github.com/throttled/throttled/v2 v2.9.1
	github.com/unrolled/secure v1.13.0
	github.com/vcraescu/go-paginator/v2 v2.0.0
	github.com/volatiletech/sqlboiler/v4 v4.14.0
	github.com/volatiletech/strmangle v0.0.4
	go.cryptoscope.co/nocomment v0.0.0-20210520094614-fb744e81f810
	go.mindeco.de v1.12.0
	golang.org/x/crypto v0.4.0
	golang.org/x/sync v0.1.0
	golang.org/x/text v0.5.0
	golang.org/x/tools v0.4.0
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gorm.io/gorm v1.24.1 // indirect
)

exclude go.cryptoscope.co/ssb v0.0.0-20201207161753-31d0f24b7a79

// https://github.com/rubenv/sql-migrate/pull/189
// and using branch 'drop-other-drivers' for less dependency pollution (oracaldb and the like)
replace github.com/rubenv/sql-migrate => github.com/cryptix/go-sql-migrate v0.0.0-20210521142015-a3e4d9974764
