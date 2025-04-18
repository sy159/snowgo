package xerror_test

import (
	"encoding/json"
	"snowgo/pkg/xerror"
	"testing"
)

func TestNewCode(t *testing.T) {
	// 测试 NewCode 是否正确创建 Code 实例
	code := xerror.NewCode(100, "Test Error")

	if code.GetErrCode() != 100 {
		t.Errorf("Expected error code 100, but got %d", code.GetErrCode())
	}

	if code.GetErrMsg() != "Test Error" {
		t.Errorf("Expected error message 'Test Error', but got '%s'", code.GetErrMsg())
	}
}

func TestCodeToString(t *testing.T) {
	// 测试 Code 的 ToString 方法
	code := xerror.NewCode(101, "Test ToString")
	expected := `{"code":101,"msg":"Test ToString"}`
	result := code.ToString()

	// 对比 JSON 格式的错误信息
	if result != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestSetErrCode(t *testing.T) {
	// 测试 SetErrCode 方法
	code := xerror.NewCode(200, "Initial Message")
	code.SetErrCode(300)

	if code.GetErrCode() != 300 {
		t.Errorf("Expected error code 300, but got %d", code.GetErrCode())
	}
}

func TestSetErrMsg(t *testing.T) {
	// 测试 SetErrMsg 方法
	code := xerror.NewCode(200, "Initial Message")
	code.SetErrMsg("Updated Message")

	if code.GetErrMsg() != "Updated Message" {
		t.Errorf("Expected error message 'Updated Message', but got '%s'", code.GetErrMsg())
	}
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
		{xerror.TokenNotFound, 10201, "token不能为空"},
		{xerror.UserNotFound, 10301, "用户不存在"},
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
	code := xerror.NewCode(400, "Bad Request")
	raw, err := json.Marshal(code)
	if err != nil {
		t.Errorf("Failed to marshal code: %v", err)
		return
	}

	expected := `{"code":400,"msg":"Bad Request"}`
	if string(raw) != expected {
		t.Errorf("Expected JSON '%s', but got '%s'", expected, string(raw))
	}
}
