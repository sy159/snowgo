package jwt_test

import (
	"fmt"
	"github.com/pkg/errors"
	"snowgo/pkg/xauth/jwt"
	"strings"
	"testing"
)

func TestJwt(t *testing.T) {
	var userId uint = 1
	username, role := "test", "admin"

	t.Run("jwt token", func(t *testing.T) {
		accessToken, refreshToken, err := jwt.GenerateTokens(userId, username, role)
		if err != nil {
			t.Fatalf("get refresh token is err: %v", err)
		}
		fmt.Printf("refresh token is: %v\naccess token is: %v\n", refreshToken, accessToken)

		parseToken, err := jwt.ParseToken(accessToken)
		if err != nil {
			t.Fatalf("get token info is err: %v", err)
		}
		fmt.Printf("token info is: %+v\n", parseToken)
		if parseToken.UserId != userId || parseToken.Role != role || parseToken.Username != username {
			t.Fatal("token info is err")
		}

		//time.Sleep(1 * time.Second)
		err = parseToken.ValidAccessToken()
		fmt.Println(err)

		newRefreshToken, accessToken, err := jwt.RefreshTokens(refreshToken)
		fmt.Printf("new refresh token is: %v\naccess token is: %v\nrefresh token is err: %v\n", newRefreshToken, accessToken, err)
	})

	t.Run("jwt xauth", func(t *testing.T) {
		accessToken, refreshToken, err := jwt.GenerateTokens(userId, username, role)
		if err != nil {
			t.Fatalf("get refresh token is err: %v", err)
		}
		fmt.Printf("refresh token is: %v\naccess token is: %v\n", refreshToken, accessToken)
		authHeader := fmt.Sprintf("Bearer %v", accessToken)
		if authHeader == "" {
			t.Fatalf("header is empty")
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			t.Fatalf("token format xerror")
		}
		// parts[1]是获取到的tokenString，我们使用之前定义好的解析JWT的函数来解析它
		mc, err := jwt.ParseToken(parts[1])
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
}
