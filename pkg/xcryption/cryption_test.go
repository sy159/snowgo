package xcryption_test

import (
	"snowgo/pkg/xcryption"
	"testing"
)

func TestCrypto(t *testing.T) {
	plainText, key := "hello world", "aa125678aa125678" // AesKey:"1234567890123456", // 必须是 16, 24 或 32 字节
	t.Run("aes cbc encrypt/decrypt", func(t *testing.T) {
		encrypt, err := xcryption.AesCBCEncrypt(plainText, key)
		if err != nil {
			t.Fatalf("AesCBCEncrypt error: %v", err)
		}

		decrypt, err := xcryption.AesCBCDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("AesCBCDecrypt error: %v", err)
		}
		if plainText != decrypt {
			t.Fatal("decrypt mismatch")
		}
	})
}

func TestHashPassword(t *testing.T) {
	pwd := "123456"
	t.Run("hash pwd", func(t *testing.T) {
		hashPwd, err := xcryption.HashPassword(pwd)
		if err != nil {
			t.Fatalf("HashPassword error: %v", err)
		}
		isSuccess := xcryption.CheckPassword(hashPwd, pwd)
		if !isSuccess {
			t.Fatal("password verification failed")
		}
	})
}

func TestSha256(t *testing.T) {
	t.Run("basic hash", func(t *testing.T) {
		hash := xcryption.Sha256("hello world")
		if len(hash) != 64 {
			t.Fatalf("expected 64 char hex string, got %d", len(hash))
		}
		// SHA256 of "hello world" is known
		expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		if hash != expected {
			t.Fatalf("hash mismatch: got %s, want %s", hash, expected)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		hash := xcryption.Sha256("")
		expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		if hash != expected {
			t.Fatalf("empty string hash mismatch: got %s, want %s", hash, expected)
		}
	})
}

func TestAesCBCEncryptError(t *testing.T) {
	t.Run("invalid key length", func(t *testing.T) {
		_, err := xcryption.AesCBCEncrypt("hello", "short")
		if err == nil {
			t.Fatal("expected error for invalid key length")
		}
	})
}

func TestAesCBCDecryptError(t *testing.T) {
	t.Run("invalid key length", func(t *testing.T) {
		_, err := xcryption.AesCBCDecrypt("dGVzdA==", "short")
		if err == nil {
			t.Fatal("expected error for invalid key length")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := xcryption.AesCBCDecrypt("not-valid-base64!!!", "aa125678aa125678")
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
	})

	t.Run("insufficient ciphertext", func(t *testing.T) {
		_, err := xcryption.AesCBCDecrypt("YWJj", "aa125678aa125678") // "abc" base64, too short
		if err == nil {
			t.Fatal("expected error for insufficient ciphertext")
		}
	})

	t.Run("wrong key decrypt", func(t *testing.T) {
		plainText, key := "hello world", "aa125678aa125678"
		encrypt, err := xcryption.AesCBCEncrypt(plainText, key)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		// AES-CBC does not authenticate ciphertext, so wrong key may not error
		// (padding check may pass by chance). Verify the decrypted value differs.
		decrypted, err := xcryption.AesCBCDecrypt(encrypt, "bb125678bb125678")
		if err == nil && decrypted == plainText {
			t.Fatal("wrong key should not produce correct plaintext")
		}
	})
}

func TestPkcs7UnPaddingEdgeCases(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		// This tests the internal unPadding with empty input
		_, err := xcryption.AesCBCDecrypt("", "aa125678aa125678")
		if err == nil {
			t.Fatal("expected error for empty ciphertext")
		}
	})
}

func BenchmarkCrypto(b *testing.B) {
	plainText, key := "hello world", "aa125678aa125678"
	for i := 0; i < b.N; i++ {
		encrypt, err := xcryption.AesCBCEncrypt(plainText, key)
		if err != nil {
			b.Error(err)
		}
		decrypt, err := xcryption.AesCBCDecrypt(encrypt, key)
		if err != nil {
			b.Error(err)
		}
		if plainText != decrypt {
			b.Error("decrypt xerror")
		}
	}
}

func BenchmarkHashPwd(b *testing.B) {
	pwd := "123456"
	for i := 0; i < b.N; i++ {
		hashPwd, err := xcryption.HashPassword(pwd)
		if err != nil {
			b.Error(err)
		}
		isSuccess := xcryption.CheckPassword(hashPwd, pwd)
		if !isSuccess {
			b.Error("hash password error")
		}
	}
}
