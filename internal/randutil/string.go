// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package randutil

import "math/rand"

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// String returns a random string of length n, using the alphnum character set (a-z, A-Z, 0-9)
func String(n int) string {
	s := make([]rune, n)

	for i := range s {
		s[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(s)
}
