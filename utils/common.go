package utils

// ErrorToString 错误转为字符串
func ErrorToString(err interface{}) string {
	switch v := err.(type) {
	case error:
		return v.Error()
	default:
		return err.(string)
	}
}
