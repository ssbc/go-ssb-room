// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package aliases

// IsValid decides whether an alias is okay for use or not.
// The room spec defines it as _labels valid under RFC 1035_ ( https://ssb-ngi-pointer.github.io/rooms2/#alias-string )
// but that can be mostly any string since DNS is a 8bit binary protocol,
// as long as it's shorter then 63 charachters.
//
// Right now it's pretty basic set of characters (a-z, 0-9).
// In theory we could be more liberal but there is a bunch of stuff to figure out,
// like homograph attacks (https://en.wikipedia.org/wiki/IDN_homograph_attack),
// if we would decide to allow full utf8 unicode.
func IsValid(alias string) bool {
	if len(alias) > 63 {
		return false
	}

	var valid = true
	for _, char := range alias {
		if char >= '0' && char <= '9' { // is an ASCII number
			continue
		}

		if char >= 'a' && char <= 'z' { // is an ASCII char between a and z
			continue
		}

		valid = false
		break
	}
	return valid
}
