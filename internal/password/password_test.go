package password

import (
	"testing"
)

func TestGenerateDefaultLength(t *testing.T) {
	pwd, err := Generate(Options{Length: 16})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(pwd) != 16 {
		t.Errorf("expected length 16, got %d", len(pwd))
	}
}

func TestGenerateCustomLength(t *testing.T) {
	pwd, err := Generate(Options{Length: 24})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(pwd) != 24 {
		t.Errorf("expected length 24, got %d", len(pwd))
	}
}

func TestGenerateDigitsOnly(t *testing.T) {
	pwd, err := Generate(Options{Length: 32, DigitsOnly: true})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	for _, c := range pwd {
		if c < '0' || c > '9' {
			t.Errorf("digits-only password contains non-digit: %c", c)
		}
	}
}

func TestGenerateUppercaseOnly(t *testing.T) {
	pwd, err := Generate(Options{Length: 32, UppercaseOnly: true})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	for _, c := range pwd {
		if c < 'A' || c > 'Z' {
			t.Errorf("uppercase-only password contains non-uppercase: %c", c)
		}
	}
}

func TestGenerateNoSpecialChars(t *testing.T) {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for i := 0; i < 10; i++ {
		pwd, err := Generate(Options{Length: 64, IncludeSpecial: false})
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		for _, c := range pwd {
			if string(c) == "" {
				continue
			}
			for _, s := range specialChars {
				if c == s {
					t.Errorf("password contains special char: %c", c)
				}
			}
		}
	}
}

func TestGenerateExcludeChars(t *testing.T) {
	excluded := "|![]$`"
	for i := 0; i < 10; i++ {
		pwd, err := Generate(Options{Length: 64, IncludeSpecial: true, ExcludeChars: excluded})
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		for _, c := range pwd {
			for _, s := range excluded {
				if c == s {
					t.Errorf("password contains excluded char %c: %s", c, pwd)
				}
			}
		}
	}
}

func TestGenerateExcludeAllChars(t *testing.T) {
	// Excluding all possible characters should return an error
	allChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	_, err := Generate(Options{Length: 16, IncludeSpecial: true, ExcludeChars: allChars})
	if err == nil {
		t.Error("expected error when excluding all characters, got nil")
	}
}

func TestGenerateUniquePasswords(t *testing.T) {
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pwd, err := Generate(Options{Length: 16})
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		if passwords[pwd] {
			t.Errorf("duplicate password generated: %s", pwd)
		}
		passwords[pwd] = true
	}
}
