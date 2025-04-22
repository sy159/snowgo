package jwt

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

const (
	accessType  = "access"
	refreshType = "refresh"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
	ErrInvalidToken     = errors.New("invalid token")
)

type Config struct {
	JwtSecret             string
	Issuer                string
	AccessExpirationTime  int // 单位：分钟
	RefreshExpirationTime int // 单位：分钟
}

type Manager struct {
	jwtConf *Config
}

func NewJwtManager(conf *Config) *Manager {
	return &Manager{jwtConf: conf}
}

// Claims 自定义声明
type Claims struct {
	GrantType string `json:"grant_type"` // 授权类型，区分 accessToken 与 refreshToken
	UserId    int64  `json:"user_id"`
	Username  string `json:"username"`
	//Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 创建 access token
func (m *Manager) GenerateAccessToken(userId int64, username string) (string, error) {
	now := time.Now()
	exp := now.Add(time.Duration(m.jwtConf.AccessExpirationTime) * time.Minute)
	exp = time.Unix(exp.Unix(), 0) // 去除纳秒
	accessClaims := Claims{
		GrantType: accessType,
		UserId:    userId,
		Username:  username,
		//Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    m.jwtConf.Issuer,
			Subject:   strconv.FormatInt(userId, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // jti
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	return token.SignedString([]byte(m.jwtConf.JwtSecret))
}

// GenerateRefreshToken 创建 refresh token
func (m *Manager) GenerateRefreshToken(userId int64, username string) (string, error) {
	now := time.Now()
	exp := now.Add(time.Duration(m.jwtConf.RefreshExpirationTime) * time.Minute)
	exp = time.Unix(exp.Unix(), 0) // 去除纳秒
	refreshClaims := Claims{
		GrantType: refreshType,
		UserId:    userId,
		Username:  username,
		//Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    m.jwtConf.Issuer,
			Subject:   fmt.Sprintf("snowgo: %d", userId),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // jti
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	return token.SignedString([]byte(m.jwtConf.JwtSecret))
}

// GenerateTokens 创建一对 access + refresh token
func (m *Manager) GenerateTokens(userId int64, username string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(userId, username)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Access Token error")
	}
	refreshToken, err = m.GenerateRefreshToken(userId, username)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token error")
	}
	return accessToken, refreshToken, nil
}

// ParseToken 解析 JWT token
func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	// 解析并校验token
	tokenClaims, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtConf.JwtSecret), nil
	}) // 自定义手动校验 jwt.WithoutClaimsValidation()这个是跳过所有的校验
	if err != nil {
		return nil, err
	}
	claims, ok := tokenClaims.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// RefreshTokens 用 refresh token 刷新令牌对
func (m *Manager) RefreshTokens(refreshToken string) (accessToken, newRefreshToken string, err error) {
	claims, err := m.ParseToken(refreshToken)
	if err != nil {
		return "", "", errors.Wrap(err, "Parse Refresh Token error")
	}

	// 手动检查刷新令牌是否过期
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return "", "", ErrTokenExpired
	}
	// 检查令牌类型是否为 refresh
	if !claims.IsRefreshToken() {
		return "", "", ErrInvalidTokenType
	}

	// 生成新的刷新令牌(这里如果重新生成，refresh token的过期时间又要重新算，如果沿用以前的，就是严格按照refresh token过期时间来)
	newRefreshToken, err = m.GenerateRefreshToken(claims.UserId, claims.Username)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Refresh Token error")
	}
	// 生成新的访问令牌
	accessToken, err = m.GenerateAccessToken(claims.UserId, claims.Username)
	if err != nil {
		return "", "", errors.Wrap(err, "Generate Access Token error")
	}
	return accessToken, newRefreshToken, nil
}

// IsAccessToken 判断是否是 access token
func (cm *Claims) IsAccessToken() bool {
	return cm.GrantType == accessType
}

// IsRefreshToken 判断是否是 access token
func (cm *Claims) IsRefreshToken() bool {
	return cm.GrantType == refreshType
}

// ValidAccessToken 校验 access token（类型 + 时间 + issuer）
func (cm *Claims) ValidAccessToken() error {
	// 检查到期时间 解析的时候已经校验了
	//if cm.ExpiresAt.Time.Before(time.Now()) {
	//	return ErrTokenExpired
	//}
	// 检查令牌类型
	if cm.GrantType != accessType {
		return ErrInvalidTokenType
	}
	return nil
}
