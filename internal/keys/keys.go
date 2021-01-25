// SPDX-License-Identifier: MIT

// Package keys could be it's own thing between go-ssb and this but not today
package keys

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/keks/nocomment"
	"go.cryptoscope.co/secretstream/secrethandshake"
	refs "go.mindeco.de/ssb-refs"
)

var SecretPerms = os.FileMode(0600)

type KeyPair struct {
	Id   *refs.FeedRef
	Pair secrethandshake.EdKeyPair
}

// the format of the .ssb/secret file as defined by the js implementations
type ssbSecret struct {
	Curve   string        `json:"curve"`
	ID      *refs.FeedRef `json:"id"`
	Private string        `json:"private"`
	Public  string        `json:"public"`
}

// IsValidFeedFormat checks if the passed FeedRef is for one of the two supported formats,
// legacy/crapp or GabbyGrove.
func IsValidFeedFormat(r *refs.FeedRef) error {
	if r.Algo != refs.RefAlgoFeedSSB1 && r.Algo != refs.RefAlgoFeedGabby {
		return fmt.Errorf("ssb: unsupported feed format:%s", r.Algo)
	}
	return nil
}

// NewKeyPair generates a fresh KeyPair using the passed io.Reader as a seed.
// Passing nil is fine and will use crypto/rand.
func NewKeyPair(r io.Reader) (*KeyPair, error) {
	// generate new keypair
	kp, err := secrethandshake.GenEdKeyPair(r)
	if err != nil {
		return nil, fmt.Errorf("ssb: error building key pair: %w", err)
	}

	keyPair := KeyPair{
		Id:   &refs.FeedRef{ID: kp.Public[:], Algo: refs.RefAlgoFeedSSB1},
		Pair: *kp,
	}

	return &keyPair, nil
}

// SaveKeyPair serializes the passed KeyPair to path.
// It errors if path already exists.
func SaveKeyPair(kp *KeyPair, path string) error {
	if err := IsValidFeedFormat(kp.Id); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ssb.SaveKeyPair: key already exists:%q", path)
	}
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create folder for keypair: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, SecretPerms)
	if err != nil {
		return fmt.Errorf("ssb.SaveKeyPair: failed to create file: %w", err)
	}

	if err := EncodeKeyPairAsJSON(kp, f); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("ssb.SaveKeyPair: failed to close file: %w", err)
	}

	return nil
}

// EncodeKeyPairAsJSON serializes the passed Keypair into the writer w
func EncodeKeyPairAsJSON(kp *KeyPair, w io.Writer) error {
	var sec = ssbSecret{
		Curve:   "ed25519",
		ID:      kp.Id,
		Private: base64.StdEncoding.EncodeToString(kp.Pair.Secret[:]) + ".ed25519",
		Public:  base64.StdEncoding.EncodeToString(kp.Pair.Public[:]) + ".ed25519",
	}
	err := json.NewEncoder(w).Encode(sec)
	if err != nil {
		return fmt.Errorf("ssb.EncodeKeyPairAsJSON: encoding failed: %w", err)
	}
	return nil
}

// LoadKeyPair opens fname, ignores any line starting with # and passes it ParseKeyPair
func LoadKeyPair(fname string) (*KeyPair, error) {
	f, err := os.Open(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}

		return nil, fmt.Errorf("ssb.LoadKeyPair: could not open key file %s: %w", fname, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("ssb.LoadKeyPair: could not stat key file %s: %w", fname, err)
	}

	if perms := info.Mode().Perm(); perms != SecretPerms {
		return nil, fmt.Errorf("ssb.LoadKeyPair: expected key file permissions %s, but got %s", SecretPerms, perms)
	}

	return ParseKeyPair(nocomment.NewReader(f))
}

// ParseKeyPair json decodes an object from the reader.
// It expects std base64 encoded data under the `private` and `public` fields.
func ParseKeyPair(r io.Reader) (*KeyPair, error) {
	var s ssbSecret
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, fmt.Errorf("ssb.Parse: JSON decoding failed: %w", err)
	}

	if err := IsValidFeedFormat(s.ID); err != nil {
		return nil, err
	}

	public, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(s.Public, ".ed25519"))
	if err != nil {
		return nil, fmt.Errorf("ssb.Parse: base64 decode of public part failed: %w", err)
	}

	private, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(s.Private, ".ed25519"))
	if err != nil {
		return nil, fmt.Errorf("ssb.Parse: base64 decode of private part failed: %w", err)
	}

	pair, err := secrethandshake.NewKeyPair(public, private)
	if err != nil {
		return nil, fmt.Errorf("ssb.Parse: base64 decode of private part failed: %w", err)
	}

	ssbkp := KeyPair{
		Id:   s.ID,
		Pair: *pair,
	}
	return &ssbkp, nil
}
