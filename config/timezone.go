package config

import (
	"fmt"
	"time"
)

var (
	defaultTimeZone = "Asia/Shanghai"
	location        *time.Location
)

// 初始化时区
func InitTimeZone(tz string) error {
	if tz == "" {
		tz = defaultTimeZone
	}

	var err error
	location, err = time.LoadLocation(tz)
	if err != nil {
		location, _ = time.LoadLocation(defaultTimeZone)
		return fmt.Errorf("加载时区 %s 失败，使用默认时区 %s: %v",
			tz, defaultTimeZone, err)
	}
	return nil
}

// 获取当前时间（使用配置的时区）
func Now() time.Time {
	return time.Now().In(location)
}

// 格式化时间
func FormatTime(t time.Time, layout string) string {
	return t.In(location).Format(layout)
}
