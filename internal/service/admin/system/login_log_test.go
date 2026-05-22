package system

import (
	"errors"
	"testing"
	"time"

	"snowgo/internal/dal/model"
)

func TestLoginLogServiceCreateLoginLog(t *testing.T) {
	t.Run("maps non-empty optional fields", func(t *testing.T) {
		repo := &fakeLoginLogRepo{}
		service := &LoginLogService{loginLogDao: repo}

		service.CreateLoginLog(testUserCtx(), &LoginLogInput{
			UserID:    1,
			Username:  "admin",
			IP:        "127.0.0.1",
			Status:    true,
			Message:   "ok",
			UserAgent: "Chrome",
		})

		got := repo.createdLog
		if got == nil {
			t.Fatalf("expected log to be created")
		}
		if got.UserID != 1 || got.Username != "admin" || got.IP != "127.0.0.1" || !got.Status {
			t.Fatalf("unexpected login log: %+v", got)
		}
		if got.Message == nil || *got.Message != "ok" {
			t.Fatalf("expected message to be mapped, got %+v", got.Message)
		}
		if got.UserAgent == nil || *got.UserAgent != "Chrome" {
			t.Fatalf("expected user agent to be mapped, got %+v", got.UserAgent)
		}
	})

	t.Run("keeps empty optional fields nil", func(t *testing.T) {
		repo := &fakeLoginLogRepo{}
		service := &LoginLogService{loginLogDao: repo}

		service.CreateLoginLog(testUserCtx(), &LoginLogInput{Username: "admin"})

		got := repo.createdLog
		if got == nil {
			t.Fatalf("expected log to be created")
		}
		if got.Message != nil || got.UserAgent != nil {
			t.Fatalf("expected optional fields nil, got message=%+v user_agent=%+v", got.Message, got.UserAgent)
		}
	})
}

func TestLoginLogServiceGetLoginLogList(t *testing.T) {
	t.Run("invalid start time", func(t *testing.T) {
		service := &LoginLogService{}
		_, err := service.GetLoginLogList(testUserCtx(), &LoginLogCondition{StartTime: "bad-time"})
		if !errors.Is(err, ErrTimeFormat) {
			t.Fatalf("expected ErrTimeFormat, got %v", err)
		}
	})

	t.Run("invalid end time", func(t *testing.T) {
		service := &LoginLogService{}
		_, err := service.GetLoginLogList(testUserCtx(), &LoginLogCondition{EndTime: "bad-time"})
		if !errors.Is(err, ErrTimeFormat) {
			t.Fatalf("expected ErrTimeFormat, got %v", err)
		}
	})

	t.Run("maps condition and result", func(t *testing.T) {
		status := true
		message := "登录成功"
		userAgent := "Chrome"
		createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.Local)
		repo := &fakeLoginLogRepo{
			list: []*model.SysLoginLog{{
				ID:        1,
				UserID:    2,
				Username:  "operator",
				IP:        "127.0.0.1",
				Status:    true,
				Message:   &message,
				UserAgent: &userAgent,
				CreatedAt: &createdAt,
			}},
			total: 1,
		}
		service := &LoginLogService{loginLogDao: repo}

		got, err := service.GetLoginLogList(testUserCtx(), &LoginLogCondition{
			UserID:    2,
			Username:  "operator",
			Status:    &status,
			StartTime: "2026-05-22 00:00:00",
			EndTime:   "2026-05-22 23:59:59",
			Offset:    20,
			Limit:     10,
		})
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if got.Total != 1 || len(got.List) != 1 {
			t.Fatalf("unexpected login log list: %+v", got)
		}
		item := got.List[0]
		if item.ID != 1 || item.UserID != 2 || item.Username != "operator" || item.IP != "127.0.0.1" || !item.Status {
			t.Fatalf("unexpected login log item: %+v", item)
		}
		if item.Message == nil || *item.Message != message || item.UserAgent == nil || *item.UserAgent != userAgent {
			t.Fatalf("unexpected optional fields: %+v", item)
		}
		if repo.condition == nil {
			t.Fatalf("expected dao condition to be captured")
		}
		if repo.condition.UserID != 2 || repo.condition.Username != "operator" || repo.condition.Status != &status ||
			repo.condition.Offset != 20 || repo.condition.Limit != 10 {
			t.Fatalf("unexpected dao condition: %+v", repo.condition)
		}
		if repo.condition.StartTime == nil || repo.condition.EndTime == nil {
			t.Fatalf("expected time range to be parsed, got %+v", repo.condition)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		service := &LoginLogService{loginLogDao: &fakeLoginLogRepo{listErr: errTestDAO}}

		_, err := service.GetLoginLogList(testUserCtx(), &LoginLogCondition{})
		if !errors.Is(err, errTestDAO) {
			t.Fatalf("expected dao error, got %v", err)
		}
	})
}
