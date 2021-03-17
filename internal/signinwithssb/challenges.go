package signinwithssb

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/ed25519"

	refs "go.mindeco.de/ssb-refs"
)

// sign-in with ssb uses 256-bit nonces
const challengeLength = 32

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

func GenerateChallenge() string {
	buf := make([]byte, challengeLength)
	rand.Read(buf)
	return base64.URLEncoding.EncodeToString(buf)
}

// this structure is used to verify an incoming client response
type ClientRequest struct {
	ClientID, ServerID refs.FeedRef

	ClientChallenge string
	ServerChallenge string
}

// recreate the signed message
func (cr ClientRequest) createMessage() []byte {
	var msg bytes.Buffer
	msg.WriteString("=http-auth-sign-in:")
	msg.WriteString(cr.ServerID.Ref())
	msg.WriteString(":")
	msg.WriteString(cr.ClientID.Ref())
	msg.WriteString(":")
	msg.WriteString(cr.ServerChallenge)
	msg.WriteString(":")
	msg.WriteString(cr.ClientChallenge)
	return msg.Bytes()
}

func (cr ClientRequest) Sign(privateKey ed25519.PrivateKey) []byte {
	msg := cr.createMessage()
	return ed25519.Sign(privateKey, msg)
}

func (cr ClientRequest) Validate(signature []byte) bool {
	msg := cr.createMessage()
	return ed25519.Verify(cr.ClientID.PubKey(), msg, signature)
}
