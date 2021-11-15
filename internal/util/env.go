package util

func GetFirstValue(env map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := env[key]
		if ok {
			return value, true
		}
	}
	return "", false
}
