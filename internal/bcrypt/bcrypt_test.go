package bcrypt

import (
	"testing"
)

func TestHashGeneratesValidBcryptHash(t *testing.T) {
	password := "testpassword123"
	hash, err := Hash(password, 12)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if len(hash) != 60 {
		t.Errorf("expected hash length 60, got %d", len(hash))
	}
}

func TestVerifyReturnsTrueForCorrectPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := Hash(password, 12)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	match, err := Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !match {
		t.Error("Verify returned false for correct password")
	}
}

func TestVerifyReturnsFalseForIncorrectPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := Hash(password, 12)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	match, err := Verify("wrongpassword", hash)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if match {
		t.Error("Verify returned true for incorrect password")
	}
}

func TestHashWithDifferentCosts(t *testing.T) {
	password := "testpassword123"
	hash4, err := Hash(password, 4)
	if err != nil {
		t.Fatalf("Hash with cost 4 returned error: %v", err)
	}
	hash10, err := Hash(password, 10)
	if err != nil {
		t.Fatalf("Hash with cost 10 returned error: %v", err)
	}

	if hash4 == hash10 {
		t.Error("Different costs should produce different hashes")
	}

	match4, _ := Verify(password, hash4)
	match10, _ := Verify(password, hash10)
	if !match4 {
		t.Error("Verify failed for cost 4 hash")
	}
	if !match10 {
		t.Error("Verify failed for cost 10 hash")
	}
}

func TestHashSamePasswordMultipleTimes(t *testing.T) {
	password := "testpassword123"
	hash1, _ := Hash(password, 12)
	hash2, _ := Hash(password, 12)

	if hash1 == hash2 {
		t.Error("Same password should produce different hashes (salt)")
	}

	match1, _ := Verify(password, hash1)
	match2, _ := Verify(password, hash2)
	if !match1 || !match2 {
		t.Error("Verify should succeed for both hashes")
	}
}
