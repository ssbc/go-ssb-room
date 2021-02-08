package admindb

import (
	"go.mindeco.de/http/auth"
)

// FallbackAuth might be helpful for scenarios where one lost access to his ssb device or key
type FallbackAuth interface {
	auth.Auther
}

// AuthService defines functions needed for the challange/response system of sign-in with ssb
type AuthService interface{}

// RoomService deals with changing the privacy modes and managing the allow/deny lists of the room
type RoomService interface{}

// AliasService manages alias handle registration and lookup
type AliasService interface{}

// for tests we use generated mocks from these interfaces created with https://github.com/maxbrunsfeld/counterfeiter

//go:generate counterfeiter -o mockdb/auth.go . AuthService

//go:generate counterfeiter -o mockdb/auth_fallback.go . FallbackAuth

//go:generate counterfeiter -o mockdb/room.go . RoomService

//go:generate counterfeiter -o mockdb/alias.go . AliasService
