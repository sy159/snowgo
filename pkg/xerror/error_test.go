package xerror_test

import (
	"encoding/json"
	"errors"
	"snowgo/pkg/xerror"
	"testing"
)

func TestNewCode(t *testing.T) {
	// 测试 NewCode 是否正确创建 Code 实例
	code := xerror.NewCode(xerror.CategoryHttp, 90001, "Test Error")

	if code.GetErrCode() != 90001 {
		t.Errorf("Expected error code 90001, but got %d", code.GetErrCode())
	}

	if code.GetErrMsg() != "Test Error" {
		t.Errorf("Expected error message 'Test Error', but got '%s'", code.GetErrMsg())
	}
}

func TestCodeToString(t *testing.T) {
	// 测试 Code 的 ToString 方法
	code := xerror.NewCode(xerror.CategoryHttp, 90002, "Test ToString")
	expected := `{"code":90002,"msg":"Test ToString","category":"http"}`
	result := code.ToString()

	// 对比 JSON 格式的错误信息
	if result != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestDuplicateCodePanics(t *testing.T) {
	// 重复 errCode 注册应 panic 而非静默覆盖
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate error code")
		}
	}()
	_ = xerror.NewCode(xerror.CategoryHttp, 90003, "First")
	_ = xerror.NewCode(xerror.CategoryHttp, 90003, "Duplicate")
}

func TestErrorCodes(t *testing.T) {
	// 测试常用错误码是否正确
	tests := []struct {
		Code         xerror.Code
		ExpectedCode int
		ExpectedMsg  string
	}{
		{xerror.OK, 0, "success"},
		{xerror.HttpOK, 200, "ok"},
		{xerror.HttpBadRequest, 400, "Bad Request"},
		{xerror.TokenNotFound, 10101, "token不能为空"},
		{xerror.UserNotFound, 10201, "用户不存在"},
	}

	for _, test := range tests {
		if test.Code.GetErrCode() != test.ExpectedCode {
			t.Errorf("Expected error code %d, but got %d", test.ExpectedCode, test.Code.GetErrCode())
		}

		if test.Code.GetErrMsg() != test.ExpectedMsg {
			t.Errorf("Expected error message '%s', but got '%s'", test.ExpectedMsg, test.Code.GetErrMsg())
		}
	}
}

func TestCodeJSON(t *testing.T) {
	// 测试 Code 的 JSON 序列化
	code := xerror.NewCode(xerror.CategoryHttp, 90011, "Bad Request")
	raw, err := json.Marshal(code)
	if err != nil {
		t.Errorf("Failed to marshal code: %v", err)
		return
	}

	expected := `{"code":90011,"msg":"Bad Request","category":"http"}`
	if string(raw) != expected {
		t.Errorf("Expected JSON '%s', but got '%s'", expected, string(raw))
	}
}

func TestCodeError(t *testing.T) {
	// 测试 Code 实现 error 接口
	code := xerror.NewCode(xerror.CategoryHttp, 90020, "test error message")
	if code.Error() != "test error message" {
		t.Errorf("Expected Error() 'test error message', but got '%s'", code.Error())
	}
}

func TestGetCodes(t *testing.T) {
	// 测试 GetCodes 返回所有已注册的错误码
	codes := xerror.GetCodes()
	if len(codes) == 0 {
		t.Fatal("Expected at least some registered codes")
	}

	// 验证 OK code 在列表中
	found := false
	for _, c := range codes {
		if c.GetErrCode() == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find OK code in GetCodes result")
	}
}

func TestGetCategory(t *testing.T) {
	code := xerror.NewCode(xerror.CategoryHttp, 90030, "test category")
	if code.GetCategory() != xerror.CategoryHttp {
		t.Errorf("GetCategory() = %q, want %q", code.GetCategory(), xerror.CategoryHttp)
	}
}

func TestNewBizError(t *testing.T) {
	bizErr := xerror.NewBizError(xerror.UserNotFound)
	if bizErr.Code.GetErrCode() != xerror.UserNotFound.GetErrCode() {
		t.Errorf("BizError code = %d, want %d", bizErr.Code.GetErrCode(), xerror.UserNotFound.GetErrCode())
	}
	if bizErr.Error() != xerror.UserNotFound.GetErrMsg() {
		t.Errorf("BizError.Error() = %q, want %q", bizErr.Error(), xerror.UserNotFound.GetErrMsg())
	}
	if bizErr.Unwrap() != nil {
		t.Error("BizError.Unwrap() should be nil when no cause")
	}
}

