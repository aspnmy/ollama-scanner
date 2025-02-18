// v2.2.1 增加断点续扫功能 支持进度条显示
// 自动获取 eth0 网卡的 MAC 地址
// 在以下情况下尝试自动获取 MAC 地址：
// 配置文件不存在时
// 配置文件中的 MAC 地址为空时
// 命令行参数未指定 MAC 地址时
// 获取失败时给出相应的错误提示
// 合并组件zmap和masscan，根据操作系统自动选择扫描器
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aspnmy/ollama_scanner_envmanager"
)

const (
	defaultPort        = 11434 // 修改为 defaultPort
	timeout            = 3 * time.Second
	maxWorkers         = 200
	maxIdleConns       = 100
	idleConnTimeout    = 90 * time.Second
	benchTimeout       = 30 * time.Second
	defaultCSVFile     = "results.csv"
	defaultZmapThreads = 10   // zmap 默认线程数
	defaultMasscanRate = 1000 // masscan 默认扫描速率
	defaultBenchPrompt = "为什么太阳会发光？用一句话回答"
)

// init 函数放在最上方
func init() {
	// 先执行 reloadEnv 加载配置文件
	if err := envmanager.ReloadEnv(); err != nil {
		log.Fatalf("初始化环境变量失败: %v", err)
	}

	// 初始化默认值
	if err := initDefaultValues(); err != nil {
		log.Fatalf("初始化默认值失败: %v", err)
	}
}

type ScanResult struct {
	IP     string
	Models []ModelInfo
}

type ModelInfo struct {
	Name            string
	FirstTokenDelay time.Duration
	TokensPerSec    float64
	Status          string
}

// 添加获取MAC地址的函数
func getEth0MAC() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("获取网络接口失败: %w", err)
	}

	for _, iface := range ifaces {
		// 查找 eth0 接口
		if iface.Name == "eth0" {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				return mac, nil
			}
		}
	}
	return "", fmt.Errorf("未找到 eth0 网卡或获取MAC地址失败")
}

type Progress struct {
	mu        sync.Mutex
	total     int
	current   int
	startTime time.Time
}

func (p *Progress) Init(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
	p.current = 0
	p.startTime = time.Now()
}

func (p *Progress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	p.printProgress()
}

func (p *Progress) printProgress() {
	percentage := float64(p.current) / float64(p.total) * 100
	elapsed := time.Since(p.startTime)
	remainingTime := time.Duration(0)
	if p.current > 0 {
		remainingTime = time.Duration(float64(elapsed) / float64(p.current) * float64(p.total-p.current))
	}
	fmt.Printf("\r当前进度: %.1f%% (%d/%d) 已用时: %v 预计剩余: %v",
		percentage, p.current, p.total, elapsed.Round(time.Second), remainingTime.Round(time.Second))
}

var (
	resultsChan chan ScanResult
	csvFile     *os.File
	csvWriter   *csv.Writer
)

// main 函数是程序的入口点,负责初始化程序、检查并安装 zmap、设置信号处理和启动扫描过程.
func main() {
	// 解析命令行参数
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultsChan = make(chan ScanResult, 100)

	// 初始化扫描器
	if err := checkAndInstallZmap(); err != nil {
		log.Printf("❌ 初始化扫描器失败: %v\n", err)
		fmt.Printf("是否继续执行程序？(y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "y" {
			os.Exit(1)
		}
	}

	// 初始化 CSV 写入器,用于将扫描结果保存到文件中
	initCSVWriter()
	// 确保在函数退出时关闭 CSV 文件
	defer csvFile.Close()
	// 设置信号处理,以便在收到终止信号时清理资源并退出程序
	setupSignalHandler(cancel)
	// 启动扫描过程,如果扫描失败则打印错误信息
	if err := runScanProcess(ctx); err != nil {
		fmt.Printf("❌ 扫描失败: %v\n", err)
	}
}

