package account

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/pkg/xcryption"
	e "snowgo/pkg/xerror"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantCode e.Code
	}{
		{name: "empty", password: "", wantCode: e.PwdComplexityError},
		{name: "too short", password: "a1@", wantCode: e.PwdLengthError},
		{name: "too long", password: "a1@" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantCode: e.PwdLengthError},
		{name: "invalid char", password: "abc123中文", wantCode: e.PwdInvalidCharError},
		{name: "single class", password: "abcdef", wantCode: e.PwdComplexityError},
		{name: "letter and digit", password: "abc123", wantCode: nil},
		{name: "letter and symbol", password: "abcdef!", wantCode: nil},
		{name: "digit and symbol", password: "123456!", wantCode: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantCode == nil {
				if err != nil {
					t.Fatalf("expected password to be valid, got %v", err)
				}
				return
			}
			var bizErr *e.BizError
			if !errors.As(err, &bizErr) {
				t.Fatalf("expected BizError, got %v", err)
			}
			if bizErr.Code.GetErrCode() != tt.wantCode.GetErrCode() {
				t.Fatalf("expected code %v, got %v", tt.wantCode, bizErr.Code)
			}
		})
	}
}

func TestUserServiceEarlyValidation(t *testing.T) {
	service := &UserService{}

	if err := service.DeleteById(testUserCtx(), 0); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("DeleteById expected ErrUserNotFound, got %v", err)
	}
	if err := service.DeleteById(testUserCtx(), 1); !errors.Is(err, ErrDeleteSelf) {
		t.Fatalf("DeleteById self expected ErrDeleteSelf, got %v", err)
	}
	if err := service.ResetPwdById(testUserCtx(), 0, "abc123"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ResetPwdById expected ErrUserNotFound, got %v", err)
	}
	err := service.ResetPwdById(testUserCtx(), 2, "abcdef")
	var bizErr *e.BizError
	if !errors.As(err, &bizErr) || bizErr.Code.GetErrCode() != e.PwdComplexityError.GetErrCode() {
		t.Fatalf("ResetPwdById weak password expected PwdComplexityError, got %v", err)
	}
}

func TestUserServiceAuthenticate(t *testing.T) {
	activeStatus := constant.UserStatusActive
	disabledStatus := constant.UserStatusDisabled
	passwordHash, err := xcryption.HashPassword("abc123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tests := []struct {
		name       string
		repo       *fakeUserRepo
		password   string
		wantErr    error
		wantStatus int8
	}{
		{
			name:     "empty username",
			repo:     &fakeUserRepo{},
			password: "abc123",
			wantErr:  ErrUserNotFound,
		},
		{
			name:       "user not found returns auth error",
			repo:       &fakeUserRepo{userByUsernameErr: gorm.ErrRecordNotFound},
			password:   "abc123",
			wantErr:    ErrAuth,
			wantStatus: 0,
		},
		{
			name: "wrong password",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Status:   &activeStatus,
			}},
			password: "wrong123",
			wantErr:  ErrAuth,
		},
		{
			name: "disabled user",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Status:   &disabledStatus,
			}},
			password: "abc123",
			wantErr:  ErrAuth,
		},
		{
			name: "success",
			repo: &fakeUserRepo{userByUsername: &model.SysUser{
				ID:       2,
				Username: "operator",
				Password: passwordHash,
				Tel:      "18712345678",
				Status:   &activeStatus,
			}},
			password:   "abc123",
			wantStatus: activeStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username := "operator"
			if tt.wantErr == ErrUserNotFound {
				username = ""
			}
			service := &UserService{userDao: tt.repo}
			got, err := service.Authenticate(testUserCtx(), username, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected err %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if got.ID != 2 || got.Username != "operator" || got.Tel != "18712345678" || got.Status != tt.wantStatus {
				t.Fatalf("unexpected user info: %+v", got)
			}
		})
	}
}

func TestUserServiceGetRoleIdsByUserId(t *testing.T) {
	cacheKey := constant.CacheUserRolePrefix + "2"

	t.Run("invalid user id", func(t *testing.T) {
		service := &UserService{}
		_, err := service.GetRoleIdsByUserId(testUserCtx(), 0)
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "[1,2]"
		repo := &fakeUserRepo{roleIds: []int32{3}}
		service := &UserService{userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{1, 2}) {
			t.Fatalf("expected cached role ids [1 2], got %v", got)
		}
		if repo.getRoleIDsCalls != 0 {
			t.Fatalf("expected dao not called, got %d calls", repo.getRoleIDsCalls)
		}
	})

	t.Run("cache miss reads dao and writes cache", func(t *testing.T) {
		cache := newFakeCache()
		repo := &fakeUserRepo{roleIds: []int32{3, 4}}
		service := &UserService{userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{3, 4}) {
			t.Fatalf("expected dao role ids [3 4], got %v", got)
		}
		if cache.sets[cacheKey] != "[3,4]" {
			t.Fatalf("expected cache set [3,4], got %q", cache.sets[cacheKey])
		}
	})

	t.Run("broken cache falls back to dao", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "not-json"
		repo := &fakeUserRepo{roleIds: []int32{5}}
		service := &UserService{userDao: repo, cache: cache}

		got, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if !equalInt32s(got, []int32{5}) {
			t.Fatalf("expected dao role ids [5], got %v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		service := &UserService{
			userDao: &fakeUserRepo{roleIdsErr: errTestDAO},
			cache:   newFakeCache(),
		}

		_, err := service.GetRoleIdsByUserId(testUserCtx(), 2)
		if !errors.Is(err, errTestDAO) {
			t.Fatalf("expected dao error, got %v", err)
		}
	})
}

func equalInt32s(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
