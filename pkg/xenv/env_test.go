package xenv_test

import (
	"os"
	"snowgo/pkg/xenv"
	"testing"
)

func TestEnv(t *testing.T) {
	// 测试不同的环境变量设置，确保能够正确获取环境

	// 测试默认环境
	os.Unsetenv("ENV")
	if xenv.Env() != xenv.DevConstant {
		t.Errorf("Expected %s, got %s", xenv.DevConstant, xenv.Env())
	}

	// 设置为prod环境
	os.Setenv("ENV", xenv.ProdConstant)
	if xenv.Env() != xenv.ProdConstant {
		t.Errorf("Expected %s, got %s", xenv.ProdConstant, xenv.Env())
	}

	// 设置为uat环境
	os.Setenv("ENV", xenv.UatConstant)
	if xenv.Env() != xenv.UatConstant {
		t.Errorf("Expected %s, got %s", xenv.UatConstant, xenv.Env())
	}
}

func TestProd(t *testing.T) {
	// 测试 Prod() 函数，检查是否正确判断是否是生产环境

	// 设置为prod环境
	os.Setenv("ENV", xenv.ProdConstant)
	if !xenv.Prod() {
		t.Errorf("Expected prod environment, but got %s", xenv.Env())
	}

	// 设置为非prod环境
	os.Setenv("ENV", xenv.UatConstant)
	if xenv.Prod() {
		t.Errorf("Expected non-prod environment, but got %s", xenv.Env())
	}
}

func TestUat(t *testing.T) {
	// 测试 Uat() 函数，检查是否正确判断是否是UAT环境

	// 设置为uat环境
	os.Setenv("ENV", xenv.UatConstant)
	if !xenv.Uat() {
		t.Errorf("Expected uat environment, but got %s", xenv.Env())
	}

	// 设置为非uat环境
	os.Setenv("ENV", xenv.ProdConstant)
	if xenv.Uat() {
		t.Errorf("Expected non-uat environment, but got %s", xenv.Env())
	}
}

func TestDev(t *testing.T) {
	// 测试 Dev() 函数，检查是否正确判断是否是开发环境

	// 设置为dev环境
	os.Setenv("ENV", xenv.DevConstant)
	if !xenv.Dev() {
		t.Errorf("Expected dev environment, but got %s", xenv.Env())
	}

	// 设置为非dev环境
	os.Setenv("ENV", xenv.ProdConstant)
	if xenv.Dev() {
		t.Errorf("Expected non-dev environment, but got %s", xenv.Env())
	}
}
