// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/maybemod/keys"
	refs "go.mindeco.de/ssb-refs"
)

func DefaultKeyPair(r Interface) (*keys.KeyPair, error) {
	secPath := r.GetPath("secret")
	keyPair, err := keys.LoadKeyPair(secPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("repo: error opening key pair: %w", err)
		}
		keyPair, err = keys.NewKeyPair(nil)
		if err != nil {
			return nil, fmt.Errorf("repo: no keypair but couldn't create one either: %w", err)
		}
		if err := keys.SaveKeyPair(*keyPair, secPath); err != nil {
			return nil, fmt.Errorf("repo: error saving new identity file: %w", err)
		}
		log.Printf("saved identity %s to %s", keyPair.Feed.Ref(), secPath)
	}
	return keyPair, nil
}

func NewKeyPair(r Interface, name, algo string) (*keys.KeyPair, error) {
	return newKeyPair(r, name, algo, nil)
}

func NewKeyPairFromSeed(r Interface, name, algo string, seed io.Reader) (*keys.KeyPair, error) {
	return newKeyPair(r, name, algo, seed)
}

func newKeyPair(r Interface, name, algo string, seed io.Reader) (*keys.KeyPair, error) {
	var secPath string
	if name == "-" {
		secPath = r.GetPath("secret")
	} else {
		secPath = r.GetPath("secrets", name)
		err := os.MkdirAll(filepath.Dir(secPath), 0700)
		if err != nil && !os.IsExist(err) {
			return nil, err
		}
	}
	if algo != refs.RefAlgoFeedSSB1 && algo != refs.RefAlgoFeedGabby { //  enums would be nice
		return nil, fmt.Errorf("invalid feed refrence algo")
	}
	if _, err := keys.LoadKeyPair(secPath); err == nil {
		return nil, fmt.Errorf("new key-pair name already taken")
	}
	keyPair, err := keys.NewKeyPair(seed)
	if err != nil {
		return nil, fmt.Errorf("repo: no keypair but couldn't create one either: %w", err)
	}
	keyPair.Feed.Algo = algo
	if err := keys.SaveKeyPair(*keyPair, secPath); err != nil {
		return nil, fmt.Errorf("repo: error saving new identity file: %w", err)
	}
	log.Printf("saved identity %s to %s", keyPair.Feed.Ref(), secPath)
	return keyPair, nil
}

func LoadKeyPair(r Interface, name string) (*keys.KeyPair, error) {
	secPath := r.GetPath("secrets", name)
	keyPair, err := keys.LoadKeyPair(secPath)
	if err != nil {
		return nil, fmt.Errorf("Load: failed to open %q: %w", secPath, err)
	}
	return keyPair, nil
}

func AllKeyPairs(r Interface) (map[string]*keys.KeyPair, error) {
	kps := make(map[string]*keys.KeyPair)
	err := filepath.Walk(r.GetPath("secrets"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		if kp, err := keys.LoadKeyPair(path); err == nil {
			kps[filepath.Base(path)] = kp
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kps, nil
}
