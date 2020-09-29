package task

func DefaultTaskProperties() map[string]string {
	return map[string]string{
		"allowFailures":               "false",
		"executionLock":               "null",
		"experimentalFeaturesEnabled": "false",
		"timeoutInSeconds":            "3600",
		"triggerType":                 "AUTOMATIC",
	}
}
