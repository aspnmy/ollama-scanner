package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	TimeZone string `json:"timezone"`
}

var (
	DefaultConfig = Config{
		TimeZone: "Asia/Shanghai",
	}
	currentConfig Config
	location      *time.Location
)

func Init() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	configPath := filepath.Join(filepath.Dir(execPath), ".env")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			currentConfig = DefaultConfig
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &currentConfig); err != nil {
			return err
		}
	}

	// 如果时区为空，使用默认时区
	if currentConfig.TimeZone == "" {
		currentConfig.TimeZone = DefaultConfig.TimeZone
	}

	// 加载时区
	location, err = time.LoadLocation(currentConfig.TimeZone)
	if err != nil {
		return err
	}

	return nil
}

// 获取当前时区的时间
func Now() time.Time {
	return time.Now().In(location)
}

// 格式化时间
func FormatTime(t time.Time, layout string) string {
	return t.In(location).Format(layout)
}