// checkAndInstallZmap 检查系统中是否安装了 zmap,如果未安装则尝试自动安装.
// 支持的操作系统包括 Linux(Debian/Ubuntu 使用 apt,CentOS/RHEL 使用 yum)和 macOS(使用 brew).
// 如果不支持当前操作系统或安装过程中出现错误,将返回相应的错误信息.
func checkAndInstallZmap() error {
	// 检查 zmap 是否已经安装
	_, err := exec.LookPath("zmap")
	if err == nil {
		// zmap 已安装
		log.Println("zmap 已安装")
		return nil
	}

	// zmap 未安装,尝试自动安装
	log.Println("zmap 未安装, 尝试自动安装...")
	var installErr error
	// 获取当前操作系统名称
	osName := runtime.GOOS
	log.Printf("Operating System: %s\n", osName)

	// 打印当前环境变量,方便调试
	log.Println("当前环境变量:")
	for _, env := range os.Environ() {
		log.Println(env)
	}

	// 根据不同的操作系统选择不同的安装方式
	switch osName {
	case "linux":
		// 在 Linux 系统上,尝试使用 apt(Debian/Ubuntu)或 yum(CentOS/RHEL)安装 zmap
		// 首先尝试使用 apt
		err = exec.Command("apt", "-v").Run()
		if err == nil {
			// 使用 sudo -u root 明确指定用户身份执行 apt-get update
			cmd := exec.Command("sudo", "-u", "root", "/usr/bin/apt-get", "update")
			installErr = cmd.Run()
			if installErr != nil {
				log.Printf("apt-get update failed: %v\n", installErr)
				return fmt.Errorf("apt-get update failed: %w", installErr)
			}

			// 使用 sudo -u root 明确指定用户身份执行 apt-get install zmap
			cmd = exec.Command("sudo", "-u", "root", "/usr/bin/apt-get", "install", "-y", "zmap")
			installErr = cmd.Run()
			if installErr != nil {
				log.Printf("apt-get install zmap failed: %v\n", installErr)
				return fmt.Errorf("apt-get install zmap failed: %w", installErr)
			}

		} else {
			// 如果 apt 不可用,尝试使用 yum
			err = exec.Command("yum", "-v").Run()
			if err == nil {
				// 使用 sudo -u root 明确指定用户身份执行 yum install zmap
				cmd := exec.Command("sudo", "-u", "root", "/usr/bin/yum", "install", "-y", "zmap")
				installErr = cmd.Run()
				if installErr != nil {
					log.Printf("yum install zmap failed: %v\n", installErr)
					return fmt.Errorf("yum install zmap failed: %w", installErr)
				}

			} else {
				return fmt.Errorf("apt and yum not found, cannot install zmap automatically. Please install manually")
			}
		}
	case "darwin":
		// 在 macOS 系统上,使用 brew 安装 zmap
		_, brewErr := exec.LookPath("brew")
		if brewErr != nil {
			return fmt.Errorf("未安装 brew，无法自动安装 zmap。请手动安装")
		}

		cmd := exec.Command("brew", "install", "zmap")
		installErr = cmd.Run()
		if installErr != nil {
			return fmt.Errorf("使用 brew 安装 zmap 失败: %w", installErr)
		}
	default:
		return fmt.Errorf("不支持的操作系统: %s，无法自动安装 zmap。请手动安装", osName)
	}

	log.Println("zmap 安装完成")
	return nil
}

// initCSVWriter 函数用于初始化 CSV 写入器,创建 CSV 文件并写入表头.
func initCSVWriter() {
	var err error

	// 获取输出文件路径
	outputFile := os.Getenv("OUTPUT_FILE")
	if outputFile == "" {
		if err := envmanager.UpdateEnvironmentVariable("OUTPUT_FILE", defaultCSVFile); err != nil {
			fmt.Printf("⚠️ 设置默认输出文件失败: %v\n", err)
			return
		}
		outputFile = os.Getenv("OUTPUT_FILE")
	}

	// 如果路径不是绝对路径，则使用当前目录
	if !filepath.IsAbs(outputFile) {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("⚠️ 获取当前目录失败: %v\n", err)
			return
		}
		outputFile = filepath.Join(currentDir, outputFile)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("⚠️ 创建输出目录失败: %v\n", err)
		return
	}

	// 创建 CSV 文件
	csvFile, err = os.Create(outputFile)
	if err != nil {
		fmt.Printf("⚠️ 创建CSV文件失败: %v\n", err)
		return
	}

	// 创建 CSV 写入器并写入表头
	csvWriter = csv.NewWriter(csvFile)
	headers := []string{"IP地址", "模型名称", "状态"}
	if os.Getenv("disableBench") != "true" {
		headers = append(headers, "首Token延迟(ms)", "Tokens/s")
	}
	if err := csvWriter.Write(headers); err != nil {
		fmt.Printf("⚠️ 写入CSV表头失败: %v\n", err)
		return
	}

	fmt.Printf("📝 CSV文件已创建: %s\n", outputFile)
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		fmt.Println("\n⚠️ 收到终止信号，正在保存进度...")
		if csvWriter != nil {
			csvWriter.Flush()
		}
		if csvFile != nil {
			csvFile.Close()
		}
		os.Exit(1)
	}()
}

