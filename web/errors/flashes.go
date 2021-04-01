package errors

import (
	"encoding/gob"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
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
	Message string
}

// TODO: rethink error return - maybe panic() / maybe render package?

// AddMessage expects a i18n label, translates it and adds it as a FlashNotification
func (fh FlashHelper) AddMessage(rw http.ResponseWriter, req *http.Request, label string) error {
	session, err := fh.store.Get(req, flashSession)
	if err != nil {
		return err
	}

	ih := i18n.LocalizerFromRequest(fh.locHelper, req)

	session.AddFlash(FlashMessage{
		Kind:    FlashNotification,
		Message: ih.LocalizeSimple(label),
	})

	return session.Save(req, rw)
}

// AddError adds a FlashError and translates the passed err using localizeError()
func (fh FlashHelper) AddError(rw http.ResponseWriter, req *http.Request, err error) error {
	session, getErr := fh.store.Get(req, flashSession)
	if getErr != nil {
		return getErr
	}

	ih := i18n.LocalizerFromRequest(fh.locHelper, req)

	_, msg := localizeError(ih, err)

	session.AddFlash(FlashMessage{
		Kind:    FlashError,
		Message: msg,
	})

	return session.Save(req, rw)
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
