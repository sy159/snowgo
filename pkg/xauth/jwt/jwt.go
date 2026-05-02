package jwt

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

type Token struct {
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	AccessExpire  time.Time `json:"access_expire"`
	RefreshExpire time.Time `json:"refresh_expire"`
}

type Config struct {
	JwtSecret             string
	Issuer                string
	AccessExpirationTime  time.Duration
	RefreshExpirationTime time.Duration
}

type Manager struct {
	jwtConf *Config
}

func NewJwtManager(conf *Config) (*Manager, error) {
	if conf == nil {
		return nil, errors.New("jwt config is nil")
	}
	if len(conf.JwtSecret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 characters for HS256")
	}
	if conf.Issuer == "" {
		return nil, errors.New("jwt issuer required")
	}
	if conf.AccessExpirationTime <= 0 {
		return nil, errors.New("access expiration time must be > 0")
	}
	if conf.RefreshExpirationTime <= 0 {
		return nil, errors.New("refresh expiration time must be > 0")
	}
	return &Manager{jwtConf: conf}, nil
}

// Claims 自定义声明
type Claims struct {
	GrantType string `json:"grant_type"` // 授权类型，区分 accessToken 与 refreshToken
	UserId    int32  `json:"user_id"`
	Username  string `json:"username"`
	SessionId string `json:"session_id"`
	//Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 创建 access token
func (m *Manager) GenerateAccessToken(userId int32, username, refreshJti string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.jwtConf.AccessExpirationTime)
	exp = time.Unix(exp.Unix(), 0) // 去除纳秒
	accessClaims := Claims{
		GrantType: accessType,
		UserId:    userId,
		Username:  username,
		SessionId: refreshJti, // refresh id，用于关联信息
		//Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    m.jwtConf.Issuer,
			Subject:   strconv.FormatInt(int64(userId), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // jti
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenString, err := token.SignedString([]byte(m.jwtConf.JwtSecret))
	return tokenString, exp, err
}

// GenerateRefreshToken 创建 refresh token
func (m *Manager) GenerateRefreshToken(userId int32, username string) (string, string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.jwtConf.RefreshExpirationTime)
	exp = time.Unix(exp.Unix(), 0) // 去除纳秒
	jti := uuid.New().String()
	refreshClaims := Claims{
		GrantType: refreshType,
		UserId:    userId,
		Username:  username,
		SessionId: jti,
		//Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    m.jwtConf.Issuer,
			Subject:   strconv.FormatInt(int64(userId), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti, // jti
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	tokenString, err := token.SignedString([]byte(m.jwtConf.JwtSecret))
	return tokenString, jti, exp, err
}

// GenerateTokens 创建一对 access + refresh token
func (m *Manager) GenerateTokens(userId int32, username string) (token *Token, err error) {
	refreshToken, refreshJti, refreshExp, err := m.GenerateRefreshToken(userId, username)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token error: %w", err)
	}
	accessToken, accessExp, err := m.GenerateAccessToken(userId, username, refreshJti)
	if err != nil {
		return nil, fmt.Errorf("generate access token error: %w", err)
	}
	return &Token{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpire:  accessExp,
		RefreshExpire: refreshExp,
	}, nil
}

// ParseToken 解析 JWT token
func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		return []byte(m.jwtConf.JwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.jwtConf.Issuer),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

// RefreshTokens 用 refresh token 刷新令牌对
func (m *Manager) RefreshTokens(refreshToken string) (token *Token, err error) {
	claims, err := m.ParseToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token error: %w", err)
	}

	// 检查令牌类型是否为 refresh
	if !claims.IsRefreshToken() {
		return nil, ErrInvalidTokenType
	}

	// 生成新的刷新令牌
	newRefreshToken, refreshJti, refreshExp, err := m.GenerateRefreshToken(claims.UserId, claims.Username)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token error: %w", err)
	}
	// 生成新的访问令牌
	accessToken, accessExp, err := m.GenerateAccessToken(claims.UserId, claims.Username, refreshJti)
	if err != nil {
		return nil, fmt.Errorf("generate access token error: %w", err)
	}
	return &Token{
		AccessToken:   accessToken,
		RefreshToken:  newRefreshToken,
		AccessExpire:  accessExp,
		RefreshExpire: refreshExp,
	}, nil
}

// IsAccessToken 判断是否是 access token
func (cm *Claims) IsAccessToken() bool {
	return cm.GrantType == accessType
}

// IsRefreshToken 判断是否是 refresh token
func (cm *Claims) IsRefreshToken() bool {
	return cm.GrantType == refreshType
}

// ValidAccessToken 校验 access token 的类型
func (cm *Claims) ValidAccessToken() error {
	// 检查令牌类型
	if cm.GrantType != accessType {
		return ErrInvalidTokenType
	}
	return nil
}