func runScanProcess(ctx context.Context) error {
	// 先设置 MAC 地址
	if err := setupGatewayMAC(); err != nil {
		return err
	}

	if err := validateInput(); err != nil {
		return err
	}
	gatewayMAC := os.Getenv("GATEWAY_MAC")
	fmt.Printf("🔍 开始扫描目标，使用网关MAC: %s\n", gatewayMAC)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Start goroutine to handle results
		go handleScanResults()
		if err := execScan(); err != nil {
			return err
		}
		close(resultsChan)
		return nil
	}
}

// 简化后的 processScanResults 函数
func processScanResults() error {
	outputFile := os.Getenv("OUTPUT_FILE")
	file, err := os.Open(outputFile)
	if err != nil {
		return fmt.Errorf("打开结果文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if checkPort(ip) && checkOllama(ip) {
			result := ScanResult{IP: ip}
			if models := getModels(ip); len(models) > 0 {
				models = sortModels(models)
				for _, model := range models {
					info := ModelInfo{Name: model}
					if os.Getenv("disableBench") != "true" {
						latency, tps, status := benchmarkModel(ip, model, os.Getenv("benchPrompt"))
						info.FirstTokenDelay = latency
						info.TokensPerSec = tps
						info.Status = status
					} else {
						info.Status = "发现"
					}
					result.Models = append(result.Models, info)
				}
				printResult(result)
				writeCSV(result)
			}
		}
	}

	if csvWriter != nil {
		csvWriter.Flush()
	}

	fmt.Printf("\n✅ 扫描完成，结果已保存到: %s\n", outputFile)
	return nil
}

func handleScanResults() {
	for res := range resultsChan {
		printResult(res)
		writeCSV(res)
	}
}

func printResult(res ScanResult) {
	disableBench := os.Getenv("disableBench")
	fmt.Printf("\nIP地址: %s\n", res.IP)
	fmt.Println(strings.Repeat("-", 50))
	for _, model := range res.Models {
		fmt.Printf("├─ 模型: %-25s\n", model.Name)
		if disableBench != "true" {
			fmt.Printf("│ ├─ 状态: %s\n", model.Status)
			fmt.Printf("│ ├─ 首Token延迟: %v\n", model.FirstTokenDelay.Round(time.Millisecond))
			fmt.Printf("│ └─ 生成速度: %.1f tokens/s\n", model.TokensPerSec)
		} else {
			fmt.Printf("│ └─ 状态: %s\n", model.Status)
		}
		fmt.Println(strings.Repeat("-", 50))
	}
}

