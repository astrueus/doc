package gob

import "testing"

type sampleToken struct {
	UserID int
	Name   string
	Active bool
}

func TestEncodeDecode(t *testing.T) {
	in := sampleToken{UserID: 42, Name: "alice", Active: true}
	encoded, err := Encode(in)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("Encode returned empty string")
	}
	var out sampleToken
	if err := Decode(encoded, &out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip: got %+v want %+v", out, in)
	}
}
