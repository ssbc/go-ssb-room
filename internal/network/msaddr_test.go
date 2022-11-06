// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package network

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/stretchr/testify/assert"
)

func TestMultiserverAddress(t *testing.T) {
	a := assert.New(t)

	var sed ServerEndpointDetails
	sed.Domain = "the.ho.st"
	sed.ListenAddressMUXRPC = ":8008"

	roomID, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("ohai"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}
	sed.RoomID = roomID

	gotMultiAddr := sed.MultiserverAddress()

	a.Equal("net:the.ho.st:8008~shs:b2hhaW9oYWlvaGFpb2hhaW9oYWlvaGFpb2hhaW9oYWk=", gotMultiAddr)
	a.True(strings.HasPrefix(gotMultiAddr, "net:the.ho.st:8008~shs:"), "not for the test host? %s", gotMultiAddr)
	a.True(strings.HasSuffix(gotMultiAddr, base64.StdEncoding.EncodeToString(sed.RoomID.PubKey())), "public key missing? %s", gotMultiAddr)

}
