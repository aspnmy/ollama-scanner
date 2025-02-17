package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logFile *os.File
	Logger  *log.Logger
)

// 初始化日志系统
func Init(logPath string) error {
	// 确保日志目录存在
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 创建或打开日志文件（追加模式）
	file, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 创建多输出的日志记录器
	multiWriter := log.MultiWriter(os.Stdout, file)
	Logger = log.New(multiWriter, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	logFile = file

	Logger.Printf("日志系统初始化完成，日志文件: %s", logPath)
	return nil
}

// 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// 记录扫描结果
func LogScanResult(ip string, models []string, status string) {
	if Logger != nil {
		Logger.Printf("扫描结果 - IP: %s, 模型: %v, 状态: %s", ip, models, status)
	}
}

// 记录性能测试结果
func LogBenchmarkResult(ip string, model string, latency time.Duration, tps float64) {
	if Logger != nil {
		Logger.Printf("性能测试 - IP: %s, 模型: %s, 延迟: %v, TPS: %.2f",
			ip, model, latency, tps)
	}
}

// 记录错误信息
func LogError(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf("错误: "+format, v...)
	}
}
