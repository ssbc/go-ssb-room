// SPDX-License-Identifier: MIT

// Package aliases implements the validation and signing features of https://ssb-ngi-pointer.github.io/rooms2/#alias
package aliases

import (
	"bytes"

	"golang.org/x/crypto/ed25519"

	refs "go.mindeco.de/ssb-refs"
)

// Registration ties an alias to the ID of the user and the RoomID it should be registerd on
type Registration struct {
	Alias  string
	UserID refs.FeedRef
	RoomID refs.FeedRef
}

// Sign takes the public key (belonging to UserID) and returns the signed confirmation
func (r Registration) Sign(privKey ed25519.PrivateKey) Confirmation {
	var conf Confirmation
	conf.Registration = r
	msg := r.createRegistrationMessage()
	conf.Signature = ed25519.Sign(privKey, msg)
	return conf
}

// createRegistrationMessage returns the string of bytes that should be signed
func (r Registration) createRegistrationMessage() []byte {
	var message bytes.Buffer
	message.WriteString("=room-alias-registration:")
	message.WriteString(r.RoomID.Ref())
	message.WriteString(":")
	message.WriteString(r.UserID.Ref())
	message.WriteString(":")
	message.WriteString(r.Alias)
	return message.Bytes()
}

// Confirmation combines a registration with the corresponding signature
type Confirmation struct {
	Registration

	Signature []byte
}

// Verify checks that the confirmation is for the expected room and from the expected feed
func (c Confirmation) Verify() bool {
	// re-construct the registration
	message := c.createRegistrationMessage()

	// check the signature matches
	return ed25519.Verify(c.UserID.PubKey(), message, c.Signature)
}
