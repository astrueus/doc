// Package gob provides Encode/Decode helpers for cookie remember tokens.
// Implementation uses msgpack (not encoding/gob) so type names stay package-path independent.
package gob

import (
	"github.com/vmihailenco/msgpack/v5"
)

// Decode unmarshals msgpack bytes from value into r.
func Decode(value string, r any) error {
	return msgpack.Unmarshal([]byte(value), r)
}

// Encode marshals value to a msgpack string.
func Encode(value any) (string, error) {
	b, err := msgpack.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
