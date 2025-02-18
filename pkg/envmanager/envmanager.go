package envmanager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecEnvLoader 执行环境变量加载器命令
func ExecEnvLoader(command string) error {
	cmd := exec.Command("aspnmy_envloader", command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 aspnmy_envloader %s 失败: %w", command, err)
	}
	return nil
}

// ReloadEnv 重新加载环境变量
func ReloadEnv() error {
	// 先执行 reload 加载配置文件
	if err := ExecEnvLoader("reload"); err != nil {
		return fmt.Errorf("加载环境变量失败: %v", err)
	}

	// 再 source ~/.bashrc 使环境变量生效
	cmd := exec.Command("bash", "-c", "source ~/.bashrc")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 source ~/.bashrc 失败: %v", err)
	}

	return nil
}

// UpdateEnvironmentVariable 更新环境变量
func UpdateEnvironmentVariable(key, value string) error {
	// 1. 获取 .env 文件路径
	BaseDir := os.Getenv("ollama_scannerBaseDir")
	envFile := filepath.Join(BaseDir, ".env")

	// 2. 读取当前内容
	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("读取 .env 文件失败: %v", err)
	}

	// 3. 按行分割
	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))
	found := false

	// 4. 查找并更新或保留现有行
	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			found = true
			continue
		}
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	// 5. 添加新的环境变量
	newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))

	// 6. 写回文件
	if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %v", err)
	}

	// 7. 重新加载环境变量
	if err := ReloadEnv(); err != nil {
		return fmt.Errorf("重新加载环境变量失败: %v", err)
	}

	// 8. 验证更新结果
	newValue := os.Getenv(key)
	if newValue != value {
		return fmt.Errorf("环境变量更新失败：%s 值不匹配", key)
	}

	if !found {
		log.Printf("新增环境变量: %s=%s", key, value)
	} else {
		log.Printf("更新环境变量: %s=%s", key, value)
	}

	return nil
}
