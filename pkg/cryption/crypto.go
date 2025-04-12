package cryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// Md5 md5加密
func Md5(s string) string {
	m := md5.New()
	m.Write([]byte(s))
	res := hex.EncodeToString(m.Sum(nil))
	return res
}

// Sha256 Sha256加密
func Sha256(s string) string {
	m := sha256.New()
	m.Write([]byte(s))
	res := hex.EncodeToString(m.Sum(nil))
	return res
}

// PKCS7填充，PKCS5就是blockSize固定为8
func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

// PKCS7取出填充
func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

// AesCBCEncrypt aes cbc模式加密,使用base64编码更直观 key长度(16，24，32)执行AES-128, AES-192, AES-256算法
func AesCBCEncrypt(plainText, key string) (string, error) {
	plainByte, keyByte := []byte(plainText), []byte(key)
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	plainByte = pkcs7Padding(plainByte, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, keyByte[:blockSize])
	cipherText := make([]byte, len(plainByte))
	blockMode.CryptBlocks(cipherText, plainByte)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// AesCBCDecrypt aes cbc解密
func AesCBCDecrypt(cipherText, key string) (string, error) {
	cipherByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	keyByte := []byte(key)
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, keyByte[:blockSize])
	plainText := make([]byte, len(cipherByte))
	blockMode.CryptBlocks(plainText, cipherByte)
	plainText = pkcs7UnPadding(plainText)
	return string(plainText), nil
}
