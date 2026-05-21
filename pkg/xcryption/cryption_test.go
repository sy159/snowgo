package xcryption_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"snowgo/pkg/xcryption"
)

func TestAesGCM_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: empty plaintext roundtrip", func(t *testing.T) {
		key := "aa125678aa125678"
		encrypt, err := xcryption.AesGCMEncrypt("", key)
		if err != nil {
			t.Fatalf("AesGCMEncrypt empty plaintext error: %v", err)
		}
		if encrypt == "" {
			t.Fatal("ciphertext should not be empty (contains nonce)")
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("AesGCMDecrypt error: %v", err)
		}
		if decrypt != "" {
			t.Fatalf("expected empty plaintext, got %q", decrypt)
		}
	})

	t.Run("boundary: single char plaintext", func(t *testing.T) {
		key := "aa125678aa125678"
		encrypt, err := xcryption.AesGCMEncrypt("a", key)
		if err != nil {
			t.Fatalf("AesGCMEncrypt single char error: %v", err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("AesGCMDecrypt error: %v", err)
		}
		if decrypt != "a" {
			t.Fatalf("decrypt mismatch: got %q, want %q", decrypt, "a")
		}
	})

	t.Run("boundary: very long plaintext", func(t *testing.T) {
		key := "aa125678aa125678"
		plain := make([]byte, 10000)
		for i := range plain {
			plain[i] = byte('a' + i%26)
		}
		encrypt, err := xcryption.AesGCMEncrypt(string(plain), key)
		if err != nil {
			t.Fatalf("AesGCMEncrypt long plaintext error: %v", err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("AesGCMDecrypt error: %v", err)
		}
		if decrypt != string(plain) {
			t.Fatal("decrypt mismatch for long plaintext")
		}
	})

	t.Run("boundary: key length 16 (minimum AES-128)", func(t *testing.T) {
		encrypt, err := xcryption.AesGCMEncrypt("hello", "1234567890123456")
		if err != nil {
			t.Fatalf("16-byte key error: %v", err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, "1234567890123456")
		if err != nil {
			t.Fatalf("16-byte key decrypt error: %v", err)
		}
		if decrypt != "hello" {
			t.Fatal("16-byte key decrypt mismatch")
		}
	})

	t.Run("boundary: key length 32 (maximum AES-256)", func(t *testing.T) {
		encrypt, err := xcryption.AesGCMEncrypt("hello", "12345678901234567890123456789012")
		if err != nil {
			t.Fatalf("32-byte key error: %v", err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, "12345678901234567890123456789012")
		if err != nil {
			t.Fatalf("32-byte key decrypt error: %v", err)
		}
		if decrypt != "hello" {
			t.Fatal("32-byte key decrypt mismatch")
		}
	})

	t.Run("boundary: key length 24 (AES-192)", func(t *testing.T) {
		key := "123456789012345678901234"
		encrypt, err := xcryption.AesGCMEncrypt("hello", key)
		if err != nil {
			t.Fatalf("24-byte key error: %v", err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("24-byte key decrypt error: %v", err)
		}
		if decrypt != "hello" {
			t.Fatal("24-byte key decrypt mismatch")
		}
	})

	// === Expected errors ===
	t.Run("error: key length 15 (too short)", func(t *testing.T) {
		_, err := xcryption.AesGCMEncrypt("hello", "123456789012345")
		if err == nil {
			t.Fatal("expected error for 15-byte key")
		}
	})

	t.Run("error: key length 17 (invalid)", func(t *testing.T) {
		_, err := xcryption.AesGCMEncrypt("hello", "12345678901234567")
		if err == nil {
			t.Fatal("expected error for 17-byte key")
		}
	})

	t.Run("error: empty key", func(t *testing.T) {
		_, err := xcryption.AesGCMEncrypt("hello", "")
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})

	t.Run("error: empty ciphertext", func(t *testing.T) {
		_, err := xcryption.AesGCMDecrypt("", "aa125678aa125678")
		if err == nil {
			t.Fatal("expected error for empty ciphertext")
		}
	})

	t.Run("error: tampered ciphertext", func(t *testing.T) {
		key := "aa125678aa125678"
		encrypt, err := xcryption.AesGCMEncrypt("sensitive data", key)
		if err != nil {
			t.Fatalf("AesGCMEncrypt error: %v", err)
		}
		// Flip a byte in the middle of the ciphertext
		b := []byte(encrypt)
		b[len(b)/2] ^= 0xFF
		_, err = xcryption.AesGCMDecrypt(string(b), key)
		if err == nil {
			t.Fatal("expected error for tampered ciphertext")
		}
	})

	t.Run("error: decrypt with wrong key", func(t *testing.T) {
		encrypt, err := xcryption.AesGCMEncrypt("hello", "aa125678aa125678")
		if err != nil {
			t.Fatalf("AesGCMEncrypt error: %v", err)
		}
		_, err = xcryption.AesGCMDecrypt(encrypt, "bb125678bb125678")
		if err == nil {
			t.Fatal("expected error for wrong key")
		}
	})

	t.Run("error: truncated ciphertext (shorter than nonce)", func(t *testing.T) {
		_, err := xcryption.AesGCMDecrypt("YQ==", "aa125678aa125678") // "a" is 1 byte, much less than 12-byte nonce
		if err == nil {
			t.Fatal("expected error for truncated ciphertext")
		}
	})
}

func TestHashPassword_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: empty password", func(t *testing.T) {
		hashPwd, err := xcryption.HashPassword("")
		if err != nil {
			t.Fatalf("HashPassword empty error: %v", err)
		}
		if hashPwd == "" {
			t.Fatal("hash should not be empty")
		}
		if !xcryption.CheckPassword(hashPwd, "") {
			t.Fatal("CheckPassword failed for empty password")
		}
	})

	t.Run("boundary: single char password", func(t *testing.T) {
		hashPwd, err := xcryption.HashPassword("a")
		if err != nil {
			t.Fatalf("HashPassword single char error: %v", err)
		}
		if !xcryption.CheckPassword(hashPwd, "a") {
			t.Fatal("CheckPassword failed for single char")
		}
	})

	t.Run("boundary: very long password (72 bytes, bcrypt max)", func(t *testing.T) {
		long := make([]byte, 72)
		for i := range long {
			long[i] = byte('a' + i%26)
		}
		hashPwd, err := xcryption.HashPassword(string(long))
		if err != nil {
			t.Fatalf("HashPassword 72-byte error: %v", err)
		}
		if !xcryption.CheckPassword(hashPwd, string(long)) {
			t.Fatal("CheckPassword failed for 72-byte password")
		}
	})

	t.Run("boundary: password exceeds 72 bytes (bcrypt limit) returns error", func(t *testing.T) {
		long := make([]byte, 73)
		_, err := xcryption.HashPassword(string(long))
		if err == nil {
			t.Fatal("expected error for password > 72 bytes")
		}
	})

	// === Happy path ===
	t.Run("happy: wrong password fails", func(t *testing.T) {
		hashPwd, err := xcryption.HashPassword("correct")
		if err != nil {
			t.Fatalf("HashPassword error: %v", err)
		}
		if xcryption.CheckPassword(hashPwd, "wrong") {
			t.Fatal("CheckPassword should fail for wrong password")
		}
	})
}

func TestSha256_Boundary(t *testing.T) {
	t.Run("boundary: empty string", func(t *testing.T) {
		result := xcryption.Sha256("")
		if result == "" {
			t.Fatal("Sha256 of empty string should not be empty")
		}
		expected := sha256.Sum256([]byte(""))
		if result != hex.EncodeToString(expected[:]) {
			t.Fatal("Sha256 empty mismatch with stdlib")
		}
	})

	t.Run("happy: deterministic output", func(t *testing.T) {
		h1 := xcryption.Sha256("test")
		h2 := xcryption.Sha256("test")
		if h1 != h2 {
			t.Fatal("Sha256 should be deterministic")
		}
		// Verify against stdlib
		expected := sha256.Sum256([]byte("test"))
		if h1 != hex.EncodeToString(expected[:]) {
			t.Fatal("Sha256 mismatch with stdlib")
		}
	})
}
