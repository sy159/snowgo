package jwt_test

import (
	"fmt"
	"github.com/pkg/errors"
	"snowgo/pkg/xauth/jwt"
	"strings"
	"testing"
	"time"
)

func TestJwt(t *testing.T) {
	var userId int64 = 1
	username := "test"

	jwtManager, _ := jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             "Tphdi%Aapi5iXsX67F7MX5ZRJxZF*6wK",
		Issuer:                "test-snow",
		AccessExpirationTime:  10,
		RefreshExpirationTime: 30,
	})

	t.Run("generate and parse tokens", func(t *testing.T) {
		tokenPair, err := jwtManager.GenerateTokens(userId, username)
		if err != nil {
			t.Fatalf("GenerateTokens error: %v", err)
		}
		fmt.Printf("AccessToken: %s\nRefreshToken: %s\n", tokenPair.AccessToken, tokenPair.RefreshToken)

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
		authHeader := fmt.Sprintf("Bearer %s", tokenPair.AccessToken)
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
			AccessExpirationTime:  0,
			RefreshExpirationTime: 0,
		})
		accessToken, _, err := expiredManager.GenerateAccessToken(userId, username, "123")
		if err != nil {
			t.Fatalf("generate expired access token error: %v", err)
		}

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

		fmt.Printf("New AccessToken: %s\nNew RefreshToken: %s\n", newPair.AccessToken, newPair.RefreshToken)
	})

	t.Run("refresh token reuse old access token", func(t *testing.T) {
		// 测试旧 access token 仍然可以解析，但 sessionId 指向旧 refresh token
		oldPair, _ := jwtManager.GenerateTokens(userId, username)
		time.Sleep(1 * time.Second)
		claims, err := jwtManager.ParseToken(oldPair.AccessToken)
		if err != nil {
			t.Fatalf("Parse old access token error: %v", err)
		}
		if claims.SessionId != oldPair.RefreshToken[:36] && claims.SessionId == "" {
			// 这里只是简单检查格式
			fmt.Println("Old access token still valid for session id")
		}
	})

}
