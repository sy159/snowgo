package cryption_test

import (
	"fmt"
	"snowgo/utils/cryption"
	"testing"
)

func TestCrypto(t *testing.T) {
	plainText, key := "hello world", "aa125678aa125678" // AesKey:"1234567890123456", // 必须是 16, 24 或 32 字节
	t.Run("aes cbc encrypt/decrypt", func(t *testing.T) {
		encrypt, err := cryption.AesCBCEncrypt(plainText, key)
		fmt.Println(encrypt, err)
		if err != nil {
			t.Error(err)
		}

		decrypt, err := cryption.AesCBCDecrypt(encrypt, key)
		fmt.Println(decrypt, err)
		if err != nil {
			t.Error(err)
		}
		if plainText != decrypt {
			t.Error("decrypt error")
		}
	})
}

func TestEncode(t *testing.T) {
	var id uint
	id = 11111
	t.Run("encode/decode", func(t *testing.T) {
		code := cryption.Id2Code(id, 8)
		fmt.Println(code)
		code2Id, err := cryption.Code2Id(code)
		if err != nil {
			t.Error(err)
		}
		if code2Id != id {
			t.Error("code error")
		}
	})
}

func BenchmarkCrypto(b *testing.B) {
	plainText, key := "hello world", "aa125678aa125678"
	for i := 0; i < b.N; i++ {
		encrypt, err := cryption.AesCBCEncrypt(plainText, key)
		if err != nil {
			b.Error(err)
		}
		decrypt, err := cryption.AesCBCDecrypt(encrypt, key)
		if err != nil {
			b.Error(err)
		}
		if plainText != decrypt {
			b.Error("decrypt error")
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		code := cryption.Id2Code(uint(i), 8)
		id, err := cryption.Code2Id(code)
		if err != nil {
			b.Error(err)
		}
		if uint(i) != id {
			b.Error("code error")
		}
	}
}
