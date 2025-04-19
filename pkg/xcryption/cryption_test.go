package xcryption_test

import (
	"fmt"
	"snowgo/pkg/xcryption"
	"testing"
)

func TestCrypto(t *testing.T) {
	plainText, key := "hello world", "aa125678aa125678" // AesKey:"1234567890123456", // 必须是 16, 24 或 32 字节
	t.Run("aes cbc encrypt/decrypt", func(t *testing.T) {
		encrypt, err := xcryption.AesCBCEncrypt(plainText, key)
		fmt.Println(encrypt, err)
		if err != nil {
			t.Error(err)
		}

		decrypt, err := xcryption.AesCBCDecrypt(encrypt, key)
		fmt.Println(decrypt, err)
		if err != nil {
			t.Error(err)
		}
		if plainText != decrypt {
			t.Error("decrypt xerror")
		}
	})
}

func TestEncode(t *testing.T) {
	var id uint = 11111
	t.Run("encode/decode", func(t *testing.T) {
		code := xcryption.Id2Code(id, 8)
		fmt.Println(code)
		code2Id, err := xcryption.Code2Id(code)
		if err != nil {
			t.Error(err)
		}
		if code2Id != id {
			t.Error("code xerror")
		}
	})
}

func TestHashPassword(t *testing.T) {
	pwd := "123456"
	t.Run("hash pwd", func(t *testing.T) {
		hashPwd, err := xcryption.HashPassword(pwd)
		fmt.Println(hashPwd, err)
		if err != nil {
			t.Error(err)
		}
		isSuccess := xcryption.CheckPassword(hashPwd, pwd)
		if !isSuccess {
			t.Error("hash password error")
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

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		code := xcryption.Id2Code(uint(i), 8)
		id, err := xcryption.Code2Id(code)
		if err != nil {
			b.Error(err)
		}
		if uint(i) != id {
			b.Error("code xerror")
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
