package changelog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ChangeLog struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"` // add/remove/modify
	Content   string `json:"content"`
	Author    string `json:"author"`
}

type ChangeLogFile struct {
	Version    string      `json:"version"`
	LastUpdate string      `json:"last_update"`
	ChangeLogs []ChangeLog `json:"changelogs"`
	TotalCount int         `json:"total_count"`
}

const (
	changelogFile = "changelog.json"
)

// 添加变更记录
func AddChange(changeType, content, author string) error {
	changelog := ChangeLog{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Type:      changeType,
		Content:   content,
		Author:    author,
	}

	file, err := loadChangeLogFile()
	if err != nil {
		return err
	}

	file.ChangeLogs = append(file.ChangeLogs, changelog)
	file.LastUpdate = changelog.Timestamp
	file.TotalCount++

	return saveChangeLogFile(file)
}

// 加载变更记录文件
func loadChangeLogFile() (*ChangeLogFile, error) {
	execDir, err := getExecutableDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(execDir, "logs", changelogFile)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ChangeLogFile{
			Version:    "v2.2.1",
			ChangeLogs: make([]ChangeLog, 0),
		}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var file ChangeLogFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

// 保存变更记录文件
func saveChangeLogFile(file *ChangeLogFile) error {
	execDir, err := getExecutableDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(execDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(logDir, changelogFile)
	return os.WriteFile(filePath, data, 0644)
}

func getExecutableDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}
