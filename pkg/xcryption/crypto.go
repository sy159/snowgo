package xcryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Sha256 Sha256加密
func Sha256(s string) string {
	m := sha256.New()
	m.Write([]byte(s))
	res := hex.EncodeToString(m.Sum(nil))
	return res
}

// HashPassword 生成密码哈希（自动加盐）COST是一个介于4到31之间的整数，更高的值表示更高的计算成本
func HashPassword(password string) (string, error) {
	bytesPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytesPwd), err
}

// CheckPassword 验证密码
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// PKCS7填充，PKCS5就是blockSize固定为8
func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

// PKCS7取出填充
func pkcs7UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("密文为空")
	}
	unPadding := int(origData[length-1])
	if unPadding > length || unPadding > aes.BlockSize {
		return nil, errors.New("无效的填充")
	}
	return origData[:(length - unPadding)], nil
}

// AesCBCEncrypt aes cbc模式加密,使用base64编码更直观 key长度(16，24，32)执行AES-128, AES-192, AES-256算法，IV随机生成
func AesCBCEncrypt(plainText, key string) (string, error) {
	// 检查密钥长度是否有效
	keyByte := []byte(key)
	if len(keyByte) != 16 && len(keyByte) != 24 && len(keyByte) != 32 {
		return "", fmt.Errorf("无效的密钥长度：必须为 16, 24 或 32 字节，当前为 %d", len(keyByte))
	}

	// 创建 AES 密码块
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", errors.Wrap(err, "创建 AES 密码块失败")
	}

	// 填充明文
	plainByte := pkcs7Padding([]byte(plainText), block.BlockSize())

	// 生成随机 IV
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return "", errors.Wrap(err, "生成 IV 失败")
	}

	// 使用 CBC 模式加密
	blockMode := cipher.NewCBCEncrypter(block, iv)
	cipherText := make([]byte, len(plainByte))
	blockMode.CryptBlocks(cipherText, plainByte)

	// 拼接IV和密文并返回Base64编码；拼接方式: [IV | 密文]
	cipherText = append(iv, cipherText...)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// AesCBCDecrypt aes cbc解密
func AesCBCDecrypt(cipherText, key string) (string, error) {
	// 解码 Base64 密文
	cipherByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", errors.Wrap(err, "解码 Base64 密文失败")
	}

	// 检查密钥长度是否有效
	keyByte := []byte(key)
	if len(keyByte) != 16 && len(keyByte) != 24 && len(keyByte) != 32 {
		return "", fmt.Errorf("无效的密钥长度：必须为 16, 24 或 32 字节，当前为 %d", len(keyByte))
	}

	// 创建 AES 密码块
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", errors.Wrap(err, "创建 AES 密码块失败")
	}

	blockSize := block.BlockSize()
	if len(cipherByte) < blockSize {
		return "", errors.New("密文长度不足，无法提取 IV")
	}

	// 分离 IV 和密文
	iv := cipherByte[:blockSize]
	cipherByte = cipherByte[blockSize:]

	// 使用 CBC 模式解密
	blockMode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherByte))
	blockMode.CryptBlocks(plainText, cipherByte)

	// 去除填充
	plainText, err = pkcs7UnPadding(plainText)
	if err != nil {
		return "", errors.Wrap(err, "去除填充失败")
	}

	return string(plainText), nil
}
