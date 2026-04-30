package jwt_test

import (
	"errors"
	"snowgo/pkg/xauth/jwt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestJwt(t *testing.T) {
	var userId int32 = 1
	username := "test"

	jwtManager, _ := jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             "Tphdi%Aapi5iXsX67F7MX5ZRJxZF*6wK",
		Issuer:                "test-snow",
		AccessExpirationTime:  10 * time.Minute,
		RefreshExpirationTime: 30 * time.Minute,
	})

	t.Run("NewJwtManager validation errors", func(t *testing.T) {
		tests := []struct {
			name   string
			conf   *jwt.Config
			wantOk bool
		}{
			{"nil config", nil, false},
			{"secret too short", &jwt.Config{JwtSecret: "short", Issuer: "test", AccessExpirationTime: time.Minute, RefreshExpirationTime: time.Minute}, false},
			{"empty issuer", &jwt.Config{JwtSecret: "longsecret123456", Issuer: "", AccessExpirationTime: time.Minute, RefreshExpirationTime: time.Minute}, false},
			{"zero access expiration", &jwt.Config{JwtSecret: "longsecret123456", Issuer: "test", AccessExpirationTime: 0, RefreshExpirationTime: time.Minute}, false},
			{"zero refresh expiration", &jwt.Config{JwtSecret: "longsecret123456", Issuer: "test", AccessExpirationTime: time.Minute, RefreshExpirationTime: 0}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := jwt.NewJwtManager(tt.conf)
				if tt.wantOk && err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if !tt.wantOk && err == nil {
					t.Fatal("expected error, got nil")
				}
			})
		}
	})

	t.Run("generate and parse tokens", func(t *testing.T) {
		tokenPair, err := jwtManager.GenerateTokens(userId, username)
		if err != nil {
			t.Fatalf("GenerateTokens error: %v", err)
		}
		t.Logf("AccessToken: %s\nRefreshToken: %s", tokenPair.AccessToken, tokenPair.RefreshToken)

		accessClaims, err := jwtManager.ParseToken(tokenPair.AccessToken)
		if err != nil {
			t.Fatalf("Parse access token error: %v", err)
		}

		if accessClaims.UserId != userId || accessClaims.Username != username {
			t.Fatalf("access token claims mismatch")
		}

		// 校验 access token 类型
		if err := accessClaims.ValidAccessToken(); err != nil {
			t.Fatalf("ValidAccessToken error: %v", err)
		}

		// 校验 SessionId 对应 refresh token jti
		refreshClaims, err := jwtManager.ParseToken(tokenPair.RefreshToken)
		if err != nil {
			t.Fatalf("Parse refresh token error: %v", err)
		}
		if accessClaims.SessionId != refreshClaims.ID {
			t.Fatalf("access token session_id not match refresh token jti")
		}
	})

	t.Run("access token validation from header", func(t *testing.T) {
		tokenPair, _ := jwtManager.GenerateTokens(userId, username)
		authHeader := "Bearer " + tokenPair.AccessToken
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			t.Fatalf("invalid Authorization header format")
		}
		claims, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			t.Fatalf("ParseToken from header error: %v", err)
		}
		if err := claims.ValidAccessToken(); err != nil {
			t.Fatalf("access token validation failed: %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expiredManager, _ := jwt.NewJwtManager(&jwt.Config{
			JwtSecret:             "Tphdi%Aapi5iXsX67F7MX5ZRJxZF*6wK",
			Issuer:                "test-snow",
			AccessExpirationTime:  100 * time.Millisecond,
			RefreshExpirationTime: 100 * time.Millisecond,
		})
		accessToken, _, err := expiredManager.GenerateAccessToken(userId, username, "123")
		if err != nil {
			t.Fatalf("generate expired access token error: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
		_, err = expiredManager.ParseToken(accessToken)
		if !errors.Is(err, jwt.ErrTokenExpired) {
			t.Fatalf("expected ErrTokenExpired, got: %v", err)
		}
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := jwtManager.ParseToken("invalid.token.format")
		if err == nil {
			t.Fatal("expected error on invalid token format, got nil")
		}
	})

	t.Run("invalid access token type", func(t *testing.T) {
		refreshToken, _, _, err := jwtManager.GenerateRefreshToken(userId, username)
		if err != nil {
			t.Fatalf("generate refresh token error: %v", err)
		}
		claims, _ := jwtManager.ParseToken(refreshToken)
		if err := claims.ValidAccessToken(); !errors.Is(err, jwt.ErrInvalidTokenType) {
			t.Fatalf("expected ErrInvalidTokenType, got: %v", err)
		}
	})

	t.Run("refresh tokens generate new access token", func(t *testing.T) {
		tokenPair, _ := jwtManager.GenerateTokens(userId, username)
		newPair, err := jwtManager.RefreshTokens(tokenPair.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshTokens error: %v", err)
		}

		newAccessClaims, _ := jwtManager.ParseToken(newPair.AccessToken)
		if newAccessClaims.SessionId == "" {
			t.Fatal("new access token missing session_id")
		}

		newRefreshClaims, _ := jwtManager.ParseToken(newPair.RefreshToken)
		if newAccessClaims.SessionId != newRefreshClaims.ID {
			t.Fatal("new access token session_id not match new refresh token jti")
		}

		t.Logf("New AccessToken: %s\nNew RefreshToken: %s", newPair.AccessToken, newPair.RefreshToken)
	})

	t.Run("refresh token reuse old access token", func(t *testing.T) {
		// 测试旧 access token 仍然可以解析，sessionId 非空
		oldPair, _ := jwtManager.GenerateTokens(userId, username)
		time.Sleep(1 * time.Second)
		claims, err := jwtManager.ParseToken(oldPair.AccessToken)
		if err != nil {
			t.Fatalf("Parse old access token error: %v", err)
		}
		if claims.SessionId == "" {
			t.Fatalf("old access token session_id is empty")
		}
		// 验证 refresh token 的 JTI 与 access token 的 SessionId 一致
		refreshClaims, err := jwtManager.ParseToken(oldPair.RefreshToken)
		if err != nil {
			t.Fatalf("Parse old refresh token error: %v", err)
		}
		if claims.SessionId != refreshClaims.ID {
			t.Fatalf("access token session_id not match refresh token jti")
		}
	})

	t.Run("claims type check", func(t *testing.T) {
		// Test IsAccessToken and IsRefreshToken
		tokenPair, _ := jwtManager.GenerateTokens(userId, username)

		accessClaims, _ := jwtManager.ParseToken(tokenPair.AccessToken)
		if !accessClaims.IsAccessToken() {
			t.Fatal("access token claims should be access type")
		}
		if accessClaims.IsRefreshToken() {
			t.Fatal("access token claims should not be refresh type")
		}

		refreshClaims, _ := jwtManager.ParseToken(tokenPair.RefreshToken)
		if !refreshClaims.IsRefreshToken() {
			t.Fatal("refresh token claims should be refresh type")
		}
		if refreshClaims.IsAccessToken() {
			t.Fatal("refresh token claims should not be access type")
		}
	})

	t.Run("refresh token with access token should fail", func(t *testing.T) {
		tokenPair, _ := jwtManager.GenerateTokens(userId, username)
		// Try to refresh with access token instead of refresh token
		_, err := jwtManager.RefreshTokens(tokenPair.AccessToken)
		if err == nil {
			t.Fatal("expected error when refreshing access token")
		}
		if !errors.Is(err, jwt.ErrInvalidTokenType) {
			t.Fatalf("expected ErrInvalidTokenType, got: %v", err)
		}
	})

	t.Run("refresh with invalid token", func(t *testing.T) {
		_, err := jwtManager.RefreshTokens("not.a.real.token")
		if err == nil {
			t.Fatal("expected error for invalid refresh token")
		}
	})

}

// ========================
// Benchmark
// ========================

var benchMgr *jwt.Manager

func init() {
	benchMgr, _ = jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             "benchmark-secret-key-32bytes!!",
		Issuer:                "bench",
		AccessExpirationTime:  10 * time.Minute,
		RefreshExpirationTime: 30 * time.Minute,
	})
}

func BenchmarkGenerateAccessToken(b *testing.B) {
	var uid int32
	b.RunParallel(func(pb *testing.PB) {
		id := atomic.AddInt32(&uid, 1)
		for pb.Next() {
			_, _, _ = benchMgr.GenerateAccessToken(id, "user", "refresh-jti")
		}
	})
}

func BenchmarkGenerateTokens(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = benchMgr.GenerateTokens(1, "benchuser")
	}
}

func BenchmarkParseToken(b *testing.B) {
	token, _, _ := benchMgr.GenerateAccessToken(1, "benchuser", "jti")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = benchMgr.ParseToken(token)
	}
}

func BenchmarkRefreshTokens(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, refreshToken, _, _ := benchMgr.GenerateRefreshToken(1, "benchuser")
			_, _ = benchMgr.RefreshTokens(refreshToken)
		}
	})
}
