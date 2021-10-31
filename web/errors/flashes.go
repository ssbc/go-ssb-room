// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package errors

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
)

type FlashHelper struct {
	store sessions.Store

	locHelper *i18n.Helper
}

func NewFlashHelper(s sessions.Store, loc *i18n.Helper) *FlashHelper {
	gob.Register(FlashMessage{})

	return &FlashHelper{
		store:     s,
		locHelper: loc,
	}
}

const flashSession = "go-ssb-room-flash-messages"

type FlashKind uint

const (
	_ FlashKind = iota
	// FlashError signals that a problem occured
	FlashError
	// FlashNotification represents a normal message (like "xyz added/updated successfull")
	FlashNotification
)

type FlashMessage struct {
	Kind    FlashKind
	Message template.HTML
}

// TODO: rethink error return - maybe panic() / maybe render package?

// AddMessage expects a i18n label, translates it and adds it as a FlashNotification
func (fh FlashHelper) AddMessage(rw http.ResponseWriter, req *http.Request, label string) {
	session, err := fh.store.Get(req, flashSession)
	if err != nil {
		panic(fmt.Errorf("flashHelper: failed to get session: %w", err))
	}

	ih := fh.locHelper.FromRequest(req)

	session.AddFlash(FlashMessage{
		Kind:    FlashNotification,
		Message: ih.LocalizeSimple(label),
	})

	if err := session.Save(req, rw); err != nil {
		panic(fmt.Errorf("flashHelper: failed to save session: %w", err))
	}
}

// AddError adds a FlashError and translates the passed err using localizeError()
func (fh FlashHelper) AddError(rw http.ResponseWriter, req *http.Request, err error) {
	session, getErr := fh.store.Get(req, flashSession)
	if getErr != nil {
		panic(fmt.Errorf("flashHelper: failed to get session: %w", err))
	}

	ih := fh.locHelper.FromRequest(req)

	_, msg := localizeError(ih, err)

	session.AddFlash(FlashMessage{
		Kind:    FlashError,
		Message: msg,
	})
	if err := session.Save(req, rw); err != nil {
		panic(fmt.Errorf("flashHelper: failed to save session: %w", err))
	}
}

// GetAll returns all the FlashMessages, emptys and updates the store
func (fh FlashHelper) GetAll(rw http.ResponseWriter, req *http.Request) ([]FlashMessage, error) {
	session, err := fh.store.Get(req, flashSession)
	if err != nil {
		return nil, err
	}

	opaqueFlashes := session.Flashes()

	flashes := make([]FlashMessage, len(opaqueFlashes))

	for i, of := range opaqueFlashes {
		f, ok := of.(FlashMessage)
		if !ok {
			return nil, fmt.Errorf("GetFlashes: failed to unpack flash: %T", of)
		}

		flashes[i].Kind = f.Kind
		flashes[i].Message = f.Message
	}

	err = session.Save(req, rw)

	return flashes, err
}
