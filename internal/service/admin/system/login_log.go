package system

import (
	"context"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/admin/system"
	"snowgo/pkg/xlogger"
	"time"
)

// LoginLogRepo 定义登录日志相关 db 操作接口
type LoginLogRepo interface {
	Create(ctx context.Context, log *model.SysLoginLog) (*model.SysLoginLog, error)
	GetLoginLogList(ctx context.Context, condition *daoSystem.LoginLogCondition) ([]*model.SysLoginLog, int64, error)
}

type LoginLogService struct {
	db          *repo.Repository
	loginLogDao LoginLogRepo
}

func NewLoginLogService(db *repo.Repository, loginLogDao LoginLogRepo) *LoginLogService {
	return &LoginLogService{
		db:          db,
		loginLogDao: loginLogDao,
	}
}

// LoginLogInput 登录日志输入
type LoginLogInput struct {
	UserID    int32
	Username  string
	IP        string
	Status    bool   // true=成功，false=失败
	Message   string // 失败原因
	UserAgent string // 浏览器/设备信息
}

// LoginLogCondition 登录日志查询条件（API 层使用，字符串时间）
type LoginLogCondition struct {
	UserID    int32  `json:"user_id" form:"user_id"`
	Username  string `json:"username" form:"username"`
	Status    *bool  `json:"status" form:"status"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Offset    int32  `json:"offset" form:"offset"`
	Limit     int32  `json:"limit" form:"limit"`
}

// LoginLogInfo 登录日志输出信息
type LoginLogInfo struct {
	ID        int64      `json:"id"`
	UserID    int32      `json:"user_id"`
	Username  string     `json:"username"`
	IP        string     `json:"ip"`
	Status    bool       `json:"status"`
	Message   *string    `json:"message"`
	UserAgent *string    `json:"user_agent"`
	CreatedAt *time.Time `json:"created_at"`
}

// LoginLogList 登录日志列表输出
type LoginLogList struct {
	List  []*LoginLogInfo `json:"list"`
	Total int64           `json:"total"`
}

// CreateLoginLog 创建登录日志
func (l *LoginLogService) CreateLoginLog(ctx context.Context, input *LoginLogInput) {
	log := &model.SysLoginLog{
		UserID:   input.UserID,
		Username: input.Username,
		IP:       input.IP,
		Status:   input.Status,
	}
	if input.Message != "" {
		log.Message = &input.Message
	}
	if input.UserAgent != "" {
		log.UserAgent = &input.UserAgent
	}

	_, err := l.loginLogDao.Create(ctx, log)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "创建登录日志失败: %+v err: %v", input, err)
	}
}

// GetLoginLogList 获取登录日志列表
func (l *LoginLogService) GetLoginLogList(ctx context.Context, condition *LoginLogCondition) (*LoginLogList, error) {
	var startTimePtr *time.Time
	var endTimePtr *time.Time
	if condition.StartTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.StartTime, time.Local)
		if err != nil {
			return nil, ErrTimeFormat
		}
		startTimePtr = &t
	}
	if condition.EndTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.EndTime, time.Local)
		if err != nil {
			return nil, ErrTimeFormat
		}
		endTimePtr = &t
	}

	list, total, err := l.loginLogDao.GetLoginLogList(ctx, &daoSystem.LoginLogCondition{
		UserID:    condition.UserID,
		Username:  condition.Username,
		Status:    condition.Status,
		StartTime: startTimePtr,
		EndTime:   endTimePtr,
		Offset:    condition.Offset,
		Limit:     condition.Limit,
	})
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取登录日志信息列表异常: %v", err)
		return nil, fmt.Errorf("登录日志信息列表查询失败: %w", err)
	}

	logList := make([]*LoginLogInfo, 0, len(list))
	for _, item := range list {
		logList = append(logList, &LoginLogInfo{
			ID:        item.ID,
			UserID:    item.UserID,
			Username:  item.Username,
			IP:        item.IP,
			Status:    item.Status,
			Message:   item.Message,
			UserAgent: item.UserAgent,
			CreatedAt: item.CreatedAt,
		})
	}

	return &LoginLogList{List: logList, Total: total}, nil
}
