package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TaskManager struct {
	TaskCounter int    `json:"task_counter"`
	LastUpdate  string `json:"last_update"`
}

const (
	taskStateFile   = ".task_state"
	defaultTaskName = "模型扫描任务"
)

// 获取下一个任务名称
func GetNextTaskName() (string, error) {
	tm, err := loadTaskManager()
	if err != nil {
		return "", err
	}

	tm.TaskCounter++
	tm.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	if err := saveTaskManager(tm); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%d", defaultTaskName, tm.TaskCounter), nil
}

// 加载任务管理器状态
func loadTaskManager() (*TaskManager, error) {
	tm := &TaskManager{TaskCounter: 0}

	execDir, err := getExecutableDir()
	if err != nil {
		return tm, nil
	}

	statePath := filepath.Join(execDir, taskStateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return tm, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, tm); err != nil {
		return tm, nil
	}

	return tm, nil
}

// 保存任务管理器状态
func saveTaskManager(tm *TaskManager) error {
	execDir, err := getExecutableDir()
	if err != nil {
		return err
	}

	statePath := filepath.Join(execDir, taskStateFile)
	data, err := json.MarshalIndent(tm, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

// 获取可执行文件目录
func getExecutableDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}
