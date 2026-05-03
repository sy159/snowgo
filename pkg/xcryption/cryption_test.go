package xcryption_test

import (
	"snowgo/pkg/xcryption"
	"testing"
)

func TestCrypto(t *testing.T) {
	plainText, key := "hello world", "aa125678aa125678"
	t.Run("aes gcm encrypt/decrypt", func(t *testing.T) {
		encrypt, err := xcryption.AesGCMEncrypt(plainText, key)
		if err != nil {
			t.Fatalf("AesGCMEncrypt error: %v", err)
		}

		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			t.Fatalf("AesGCMDecrypt error: %v", err)
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

func TestAesGCMEncryptError(t *testing.T) {
	t.Run("invalid key length", func(t *testing.T) {
		_, err := xcryption.AesGCMEncrypt("hello", "short")
		if err == nil {
			t.Fatal("expected error for invalid key length")
		}
	})
}

func TestAesGCMDecryptError(t *testing.T) {
	t.Run("invalid key length", func(t *testing.T) {
		_, err := xcryption.AesGCMDecrypt("dGVzdA==", "short")
		if err == nil {
			t.Fatal("expected error for invalid key length")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := xcryption.AesGCMDecrypt("not-valid-base64!!!", "aa125678aa125678")
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
	})

	t.Run("insufficient ciphertext", func(t *testing.T) {
		_, err := xcryption.AesGCMDecrypt("YWJj", "aa125678aa125678") // "abc" base64, too short
		if err == nil {
			t.Fatal("expected error for insufficient ciphertext")
		}
	})

	t.Run("wrong key decrypt", func(t *testing.T) {
		plainText, key := "hello world", "aa125678aa125678"
		encrypt, err := xcryption.AesGCMEncrypt(plainText, key)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		// WHY: GCM 内置认证标签，密钥错误时 Open 一定返回 error，不会像 CBC 那样静默解密出乱码
		_, err = xcryption.AesGCMDecrypt(encrypt, "bb125678bb125678")
		if err == nil {
			t.Fatal("expected error when decrypting with wrong key (GCM authenticates)")
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		plainText, key := "hello world", "aa125678aa125678"
		encrypt, err := xcryption.AesGCMEncrypt(plainText, key)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		// 篡改 base64 编码后的密文（改变最后一个字符）
		tampered := encrypt[:len(encrypt)-4] + "XXXX"
		_, err = xcryption.AesGCMDecrypt(tampered, key)
		if err == nil {
			t.Fatal("expected error when decrypting tampered ciphertext (GCM authenticates)")
		}
	})
}

func BenchmarkCrypto(b *testing.B) {
	plainText, key := "hello world", "aa125678aa125678"
	for i := 0; i < b.N; i++ {
		encrypt, err := xcryption.AesGCMEncrypt(plainText, key)
		if err != nil {
			b.Fatal(err)
		}
		decrypt, err := xcryption.AesGCMDecrypt(encrypt, key)
		if err != nil {
			b.Fatal(err)
		}
		if plainText != decrypt {
			b.Fatal("decrypt mismatch")
		}
	}
}

func BenchmarkHashPwd(b *testing.B) {
	pwd := "123456"
	for i := 0; i < b.N; i++ {
		hashPwd, err := xcryption.HashPassword(pwd)
		if err != nil {
			b.Fatal(err)
		}
		isSuccess := xcryption.CheckPassword(hashPwd, pwd)
		if !isSuccess {
			b.Fatal("hash password error")
		}
	}
}

func BenchmarkId2Code(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xcryption.Id2Code(int64(i%1000000), 8)
	}
}

func BenchmarkId2CodeParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := int64(0)
		for pb.Next() {
			i++
			_, _ = xcryption.Id2Code(i%1000000, 8)
		}
	})
}

func BenchmarkCode2Id(b *testing.B) {
	code, _ := xcryption.Id2Code(12345, 8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = xcryption.Code2Id(code)
	}
}

func BenchmarkSha256(b *testing.B) {
	s := "hello world, this is a benchmark test string"
	for i := 0; i < b.N; i++ {
		xcryption.Sha256(s)
	}
}

func BenchmarkAesGCMEncrypt(b *testing.B) {
	plainText := "hello world, benchmark data"
	key := "aa125678aa125678"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = xcryption.AesGCMEncrypt(plainText, key)
	}
}

func BenchmarkAesGCMDecrypt(b *testing.B) {
	plainText := "hello world, benchmark data"
	key := "aa125678aa125678"
	encrypted, _ := xcryption.AesGCMEncrypt(plainText, key)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = xcryption.AesGCMDecrypt(encrypted, key)
	}
}
