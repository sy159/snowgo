package system

import (
	"context"
	"gorm.io/gen"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	"time"
)

// LoginLogDao 登录日志
type LoginLogDao struct {
	repo *repo.Repository
}

func NewLoginLogDao(repo *repo.Repository) *LoginLogDao {
	return &LoginLogDao{repo: repo}
}

type LoginLogCondition struct {
	UserID    int32      `json:"user_id" form:"user_id"`
	Username  string     `json:"username" form:"username"`
	Status    *bool      `json:"status" form:"status"`
	StartTime *time.Time `json:"start_time" form:"start_time"`
	EndTime   *time.Time `json:"end_time" form:"end_time"`
	Offset    int32      `json:"offset" form:"offset"`
	Limit     int32      `json:"limit" form:"limit"`
}

// Create 创建登录日志（非事务，独立写入）
func (l *LoginLogDao) Create(ctx context.Context, log *model.SysLoginLog) (*model.SysLoginLog, error) {
	m := l.repo.Query().SysLoginLog
	err := m.WithContext(ctx).Create(log)
	if err != nil {
		return nil, err
	}
	return log, nil
}

// GetLoginLogList 登录日志列表
func (l *LoginLogDao) GetLoginLogList(ctx context.Context, condition *LoginLogCondition) ([]*model.SysLoginLog, int64, error) {
	m := l.repo.Query().SysLoginLog
	list, total, err := m.WithContext(ctx).
		Scopes(
			l.UserIDScope(condition.UserID),
			l.UsernameScope(condition.Username),
			l.StatusScope(condition.Status),
			l.StartTimeScope(condition.StartTime),
			l.EndTimeScope(condition.EndTime),
		).
		Order(m.ID.Desc()).
		FindByPage(int(condition.Offset), int(condition.Limit))
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (l *LoginLogDao) UserIDScope(userID int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if userID <= 0 {
			return tx
		}
		m := l.repo.Query().SysLoginLog
		return tx.Where(m.UserID.Eq(userID))
	}
}

func (l *LoginLogDao) UsernameScope(username string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(username) == 0 {
			return tx
		}
		m := l.repo.Query().SysLoginLog
		return tx.Where(m.Username.Like("%" + username + "%"))
	}
}

func (l *LoginLogDao) StatusScope(status *bool) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if status == nil {
			return tx
		}
		m := l.repo.Query().SysLoginLog
		if *status {
			return tx.Where(m.Status.Is(true))
		}
		return tx.Where(m.Status.Is(false))
	}
}

func (l *LoginLogDao) StartTimeScope(startTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if startTime == nil {
			return tx
		}
		m := l.repo.Query().SysLoginLog
		return tx.Where(m.CreatedAt.Gte(*startTime))
	}
}

func (l *LoginLogDao) EndTimeScope(endTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if endTime == nil {
			return tx
		}
		m := l.repo.Query().SysLoginLog
		return tx.Where(m.CreatedAt.Lte(*endTime))
	}
}
