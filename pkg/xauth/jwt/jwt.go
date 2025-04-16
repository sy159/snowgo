package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"snowgo/config"
	"snowgo/pkg/xlogger"
)

const (
	accessType  = "access"
	refreshType = "refresh"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
)

// Claims 定义 JWT 的自定义 claims，内嵌 jwt.RegisteredClaims
type Claims struct {
	GrantType string `json:"grant_type"` // 授权类型，区分 accessToken 与 refreshToken
	UserId    uint   `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问 Token
func GenerateAccessToken(userId uint, username, role string) (string, error) {
	expiresAt := time.Now().Add(time.Duration(config.JwtConf.AccessExpirationTime) * time.Minute)

	accessClaims := Claims{
		GrantType: accessType,
		UserId:    userId,
		Username:  username,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    config.JwtConf.Issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := tokenClaims.SignedString([]byte(config.JwtConf.JwtSecret))
	if err != nil {
		xlogger.Errorf("Generate Access Token error: %s", err)
	}

	return accessToken, err
}

// GenerateRefreshToken 生成刷新 Token
func GenerateRefreshToken(userId uint, username, role string) (string, error) {
	expiresAt := time.Now().Add(time.Duration(config.JwtConf.RefreshExpirationTime) * time.Minute)

	refreshClaims := Claims{
		GrantType: refreshType,
		UserId:    userId,
		Username:  username,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    config.JwtConf.Issuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenClaims.SignedString([]byte(config.JwtConf.JwtSecret))

	return refreshToken, err
}

// GenerateTokens 同时生成访问令牌和刷新令牌
func GenerateTokens(userId uint, username, role string) (accessToken, refreshToken string, err error) {
	accessToken, err = GenerateAccessToken(userId, username, role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Access Token error")
	}

	// 生成刷新令牌
	refreshToken, err = GenerateRefreshToken(userId, username, role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token error")
	}

	return accessToken, refreshToken, nil
}

// ParseToken 解析 token，并返回自定义的 Claims
func ParseToken(tokenStr string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JwtConf.JwtSecret), nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Token parsing error")
	}
	claims, ok := tokenClaims.Claims.(*Claims)
	if !ok || !tokenClaims.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// RefreshTokens 根据刷新令牌生成新的刷新令牌和访问令牌
func RefreshTokens(refreshToken string) (newRefreshToken, accessToken string, err error) {
	// 解析刷新令牌
	claims, err := ParseToken(refreshToken)
	if err != nil {
		return "", "", errors.Wrap(err, "Parse Refresh Token error")
	}

	// 手动检查刷新令牌是否过期
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return "", "", ErrTokenExpired
	}
	// 检查令牌类型是否为 refresh
	if claims.GrantType != refreshType {
		return "", "", ErrInvalidTokenType
	}

	// 生成新的刷新令牌(这里如果重新生成，refresh token的过期时间又要重新算，如果沿用以前的，就是严格按照refresh token过期时间来)
	newRefreshToken, err = GenerateRefreshToken(claims.UserId, claims.Username, claims.Role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token error")
	}

	// 生成新的访问令牌
	accessToken, err = GenerateAccessToken(claims.UserId, claims.Username, claims.Role)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Access Token error")
	}

	return newRefreshToken, accessToken, nil
}

// IsAccessToken 检查当前 Claims 是否为访问令牌
func (cm *Claims) IsAccessToken() bool {
	return cm.GrantType == accessType
}

// ValidAccessToken 检查访问令牌的有效性（包括过期时间和令牌类型）
func (cm *Claims) ValidAccessToken() error {
	// 检查到期时间 cm.Valid()默认的校验
	if cm.ExpiresAt.Time.Before(time.Now()) {
		return ErrTokenExpired
	}
	// 检查令牌类型
	if cm.GrantType != accessType {
		return ErrInvalidTokenType
	}
	return nil
}
