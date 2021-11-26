package jwt

import (
	"snowgo/config"
	"snowgo/utils/logger"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	accessType  = "access"
	refreshType = "refresh"
)

type Claims struct {
	GrantType string `json:"grant_type"` // 授权类型，区分accessToken跟refreshToken
	UserId    uint   `json:"user_id"`
	Username  string `json:"username"`
	jwt.StandardClaims
}

// GenerateAccessToken 生成访问token
func GenerateAccessToken(userId uint, username string) (string, error) {
	accessClaims := Claims{
		GrantType: accessType,
		UserId:    userId,
		Username:  username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(config.JwtConf.AccessExpirationTime) * time.Minute).Unix(),
			Issuer:    config.JwtConf.Issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := tokenClaims.SignedString([]byte(config.JwtConf.JwtSecret))
	if err != nil {
		logger.Errorf("Generate Access Token is err: %s", err)
	}

	return accessToken, err
}

// GenerateRefreshToken 生成refresh_token
func GenerateRefreshToken(userId uint, username string) (string, error) {
	accessClaims := Claims{
		GrantType: refreshType,
		UserId:    userId,
		Username:  username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(config.JwtConf.RefreshExpirationTime) * time.Minute).Unix(),
			Issuer:    config.JwtConf.Issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken, err := tokenClaims.SignedString([]byte(config.JwtConf.JwtSecret))

	return refreshToken, err
}

// ParseToken 解析token
func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JwtConf.JwtSecret), nil
	})

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, err
}

// CheckTypeByClaims 检查token的类型是不是访问token，而不是用刷新token来请求
func (cm *Claims) CheckTypeByClaims() bool {
	if cm.GrantType != accessType {
		return false
	}
	return true
}

// CheckTimeByClaims 校验token是否过期
func (cm *Claims) CheckTimeByClaims() bool {
	if cm.StandardClaims.ExpiresAt < time.Now().Unix() {
		return false
	}
	return true
}
