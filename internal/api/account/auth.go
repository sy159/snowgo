package account

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"snowgo/internal/constant"
	"snowgo/internal/di"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xauth/jwt"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlimiter"
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
	ctx := c.Request.Context()

	container := di.GetContainer(c)
	cache := container.Cache

	// 登录失败限流，3分钟内，最多失败5次
	loginFailKey := fmt.Sprintf("%s%s", constant.CacheLoginFailPrefix, req.Username)
	limiter := xlimiter.NewFixedWindowLimiter(cache, loginFailKey, constant.CacheLoginFailWindowSecond, 5)
	// 尝试增加失败计数前，先检查限流器
	allowed, _, ttl, err := limiter.Add(ctx)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "login limiter error: %v", err)
		xresponse.FailByError(c, e.HttpInternalServerError)
		return
	}

	// 如果不允许，直接返回锁定信息
	if !allowed {
		xresponse.Fail(c, e.LoginLocked.GetErrCode(), fmt.Sprintf("登录失败次数过多，请等待%d秒后再试", int(ttl.Seconds())))
		return
	}

	// 验证用户名密码
	user, err := container.UserService.Authenticate(ctx, req.Username, req.Password)
	if err != nil {
		xresponse.FailByError(c, e.AuthError)
		return
	}

	// 成功登录 → 重置失败计数
	_ = limiter.Reset(ctx)

	jwtMgr := container.JwtManager
	token, err := jwtMgr.GenerateTokens(user.ID, user.Username)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "jwt generate tokens err: %v", err)
		xresponse.FailByError(c, e.TokenError)
		return
	}

	// 保存 refresh token 的 jti，设置过期时间（防止重放攻击、每个refresh token只能使用一次）
	if claims, err := jwtMgr.ParseToken(token.RefreshToken); err == nil {
		jtiKey := constant.CacheRefreshJtiPrefix + claims.ID
		_ = container.Cache.Set(ctx, jtiKey, "1", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time))
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
	ctx := c.Request.Context()

	container := di.GetContainer(c)
	jwtMgr := container.JwtManager

	// 检查 jti 是否使用过（防止重放攻击、每个refresh token只能使用一次）
	claims, err := jwtMgr.ParseToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			xresponse.Fail(c, e.HttpUnauthorized.GetErrCode(), e.TokenExpired.GetErrMsg())
			return
		}
		xlogger.ErrorfCtx(c.Request.Context(), "parse token(%s) is err: %v", req.RefreshToken, err)
		xresponse.Fail(c, e.HttpUnauthorized.GetErrCode(), e.TokenInvalid.GetErrMsg())
		return
	}

	// 生成新的token
	token, err := jwtMgr.RefreshTokens(req.RefreshToken)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "refresh access token err: %s", err.Error())
		xresponse.FailByError(c, e.HttpInternalServerError)
		return
	}

	jtiKey := constant.CacheRefreshJtiPrefix + claims.ID
	if del, _ := container.Cache.Delete(ctx, jtiKey); del == 0 {
		xlogger.ErrorfCtx(ctx, "refresh token reuse attempt: userID=%d, jti=%s", claims.UserId, claims.ID)
		xresponse.FailByError(c, e.TokenUsedError)
		return
	}

	// 保存 refresh token 的 jti，设置过期时间（防止重放攻击、每个refresh token只能使用一次）
	if newClaims, err := jwtMgr.ParseToken(token.RefreshToken); err == nil {
		jtiKey = constant.CacheRefreshJtiPrefix + newClaims.ID
		_ = container.Cache.Set(ctx, jtiKey, "1", newClaims.ExpiresAt.Time.Sub(newClaims.IssuedAt.Time))
	}

	xresponse.Success(c, gin.H{
		"access_token":             token.AccessToken,
		"refresh_token":            token.RefreshToken,
		"access_expire_timestamp":  token.AccessExpire.Unix(),
		"refresh_expire_timestamp": token.RefreshExpire.Unix(),
	})
}

func Logout(c *gin.Context) {
	// 获取登录ctx
	ctx := c.Request.Context()
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		xresponse.FailByError(c, e.HttpForbidden)
		return
	}

	cache := di.GetContainer(c).Cache

	// 根据 sessionId 删除 refresh token
	jtiKey := constant.CacheRefreshJtiPrefix + userContext.SessionId
	_, _ = cache.Delete(ctx, jtiKey)

	xresponse.Success(c, gin.H{
		"user_id": userContext.UserId,
	})
}
