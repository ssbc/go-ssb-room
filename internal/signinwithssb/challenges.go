// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package signinwithssb

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/ed25519"

	refs "github.com/ssbc/go-ssb-refs"
)

// sign-in with ssb uses 256-bit nonces
const challengeLength = 32

// DecodeChallengeString accepts base64 encoded strings and decodes them,
// checks their length to be equal to challengeLength,
// and returns the decoded bytes
func DecodeChallengeString(c string) ([]byte, error) {
	challengeBytes, err := base64.URLEncoding.DecodeString(c)
	if err != nil {
		return nil, fmt.Errorf("invalid challenge encoding: %w", err)
	}

	if n := len(challengeBytes); n != challengeLength {
		return nil, fmt.Errorf("invalid challenge length: expected %d but got %d", challengeLength, n)
	}

	return challengeBytes, nil
}

// GenerateChallenge returs a base64 encoded string
//  with challangeLength bytes of random data
func GenerateChallenge() string {
	buf := make([]byte, challengeLength)
	rand.Read(buf)
	return base64.URLEncoding.EncodeToString(buf)
}

// ClientPayload is used to create and verify solutions
type ClientPayload struct {
	ClientID, ServerID refs.FeedRef

	ClientChallenge string
	ServerChallenge string
}

// recreate the signed message
func (cr ClientPayload) createMessage() []byte {
	var msg bytes.Buffer
	msg.WriteString("=http-auth-sign-in:")
	msg.WriteString(cr.ServerID.String())
	msg.WriteString(":")
	msg.WriteString(cr.ClientID.String())
	msg.WriteString(":")
	msg.WriteString(cr.ServerChallenge)
	msg.WriteString(":")
	msg.WriteString(cr.ClientChallenge)
	return msg.Bytes()
}

// Sign returns the signature created with the passed privateKey
func (cr ClientPayload) Sign(privateKey ed25519.PrivateKey) []byte {
	msg := cr.createMessage()
	return ed25519.Sign(privateKey, msg)
}

// Validate checks the signature by calling createMessage() and ed25519.Verify()
// together with the ClientID public key.
func (cr ClientPayload) Validate(signature []byte) bool {
	msg := cr.createMessage()
	return ed25519.Verify(cr.ClientID.PubKey(), msg, signature)
}
