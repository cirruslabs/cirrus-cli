package task

func DefaultTaskProperties() map[string]string {
	return map[string]string{
		"allow_failures":     "false",
		"experimental":       "false",
		"skip_notifications": "false",
		"timeout_in":         "3600",
		"trigger_type":       "AUTOMATIC",
	}
}
