package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnv 加载环境变量文件
func LoadEnv() error {
	// 获取项目根目录
	rootDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	// 读取.env文件
	envPath := filepath.Join(rootDir, ".env")
	content, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("读取环境变量文件失败: %w", err)
	}

	// 解析并设置环境变量
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		os.Setenv(key, value)
	}

	return nil
}

// findProjectRoot 查找项目根目录
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到项目根目录")
		}
		dir = parent
	}
}

// GetEnvWithDefault 获取环境变量，如果不存在则返回默认值
func GetEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
