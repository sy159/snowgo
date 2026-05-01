package xcryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

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

// AesGCMEncrypt AES-GCM 模式加密，使用 base64 编码
func AesGCMEncrypt(plainText, key string) (string, error) {
	keyByte := []byte(key)
	if len(keyByte) != 16 && len(keyByte) != 24 && len(keyByte) != 32 {
		return "", fmt.Errorf("无效的密钥长度：必须为 16, 24 或 32 字节，当前为 %d", len(keyByte))
	}

	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", fmt.Errorf("创建 AES 密码块失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	// GCM 推荐使用 12 字节 nonce，由 crypto/rand 生成保证不可预测
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// Seal 将 nonce 前置拼接: [nonce | 密文+tag]
	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// AesGCMDecrypt AES-GCM 模式解密
// 密文结构: base64([nonce(12字节) | 密文 | tag(16字节)])
func AesGCMDecrypt(cipherText, key string) (string, error) {
	cipherByte, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("解码 Base64 密文失败: %w", err)
	}

	keyByte := []byte(key)
	if len(keyByte) != 16 && len(keyByte) != 24 && len(keyByte) != 32 {
		return "", fmt.Errorf("无效的密钥长度：必须为 16, 24 或 32 字节，当前为 %d", len(keyByte))
	}

	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", fmt.Errorf("创建 AES 密码块失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherByte) < nonceSize {
		return "", fmt.Errorf("密文长度不足，无法提取 nonce")
	}

	// 分离 nonce 和密文+tag
	nonce := cipherByte[:nonceSize]
	cipherByte = cipherByte[nonceSize:]

	// Open 内部验证 tag，密文被篡改或密钥错误时返回 error
	plainText, err := aesGCM.Open(nil, nonce, cipherByte, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密文可能被篡改或密钥错误）: %w", err)
	}

	return string(plainText), nil
}
