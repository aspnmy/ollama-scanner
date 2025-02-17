package config

import (
	"fmt"
	"os"
)

// RequiredVars 定义必需的环境变量
var RequiredVars = map[string]string{
	"PORT":               "11434",
	"TIMEZONE":           "Asia/Shanghai",
	"LOG_DIR":            "logs",
	"TELEGRAM_URI":       "",
	"TELEGRAM_BOT_TOKEN": "",
	"TELEGRAM_CHAT_ID":   "",
	// 添加其他必需的环境变量
}

// ValidateEnv 验证环境变量
func ValidateEnv() error {
	missing := []string{}

	for key, defaultValue := range RequiredVars {
		if value := os.Getenv(key); value == "" {
			if defaultValue == "" {
				missing = append(missing, key)
			} else {
				os.Setenv(key, defaultValue)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("缺少必需的环境变量: %v", missing)
	}

	return nil
}
