package jwt_test

import (
	"fmt"
	"github.com/pkg/errors"
	"snowgo/pkg/xauth/jwt"
	"strings"
	"testing"
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

	t.Run("jwt token", func(t *testing.T) {
		token, err := jwtManager.GenerateTokens(userId, username)
		if err != nil {
			t.Fatalf("get refresh token is err: %v", err)
		}
		fmt.Printf("refresh token is: %v\naccess token is: %v\n", token.RefreshToken, token.AccessToken)

		parseToken, err := jwtManager.ParseToken(token.AccessToken)
		if err != nil {
			t.Fatalf("get token info is err: %v", err)
		}
		fmt.Printf("token info is: %+v\n", parseToken)
		if parseToken.UserId != userId || parseToken.Username != username {
			t.Fatal("token info is err")
		}

		//time.Sleep(1 * time.Second)
		err = parseToken.ValidAccessToken()
		fmt.Println(err)

		newToken, err := jwtManager.RefreshTokens(token.RefreshToken)
		fmt.Printf("new refresh token is: %v\naccess token is: %v\nrefresh token is err: %v\n", newToken.RefreshToken, newToken.AccessToken, err)
	})

	t.Run("jwt auth", func(t *testing.T) {
		token, err := jwtManager.GenerateTokens(userId, username)
		if err != nil {
			t.Fatalf("get refresh token is err: %v", err)
		}
		fmt.Printf("refresh token is: %v\naccess token is: %v\n", token.RefreshToken, token.AccessToken)
		authHeader := fmt.Sprintf("Bearer %v", token.AccessToken)
		if authHeader == "" {
			t.Fatalf("header is empty")
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			t.Fatalf("token format xerror")
		}
		// parts[1]是获取到的tokenString，我们使用之前定义好的解析JWT的函数来解析它
		mc, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			t.Fatalf("get token info is err: %v", err)
		}

		//time.Sleep(time.Second)
		// 检查token的过期时间，以及type
		if err := mc.ValidAccessToken(); err != nil {
			if errors.Is(err, jwt.ErrInvalidTokenType) {
				fmt.Printf("invalid token type")
			}
			fmt.Printf("token has expired")
		}
	})

	t.Run("token expired", func(t *testing.T) {
		expiredManager, _ := jwt.NewJwtManager(&jwt.Config{
			JwtSecret:             "Tphdi%Aapi5iXsX67F7MX5ZRJxZF*6wK",
			Issuer:                "test-snow",
			AccessExpirationTime:  0, // 已经过期
			RefreshExpirationTime: 0,
		})

		accessToken, _, err := expiredManager.GenerateAccessToken(userId, username)
		if err != nil {
			t.Fatalf("generate expired token error: %v\n", err)
		}

		_, err = expiredManager.ParseToken(accessToken)
		if err == nil {
			t.Fatalf("parse expired token error: %v\n", err)
		}
		fmt.Println(err)
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := jwtManager.ParseToken("this.is.not.jwt")
		if err == nil {
			t.Fatal("expected error on invalid token format, got nil")
		}
	})

	t.Run("invalid token type", func(t *testing.T) {
		refreshToken, _, err := jwtManager.GenerateRefreshToken(userId, username)
		if err != nil {
			t.Fatalf("generate refresh token error: %v", err)
		}
		claims, err := jwtManager.ParseToken(refreshToken)
		if err != nil {
			t.Fatalf("parse token error: %v", err)
		}
		if err := claims.ValidAccessToken(); !errors.Is(err, jwt.ErrInvalidTokenType) {
			t.Fatalf("expected invalid token type error, got: %v", err)
		}
	})

	t.Run("refresh token with access token", func(t *testing.T) {
		refreshToken, _, err := jwtManager.GenerateRefreshToken(userId, username)
		if err != nil {
			t.Fatalf("generate access token error: %v", err)
		}
		token, err := jwtManager.RefreshTokens(refreshToken)
		if err != nil {
			t.Fatalf("generate refresh token error: %v", err)
		}
		fmt.Println(token.AccessToken, token.RefreshToken)
	})

}
