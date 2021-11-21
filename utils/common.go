package utils


func ErrorToString(err interface{}) string {
	switch v := err.(type) {
	case error:
		return v.Error()
	default:
		return err.(string)
	}
}
