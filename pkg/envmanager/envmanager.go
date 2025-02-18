package envmanager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	installedPath = "/usr/local/bin/aspnmy_envloader"
	localPath     = "env_loader/aspnmy_envloader"
	testEnvKey    = "testenv"
	testEnvValue  = "test_value_123"
	envLoaderDirKey = "aspnmy_envloaderDir"  // 新增：组件路径环境变量名
)

// verifyEnvLoaderComponent 验证组件是否可用
func verifyEnvLoaderComponent(path string) error {
	cmd := exec.Command(path, "ver")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("组件验证失败: %v", err)
	}
	if len(output) == 0 {
		return fmt.Errorf("组件验证失败: 未返回版本信息")
	}
	return nil
}

// updateSystemPath 将组件路径添加到系统PATH
func updateSystemPath(componentDir string) error {
	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, componentDir) {
		newPath := fmt.Sprintf("%s:%s", currentPath, componentDir)
		if err := os.Setenv("PATH", newPath); err != nil {
			return fmt.Errorf("更新PATH环境变量失败: %v", err)
		}
	}
	return nil
}

// findEnvLoader 查找 aspnmy_envloader 组件
func findEnvLoader() (string, error) {
	// 1. 首先检查环境变量中是否已存在有效的组件路径
	if envDir := os.Getenv(envLoaderDirKey); envDir != "" {
		loaderPath := filepath.Join(envDir, filepath.Base(localPath))
		if _, err := os.Stat(loaderPath); err == nil {
			if err := verifyEnvLoaderComponent(loaderPath); err == nil {
				return loaderPath, nil
			}
		}
	}

	// 2. 如果环境变量中的路径无效，执行常规查找逻辑
	var loaderPath string

	findAndRegisterLoader := func(path string) (string, error) {
		if err := verifyEnvLoaderComponent(path); err == nil {
			// 更新系统 PATH
			if err := updateSystemPath(filepath.Dir(path)); err != nil {
				return "", err
			}
			// 保存组件路径到环境变量
			if err := UpdateEnvironmentVariable(envLoaderDirKey, filepath.Dir(path)); err != nil {
				return "", fmt.Errorf("保存组件路径到环境变量失败: %v", err)
			}
			return path, nil
		}
		return "", fmt.Errorf("组件验证失败")
	}

	// 1. 直接检查系统安装路径
	if _, err := os.Stat(installedPath); err == nil {
		loaderPath = installedPath
		if path, err := findAndRegisterLoader(loaderPath); err == nil {
			return path, nil
		}
	}

	// 2. 检查可执行文件所在目录
	execPath, err := os.Executable()
	if err == nil {
		execLocalPath := filepath.Join(filepath.Dir(execPath), localPath)
		if _, err := os.Stat(execLocalPath); err == nil {
			loaderPath = execLocalPath
			if path, err := findAndRegisterLoader(loaderPath); err == nil {
				return path, nil
			}
		}
	}

	// 3. 检查模块自身目录
	if _, filename, _, ok := runtime.Caller(0); ok {
		moduleLocalPath := filepath.Join(filepath.Dir(filename), "../..", localPath)
		if _, err := os.Stat(moduleLocalPath); err == nil {
			loaderPath = moduleLocalPath
			if path, err := findAndRegisterLoader(loaderPath); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("未找到可用的 aspnmy_envloader 组件或组件验证失败")
}

// ExecEnvLoader 执行环境变量加载器命令
func ExecEnvLoader(command string) error {
	loaderPath, err := findEnvLoader()
	if err != nil {
		return fmt.Errorf("查找 aspnmy_envloader 失败: %v", err)
	}

	cmd := exec.Command(loaderPath, command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 %s %s 失败: %w", loaderPath, command, err)
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

// VerifyEnvLoader 验证环境变量加载器是否正常工作
func VerifyEnvLoader() error {
	// 写入测试变量
	if err := UpdateEnvironmentVariable(testEnvKey, testEnvValue); err != nil {
		return fmt.Errorf("写入测试环境变量失败: %v", err)
	}

	// 验证是否能正确读取
	value := os.Getenv(testEnvKey)
	if value != testEnvValue {
		return fmt.Errorf("验证失败: 期望值 %s, 实际值 %s", testEnvValue, value)
	}

	// 清理测试变量
	if err := UpdateEnvironmentVariable(testEnvKey, ""); err != nil {
		return fmt.Errorf("清理测试环境变量失败: %v", err)
	}

	return nil
}

// RemoveEnvironmentVariable 删除环境变量
func RemoveEnvironmentVariable(key string) error {
	// 验证环境变量加载器
	if err := VerifyEnvLoader(); err != nil {
		return fmt.Errorf("环境变量加载器验证失败: %v", err)
	}

	// 1. 获取 .env 文件路径
	BaseDir := os.Getenv("ollama_scannerBaseDir")
	envFile := filepath.Join(BaseDir, ".env")

	// 2. 读取当前内容
	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("读取 .env 文件失败: %v", err)
	}

	// 3. 按行分割并过滤要删除的变量
	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))
	found := false

	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			found = true
			continue
		}
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	if !found {
		log.Printf("警告: 未找到要删除的环境变量: %s", key)
		return nil
	}

	// 4. 写回文件
	if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %v", err)
	}

	// 5. 重新加载环境变量
	if err := ReloadEnv(); err != nil {
		return fmt.Errorf("重新加载环境变量失败: %v", err)
	}

	// 6. 验证删除结果
	if value := os.Getenv(key); value != "" {
		return fmt.Errorf("环境变量删除失败：%s 仍然存在，值为 %s", key, value)
	}

	log.Printf("已删除环境变量: %s", key)
	return nil
}
