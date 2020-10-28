package task

func DefaultTaskProperties() map[string]string {
	return map[string]string{
		"allowFailures":               "false",
		"experimentalFeaturesEnabled": "false",
		"timeoutInSeconds":            "3600",
		"triggerType":                 "AUTOMATIC",
	}
}
