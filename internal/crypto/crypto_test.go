package crypto

import (
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	if len(salt) != SaltLen {
		t.Fatalf("salt length = %d, want %d", len(salt), SaltLen)
	}

	salt2, _ := GenerateSalt()
	if string(salt) == string(salt2) {
		t.Fatal("two salts are identical")
	}
}

func TestDeriveKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("test-passphrase", salt)
	if len(key) != KeyLen {
		t.Fatalf("key length = %d, want %d", len(key), KeyLen)
	}

	key2 := DeriveKey("test-passphrase", salt)
	if string(key) != string(key2) {
		t.Fatal("same passphrase+salt produced different keys")
	}

	key3 := DeriveKey("different-passphrase", salt)
	if string(key) == string(key3) {
		t.Fatal("different passphrases produced same key")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("my-secret", salt)

	plaintext := "important decision: use JWT"
	encrypted, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("encrypted text equals plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted = %q, want %q", string(decrypted), plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("correct", salt)
	key2 := DeriveKey("wrong", salt)

	encrypted, _ := Encrypt([]byte("secret"), key1)
	_, err := Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("test", salt)

	_, err := Decrypt("not-valid-base64!!!", key)
	if err == nil {
		t.Fatal("Decrypt with invalid base64 should fail")
	}

	_, err = Decrypt("YQ==", key)
	if err == nil {
		t.Fatal("Decrypt with too-short ciphertext should fail")
	}
}

func TestVerifierRoundTrip(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("my-pass", salt)

	verifier, err := CreateVerifier(key)
	if err != nil {
		t.Fatalf("CreateVerifier: %v", err)
	}

	if !CheckVerifier(verifier, key) {
		t.Fatal("CheckVerifier returned false for correct key")
	}
}

func TestVerifierWrongKey(t *testing.T) {
	salt, _ := GenerateSalt()
	key1 := DeriveKey("correct", salt)
	key2 := DeriveKey("wrong", salt)

	verifier, _ := CreateVerifier(key1)

	if CheckVerifier(verifier, key2) {
		t.Fatal("CheckVerifier returned true for wrong key")
	}
}

func TestVerifierInvalidToken(t *testing.T) {
	salt, _ := GenerateSalt()
	key := DeriveKey("test", salt)

	if CheckVerifier("garbage-data", key) {
		t.Fatal("CheckVerifier returned true for invalid token")
	}
}
