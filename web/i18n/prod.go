// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// +build !dev

package i18n

import (
	"embed"
	"io/fs"
	"log"
)

// Defaults is an embedded filesystem containing translation defaults.
var Defaults fs.FS

//go:embed defaults/*
var embedDefaults embed.FS

func init() {
	var err error
	Defaults, err = fs.Sub(embedDefaults, "defaults")
	if err != nil {
		log.Fatal(err)
	}
}
