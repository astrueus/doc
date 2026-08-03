package cryptil

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	secret := "test-secret-key"
	plain := "hello world 你好"
	enc := Encrypt(plain, secret)
	if enc == "" {
		t.Fatal("Encrypt returned empty string")
	}
	got := Decrypt(enc, secret)
	if got != plain {
		t.Fatalf("Decrypt: got %q want %q", got, plain)
	}
}

func TestDecryptWrongSecret(t *testing.T) {
	enc := Encrypt("data", "secret-a")
	if Decrypt(enc, "secret-b") != "" {
		t.Fatal("Decrypt with wrong secret should return empty")
	}
}

func TestDecryptMalformed(t *testing.T) {
	if Decrypt("not.valid", "secret") != "" {
		t.Fatal("malformed input should return empty")
	}
}

func TestMd5Crypt(t *testing.T) {
	noSalt := Md5Crypt("abc")
	if len(noSalt) != 32 {
		t.Fatalf("Md5Crypt length: got %d want 32", len(noSalt))
	}
	withSalt := Md5Crypt("pass%v", "salt")
	if withSalt == noSalt {
		t.Fatal("Md5Crypt with salt should differ from no salt")
	}
	// deterministic
	if Md5Crypt("abc") != Md5Crypt("abc") {
		t.Fatal("Md5Crypt should be deterministic")
	}
}

func TestSha1Crypt(t *testing.T) {
	noSalt := Sha1Crypt("abc")
	if len(noSalt) != 40 {
		t.Fatalf("Sha1Crypt length: got %d want 40", len(noSalt))
	}
	withSalt := Sha1Crypt("pass%v", "salt")
	if withSalt == noSalt {
		t.Fatal("Sha1Crypt with salt should differ from no salt")
	}
	if Sha1Crypt("abc") != Sha1Crypt("abc") {
		t.Fatal("Sha1Crypt should be deterministic")
	}
}

func TestUniqueId(t *testing.T) {
	id := UniqueId()
	if id == "" {
		t.Fatal("UniqueId returned empty")
	}
	if len(id) != 32 {
		t.Fatalf("UniqueId length: got %d want 32", len(id))
	}
	id2 := UniqueId()
	if id == id2 {
		t.Fatal("UniqueId should produce different values")
	}
}

func TestNewRandChars(t *testing.T) {
	if NewRandChars(0) != "" {
		t.Fatal("NewRandChars(0) should return empty string")
	}
	s := NewRandChars(8)
	if len(s) != 8 {
		t.Fatalf("NewRandChars(8) length: got %d want 8", len(s))
	}
	for _, c := range s {
		if !strings.ContainsRune(string(stdChars), c) {
			t.Fatalf("unexpected char %q in %q", c, s)
		}
	}
}
