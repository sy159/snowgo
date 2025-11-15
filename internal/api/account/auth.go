package account

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/constants"
	"snowgo/internal/di"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

// Login 登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	// 验证用户名密码
	user, err := di.GetContainer(c).UserService.Authenticate(c, req.Username, req.Password)
	if err != nil {
		xresponse.FailByError(c, e.AuthError)
		return
	}

	// 生成 JWT
	container := di.GetContainer(c)
	jwtMgr := container.JwtManager
	token, err := jwtMgr.GenerateTokens(int64(user.ID), user.Username)
	if err != nil {
		xlogger.Errorf("jwt generate tokens err: %v", err)
		xresponse.FailByError(c, e.TokenError)
		return
	}

	// 保存 refresh token 的 jti，设置过期时间（防止重放攻击、每个refresh token只能使用一次）
	if claims, err := jwtMgr.ParseToken(token.RefreshToken); err == nil {
		jtiKey := constants.CacheRefreshJtiPrefix + claims.ID
		_ = container.Cache.Set(c, jtiKey, "1", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time))
	}

	xresponse.Success(c, gin.H{
		"access_token":             token.AccessToken,
		"refresh_token":            token.RefreshToken,
		"access_expire_timestamp":  token.AccessExpire.Unix(),
		"refresh_expire_timestamp": token.RefreshExpire.Unix(),
	})
}

// RefreshToken 刷新token
func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	container := di.GetContainer(c)
	jwtMgr := container.JwtManager

	// 检查 jti 是否使用过（防止重放攻击、每个refresh token只能使用一次）
	claims, err := jwtMgr.ParseToken(req.RefreshToken)
	if err != nil {
		xresponse.Fail(c, e.TokenInvalid.GetErrCode(), err.Error())
		return
	}

	jtiKey := constants.CacheRefreshJtiPrefix + claims.ID
	if del, _ := container.Cache.Delete(c, jtiKey); del == 0 {
		xlogger.Errorf("refresh token reuse attempt: userID=%d, jti=%s", claims.UserId, claims.ID)
		xresponse.FailByError(c, e.TokenUseDError)
		return
	}

	// 生成新的token
	token, err := jwtMgr.RefreshTokens(req.RefreshToken)
	if err != nil {
		xlogger.Errorf("refresh access token err: %s", err.Error())
		xresponse.FailByError(c, e.TokenError)
		return
	}

	// 保存 refresh token 的 jti，设置过期时间（防止重放攻击、每个refresh token只能使用一次）
	if newClaims, err := jwtMgr.ParseToken(token.RefreshToken); err == nil {
		jtiKey = constants.CacheRefreshJtiPrefix + newClaims.ID
		_ = container.Cache.Set(c, jtiKey, "1", newClaims.ExpiresAt.Time.Sub(newClaims.IssuedAt.Time))
	}

	xresponse.Success(c, gin.H{
		"access_token":             token.AccessToken,
		"refresh_token":            token.RefreshToken,
		"access_expire_timestamp":  token.AccessExpire.Unix(),
		"refresh_expire_timestamp": token.RefreshExpire.Unix(),
	})
}