func writeCSV(res ScanResult) {
	disableBench := os.Getenv("disableBench")
	for _, model := range res.Models {
		record := []string{res.IP, model.Name, model.Status}
		if disableBench != "true" {
			record = append(record,
				fmt.Sprintf("%.0f", model.FirstTokenDelay.Seconds()*1000),
				fmt.Sprintf("%.1f", model.TokensPerSec))
		}
		if csvWriter != nil {
			err := csvWriter.Write(record)
			if err != nil {
				fmt.Printf("⚠️ 写入CSV失败: %v\n", err) // Handle the error appropriately
			}
		}
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, ips <-chan string) {
	disableBench := os.Getenv("disableBench")
	benchPrompt := os.Getenv("benchPrompt")
	defer wg.Done()
	for ip := range ips {
		select {
		case <-ctx.Done():
			return
		default:
			if checkPort(ip) && checkOllama(ip) {
				result := ScanResult{IP: ip}
				if models := getModels(ip); len(models) > 0 {
					models = sortModels(models)
					for _, model := range models {
						info := ModelInfo{Name: model}
						if disableBench != "true" {
							latency, tps, status := benchmarkModel(ip, model, benchPrompt)
							info.FirstTokenDelay = latency
							info.TokensPerSec = tps
							info.Status = status
						} else {
							info.Status = "发现"
						}
						result.Models = append(result.Models, info)
					}
					resultsChan <- result
				}
			}
		}
	}
}

// 修改 setupGatewayMAC 函数使用新的环境变量更新函数
func setupGatewayMAC() error {
	mac := os.Getenv("GATEWAY_MAC")
	if mac == "" {
		var err error
		mac, err = getEth0MAC()
		if err != nil {
			return fmt.Errorf("必须指定网关MAC地址,自动获取失败: %v", err)
		}

		if err := envmanager.UpdateEnvironmentVariable("GATEWAY_MAC", mac); err != nil {
			return fmt.Errorf("更新 MAC 地址失败: %v", err)
		}
	}
	return nil
}

// 修改 validateInput 函数，移除 MAC 地址相关的逻辑
func validateInput() error {
	// 获取脚本所在目录
	scriptDir, err := getScriptDir()
	if err != nil {
		return fmt.Errorf("获取脚本目录失败: %v", err)
	}

	// 优先使用命令行参数中的路径
	// 如果命令行参数是相对路径且配置文件中有绝对路径，则使用配置文件中的路径
	inputFile := os.Getenv("INPUT_FILE")
	if !filepath.IsAbs(inputFile) {
		inputFile = filepath.Join(scriptDir, inputFile)
	}
	log.Printf("使用输入文件: %s", inputFile)

	outputFile := os.Getenv("OUTPUT_FILE")
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(scriptDir, outputFile)
	}
	log.Printf("使用输出文件: %s", outputFile)

	// 检查输入文件是否存在
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		// 如果文件不存在，创建一个空文件
		emptyFile, err := os.Create(inputFile)
		if err != nil {
			return fmt.Errorf("创建输入文件失败: %v", err)
		}
		emptyFile.Close()
		log.Printf("创建了空的输入文件: %s", inputFile)
		return fmt.Errorf("请在输入文件中添加要扫描的IP地址: %s", inputFile)
	}

	return nil
}

// 获取脚本所在目录的新函数
func getScriptDir() (string, error) {
	// 尝试使用 os.Executable() 获取可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取可执行文件的实际路径（处理符号链接）
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("解析符号链接失败: %v", err)
	}

	// 获取目录路径
	dir := filepath.Dir(realPath)

	// 验证目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("脚本目录不存在: %v", err)
	}

	return dir, nil
}

func execScan() error {
	scannerType := os.Getenv("scannerType")
	if scannerType == "masscan" {
		return execMasscan()
	}
	return execZmap()
}

func execMasscan() error {
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	masscanRate := os.Getenv("masscanRate")
	gatewayMAC := os.Getenv("GATEWAY_MAC")
	inputFile := os.Getenv("INPUT_FILE")
	outputFile := os.Getenv("OUTPUT_FILE")
	cmd := exec.Command("masscan",
		"-p", OLLAMA_PORT,
		"--rate", masscanRate,
		"--interface", "eth0",
		"--source-ip", gatewayMAC,
		"-iL", inputFile,
		"-oL", outputFile)

	out, err := cmd.CombinedOutput()
	fmt.Printf("masscan 输出:\n%s\n", string(out))
	return err
}

func execZmap() error {
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	zmapThreads := os.Getenv("zmapThreads")
	gatewayMAC := strings.Trim(os.Getenv("GATEWAY_MAC"), "'") // 移除可能存在的单引号
	inputFile := os.Getenv("INPUT_FILE")
	outputFile := os.Getenv("OUTPUT_FILE")

	// 打印调试信息
	log.Printf("DEBUG: MAC地址: %s", gatewayMAC)
	log.Printf("DEBUG: 完整命令: zmap -p %s -G %s -w %s -o %s -T %s",
		OLLAMA_PORT, gatewayMAC, inputFile, outputFile, zmapThreads)

	cmd := exec.Command("zmap",
		"-p", fmt.Sprintf("%s", OLLAMA_PORT),
		"-G", gatewayMAC,
		"-w", inputFile,
		"-o", outputFile,
		"-T", fmt.Sprintf("%s", zmapThreads))

	out, err := cmd.CombinedOutput()
	fmt.Printf("zmap 输出:\n%s\n", string(out))
	return err
}

