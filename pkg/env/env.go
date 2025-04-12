package env

import "os"

const (
	ProdConstant = "prod"
	UatConstant  = "uat"
	DevConstant  = "dev"
)

// Env 获取当前环境
func Env() string {
	v := os.Getenv("ENV")
	if len(v) == 0 {
		return DevConstant
	}
	return v
}

// Prod 正式环境
func Prod() bool {
	return Env() == ProdConstant
}

// Uat uat环境
func Uat() bool {
	return Env() == UatConstant
}

// Dev dev环境
func Dev() bool {
	return Env() == DevConstant
}
