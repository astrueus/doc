package password

import "testing"

func TestPasswordHashVerifySuccess(t *testing.T) {
	pass := "MySecretPass123!"
	hash, err := PasswordHash(pass)
	if err != nil {
		t.Fatalf("PasswordHash: %v", err)
	}
	if hash == "" {
		t.Fatal("PasswordHash returned empty")
	}
	ok, err := PasswordVerify(hash, pass)
	if err != nil {
		t.Fatalf("PasswordVerify: %v", err)
	}
	if !ok {
		t.Fatal("PasswordVerify should succeed for correct password")
	}
}

func TestPasswordVerifyFail(t *testing.T) {
	hash, err := PasswordHash("correct-password")
	if err != nil {
		t.Fatalf("PasswordHash: %v", err)
	}
	ok, err := PasswordVerify(hash, "wrong-password")
	if err != nil {
		t.Fatalf("PasswordVerify: %v", err)
	}
	if ok {
		t.Fatal("PasswordVerify should fail for wrong password")
	}
}
