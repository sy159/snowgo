package account

import (
	"errors"
	"testing"

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
			if bizErr.Code != tt.wantCode {
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
	if err := service.ResetPwdById(testUserCtx(), 2, "abcdef"); !errors.Is(err, e.NewBizError(e.PwdComplexityError)) {
		var bizErr *e.BizError
		if !errors.As(err, &bizErr) || bizErr.Code != e.PwdComplexityError {
			t.Fatalf("ResetPwdById weak password expected PwdComplexityError, got %v", err)
		}
	}
}
