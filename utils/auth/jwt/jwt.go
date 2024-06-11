package jwt

import (
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"snowgo/config"
	"snowgo/utils/logger"
	"time"
)

const (
	accessType  = "access"
	refreshType = "refresh"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
)

type Claims struct {
	GrantType string `json:"grant_type"` // 授权类型，区分accessToken跟refreshToken
	UserId    uint   `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	jwt.StandardClaims
}

// GenerateAccessToken 生成访问token
func GenerateAccessToken(userId uint, username, role string) (string, error) {
	accessClaims := Claims{
		GrantType: accessType,
		UserId:    userId,
		Username:  username,
		Role:      role,
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
func GenerateRefreshToken(userId uint, username string, role string) (string, error) {
	accessClaims := Claims{
		GrantType: refreshType,
		UserId:    userId,
		Username:  username,
		Role:      role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(config.JwtConf.RefreshExpirationTime) * time.Minute).Unix(),
			Issuer:    config.JwtConf.Issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken, err := tokenClaims.SignedString([]byte(config.JwtConf.JwtSecret))

	return refreshToken, err
}

// GenerateTokens 生成访问令牌和刷新令牌
func GenerateTokens(userId uint, username string, role string) (accessToken, refreshToken string, err error) {
	// 生成访问令牌
	accessToken, err = GenerateAccessToken(userId, username, role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Access Token is err")
	}

	// 生成刷新令牌
	refreshToken, err = GenerateRefreshToken(userId, username, role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token is err")
	}

	return accessToken, refreshToken, nil
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

// RefreshTokens 根据刷新令牌生成新的刷新令牌和访问令牌
func RefreshTokens(refreshToken string) (newRefreshToken, accessToken string, err error) {
	// 解析刷新令牌
	claims, err := ParseToken(refreshToken)
	if err != nil {
		return "", "", errors.Wrap(err, "Parse Refresh Token Error")
	}

	// 检查刷新令牌
	if err := claims.Valid(); err != nil {
		return "", "", errors.Wrap(err, "Claims Valid Error")
	}

	// 生成新的刷新令牌(这里如果重新生成，refresh token的过期时间又要重新算，如果沿用以前的，就是严格按照refresh token过期时间来)
	newRefreshToken, err = GenerateRefreshToken(claims.UserId, claims.Username, claims.Role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token Error")
	}

	// 生成新的访问令牌
	accessToken, err = GenerateAccessToken(claims.UserId, claims.Username, claims.Role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token Error")
	}

	return newRefreshToken, accessToken, nil
}

// IsAccessToken 检查token的类型是不是访问token，而不是用刷新token来请求
func (cm *Claims) IsAccessToken() bool {
	if cm.GrantType != accessType {
		return false
	}
	return true
}

// ValidAccessToken 检查token的类型和有效期
func (cm *Claims) ValidAccessToken() error {
	// 检查到期时间 cm.Valid()默认的校验
	if cm.ExpiresAt < time.Now().Unix() {
		return ErrTokenExpired
	}
	// 检查令牌类型
	if cm.GrantType != accessType {
		return ErrInvalidTokenType
	}
	return nil
}