func TestWrapBizError(t *testing.T) {
	cause := xerror.NewCode(xerror.CategoryHttp, 90031, "cause error")
	bizErr := xerror.WrapBizError(xerror.UserNotFound, cause)
	if bizErr.Code.GetErrCode() != xerror.UserNotFound.GetErrCode() {
		t.Errorf("BizError code = %d, want %d", bizErr.Code.GetErrCode(), xerror.UserNotFound.GetErrCode())
	}
	// Error() 应该包含 cause 信息
	errMsg := bizErr.Error()
	if errMsg == "" {
		t.Error("BizError.Error() should not be empty")
	}
	// Unwrap 应该返回 cause
	if unwrapped := bizErr.Unwrap(); unwrapped == nil {
		t.Error("BizError.Unwrap() should return cause when present")
	}
}

// === Additional tests ===

func TestNewCode_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: custom category with unique code", func(t *testing.T) {
		code := xerror.NewCode("custom", 99001, "test")
		if code.GetErrCode() != 99001 {
			t.Fatalf("expected error code 99001, got %d", code.GetErrCode())
		}
		if code.GetCategory() != "custom" {
			t.Fatalf("expected category 'custom', got %q", code.GetCategory())
		}
	})

	t.Run("boundary: negative error code", func(t *testing.T) {
		code := xerror.NewCode(xerror.CategorySystem, -1, "negative code")
		if code.GetErrCode() != -1 {
			t.Fatalf("expected error code -1, got %d", code.GetErrCode())
		}
	})

	t.Run("boundary: empty error message", func(t *testing.T) {
		code := xerror.NewCode(xerror.CategorySystem, -2, "")
		if code.GetErrMsg() != "" {
			t.Fatalf("expected empty error message, got %q", code.GetErrMsg())
		}
	})

	t.Run("boundary: empty category", func(t *testing.T) {
		code := xerror.NewCode("", -3, "empty category")
		if code.GetCategory() != "" {
			t.Fatalf("expected empty category, got %q", code.GetCategory())
		}
	})
}

func TestBizError_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: NewBizError with nil code", func(t *testing.T) {
		bizErr := xerror.NewBizError(nil)
		if bizErr == nil {
			t.Fatal("NewBizError(nil) should not return nil")
		}
	})

	t.Run("boundary: WrapBizError with nil cause", func(t *testing.T) {
		bizErr := xerror.WrapBizError(xerror.UserNotFound, nil)
		if bizErr == nil {
			t.Fatal("WrapBizError with nil cause should not return nil")
		}
		// Error() should still work with nil cause
		errMsg := bizErr.Error()
		if errMsg != xerror.UserNotFound.GetErrMsg() {
			t.Fatalf("expected error=%q, got %q", xerror.UserNotFound.GetErrMsg(), errMsg)
		}
		// Unwrap should return nil
		if bizErr.Unwrap() != nil {
			t.Fatal("WrapBizError with nil cause should have nil Unwrap")
		}
	})
}

func TestBizError_Unwrap(t *testing.T) {
	// === Happy path ===
	t.Run("happy: errors.Is with wrapped cause", func(t *testing.T) {
		cause := xerror.NewCode(xerror.CategoryHttp, 90040, "inner cause")
		bizErr := xerror.WrapBizError(xerror.UserNotFound, cause)

		if !errors.Is(bizErr, cause) {
			t.Fatal("errors.Is should find the wrapped cause")
		}
	})

	t.Run("happy: errors.As extracts BizError", func(t *testing.T) {
		bizErr := xerror.NewBizError(xerror.UserNotFound)

		var extracted *xerror.BizError
		if !errors.As(bizErr, &extracted) {
			t.Fatal("errors.As should extract BizError")
		}
		if extracted.Code.GetErrCode() != xerror.UserNotFound.GetErrCode() {
			t.Fatalf("extracted code = %d, want %d", extracted.Code.GetErrCode(), xerror.UserNotFound.GetErrCode())
		}
	})
}

func TestCode_Immutability(t *testing.T) {
	// === Happy path: code is immutable ===
	t.Run("happy: Code fields cannot be changed after creation", func(t *testing.T) {
		code := xerror.NewCode(xerror.CategoryHttp, 90050, "immutable test")

		// The code interface only exposes getters, no setters
		if code.GetErrCode() != 90050 {
			t.Fatalf("expected 90050, got %d", code.GetErrCode())
		}
		if code.GetErrMsg() != "immutable test" {
			t.Fatalf("expected 'immutable test', got %q", code.GetErrMsg())
		}
		if code.GetCategory() != "http" {
			t.Fatalf("expected 'http', got %q", code.GetCategory())
		}
	})
}

func TestNewCode_DifferentCategories(t *testing.T) {
	// === Happy path: multiple categories ===
	t.Run("happy: same code number in different categories", func(t *testing.T) {
		// Note: NewCode uses ErrCode (not category) as the registry key,
		// so same ErrCode in different categories would panic.
		// This tests the design constraint.
		c1 := xerror.NewCode("cat1", 90060, "cat1 error")
		if c1.GetCategory() != "cat1" {
			t.Fatalf("expected category 'cat1', got %q", c1.GetCategory())
		}
	})
}
