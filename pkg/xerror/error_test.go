package xerror_test

import (
	"encoding/json"
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