func checkPort(ip string) bool {
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	if OLLAMA_PORT == "" {
		if err := envmanager.UpdateEnvironmentVariable("OLLAMA_PORT", strconv.Itoa(defaultPort)); err != nil {
			log.Printf("设置默认端口失败: %v", err)
			return false
		}
		OLLAMA_PORT = os.Getenv("OLLAMA_PORT")
	}
	port, _ := strconv.Atoi(OLLAMA_PORT)
	result := net.Dialer{Timeout: timeout}
	conn, err := result.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}
	conn.Close()
	//enableLog := os.Getenv("enableLog")
	//if enableLog == "true" {
	//	logger.LogScanResult(ip, nil, fmt.Sprintf("端口检查: %v", result))
	//}
	return true
}

func checkOllama(ip string) bool {
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	port, _ := strconv.Atoi(OLLAMA_PORT)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("http://%s:%d", ip, port), nil)
	if err != nil {
		return false
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	return strings.Contains(string(buf[:n]), "Ollama is running")
}
func getModels(ip string) []string {
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	port, _ := strconv.Atoi(OLLAMA_PORT)
	httpClient := &http.Client{}
	resp, err := httpClient.Get(fmt.Sprintf("http://%s:%d/api/tags", ip, port))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var data struct {
		Models []struct {
			Model string `json:"model"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	var models []string
	for _, m := range data.Models {
		if strings.Contains(m.Model, "deepseek-r1") {
			models = append(models, m.Model)
		}
	}
	return models
}

func parseModelSize(model string) float64 {
	parts := strings.Split(model, ":")
	if len(parts) < 2 {
		return 0
	}

	sizeStr := strings.TrimSuffix(parts[len(parts)-1], "b")
	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0
	}

	return size
}

func sortModels(models []string) []string {
	sort.Slice(models, func(i, j int) bool {
		return parseModelSize(models[i]) < parseModelSize(models[j])
	})
	return models
}

func benchmarkModel(ip string, model string, benchPrompt string) (time.Duration, float64, string) {
	disableBench := os.Getenv("disableBench")
	if disableBench == "" {
		if err := envmanager.UpdateEnvironmentVariable("disableBench", "false"); err != nil {
			log.Printf("设置默认 disableBench 失败: %v", err)
			return 0, 0, "系统配置错误"
		}
		disableBench = os.Getenv("disableBench")
	}

	if benchPrompt == "" {
		if err := envmanager.UpdateEnvironmentVariable("benchPrompt", defaultBenchPrompt); err != nil {
			log.Printf("设置默认 benchPrompt 失败: %v", err)
			return 0, 0, "系统配置错误"
		}
		benchPrompt = os.Getenv("benchPrompt")
	}
	OLLAMA_PORT := os.Getenv("OLLAMA_PORT")
	port, _ := strconv.Atoi(OLLAMA_PORT)
	start := time.Now()
	payload := map[string]interface{}{
		"model":  model,
		"prompt": benchPrompt,
		"stream": true,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("http://%s:%d/api/generate", ip, port),
		bytes.NewReader(body))
	httpClient := &http.Client{Timeout: benchTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, 0, "连接失败"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Sprintf("HTTP错误: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var (
		firstToken time.Time
		lastToken  time.Time
		tokenCount int
	)

	for scanner.Scan() {
		if tokenCount == 0 {
			firstToken = time.Now()
		}

		lastToken = time.Now()
		tokenCount++
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}

		if done, _ := data["done"].(bool); done {
			break
		}
	}

	if tokenCount == 0 {
		return 0, 0, "无响应"
	}

	totalTime := lastToken.Sub(start)
	return firstToken.Sub(start), float64(tokenCount) / totalTime.Seconds(), "完成"
}

// 添加默认值初始化函数
func initDefaultValues() error {
	defaults := map[string]string{
		"OLLAMA_PORT":  "11434",
		"disableBench": "false",
		"masscanRate":  "1000",
		"zmapThreads":  "10",
		"benchPrompt":  "为什么太阳会发光？用一句话回答",
		"OUTPUT_FILE":  "results.csv",
		"INPUT_FILE":   "ip.txt",
		"ENABLE_LOG":   "true",
		"LOG_LEVEL":    "info",
	}

	for key, defaultValue := range defaults {
		currentValue := os.Getenv(key)
		if currentValue == "" {
			if err := envmanager.UpdateEnvironmentVariable(key, defaultValue); err != nil {
				return fmt.Errorf("初始化默认值失败 %s=%s: %v", key, defaultValue, err)
			}
		}
	}
	return nil
}
