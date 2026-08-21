package domain

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

type ID string

func NewID(prefix string) ID {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return ID(strings.ToLower(prefix) + "_" + hex.EncodeToString(buf))
}

func (id ID) Empty() bool { return strings.TrimSpace(string(id)) == "" }
