package config

import "os"

// FetchEnv gets var from env or use first default value
func FetchEnv(key string, defaultValue ...string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return value
}
