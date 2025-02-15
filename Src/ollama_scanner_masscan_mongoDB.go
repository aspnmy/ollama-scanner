package main

import (
    "bufio"
    "bytes"
    "context"
    "encoding/csv"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "runtime"
    "sort"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

const (
    port            = 11434
    timeout         = 3 * time.Second
    maxWorkers      = 200
    maxIdleConns    = 100
    idleConnTimeout = 90 * time.Second
    benchTimeout    = 30 * time.Second
    defaultCSVFile  = "results.csv"
    mongoURI        = "mongodb://localhost:27017"
    dbName          = "scan_results"
    collection      = "results"
)

var (
    gatewayMAC   = flag.String("gateway-mac", "", "指定网关MAC地址(格式:aa:bb:cc:dd:ee:ff)")
    inputFile    = flag.String("input", "ip.txt", "输入文件路径(CIDR格式列表)")
    outputFile   = flag.String("output", defaultCSVFile, "CSV输出文件路径")
    disableBench = flag.Bool("no-bench", false, "禁用性能基准测试")
    benchPrompt  = flag.String("prompt", "为什么太阳会发光？用一句话回答", "性能测试提示词")
    masscanRate  = flag.Int("rate", 1000, "masscan 扫描速率 (每秒扫描的包数)")
    httpClient   *http.Client
    csvWriter    *csv.Writer
    csvFile      *os.File
    resultsChan  chan ScanResult
    allResults   []ScanResult
    mu           sync.Mutex
    mongoClient  *mongo.Client
)

type ScanResult struct {
    IP     string      `json:"ip"`
    Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
    Name           string        `json:"name"`
    FirstTokenDelay time.Duration `json:"first_token_delay"`
    TokensPerSec   float64       `json:"tokens_per_sec"`
    Status         string        `json:"status"`
}

func init() {
    flag.Usage = func() {
        helpText := `Ollama节点扫描工具 v2.2 https://t.me/Ollama_Scanner
默认功能:
- 自动执行性能测试
- 结果导出到%s
使用方法:
%s [参数]
参数说明:`
        fmt.Fprintf(os.Stderr, helpText, defaultCSVFile, os.Args[0])
        flag.PrintDefaults()
        fmt.Fprintf(os.Stderr, `
示例:
%s -gateway-mac aa:bb:cc:dd:ee:ff
%s -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
%s -gateway-mac aa:bb:cc:dd:ee:ff -rate 5000
`, os.Args[0], os.Args[0], os.Args[0])
    }

    httpClient = &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        maxIdleConns,
            MaxIdleConnsPerHost: maxIdleConns,
            IdleConnTimeout:     idleConnTimeout,
        },
        Timeout: timeout,
    }
    resultsChan = make(chan ScanResult, 100)
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
    flag.Parse()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 初始化 MongoDB
    if err := initMongoDB(); err != nil {
        log.Fatalf("❌ MongoDB 初始化失败: %v\n", err)
    }
    defer mongoClient.Disconnect(context.Background())

    if err := checkAndInstallMasscan(); err != nil {
        log.Printf("❌ 无法安装 masscan: %v\n 请手动安装 masscan 后重试\n", err)
        fmt.Printf("请确认已安装 masscan,或手动安装后重试 (https://github.com/robertdavidgraham/masscan)\n")
        fmt.Printf("是否跳过自动安装 masscan 并继续执行程序？ (y/n): ")
        var answer string
        fmt.Scanln(&answer)
        if strings.ToLower(answer) != "y" {
            os.Exit(1)
        }
    }

    initCSVWriter()
    defer csvFile.Close()
    setupSignalHandler(cancel)
    if err := runScanProcess(ctx); err != nil {
        fmt.Printf("❌ 扫描失败: %v\n", err)
    }
}

func initMongoDB() error {
    clientOptions := options.Client().ApplyURI(mongoURI)
    client, err := mongo.Connect(context.Background(), clientOptions)
    if err != nil {
        return fmt.Errorf("无法连接 MongoDB: %w", err)
    }
    mongoClient = client
    return nil
}



func checkAndInstallMasscan() error {
	_, err := exec.LookPath("masscan")
	if err == nil {
		log.Println("masscan 已安装")
		return nil
	}

	log.Println("masscan 未安装, 尝试自动安装...")
	var cmd *exec.Cmd
	var installErr error
	osName := runtime.GOOS
	log.Printf("Operating System: %s\n", osName)

	switch osName {
	case "linux":
		err = exec.Command("apt", "-v").Run()
		if err == nil {
			cmd = exec.Command("sudo", "-u", "root", "/usr/bin/apt-get", "update")
			installErr = cmd.Run()
			if installErr != nil {
				log.Printf("apt-get update failed: %v\n", installErr)
				return fmt.Errorf("apt-get update failed: %w", installErr)
			}

			cmd = exec.Command("sudo", "-u", "root", "/usr/bin/apt-get", "install", "-y", "masscan")
			installErr = cmd.Run()
			if installErr != nil {
				log.Printf("apt-get install masscan failed: %v\n", installErr)
				return fmt.Errorf("apt-get install masscan failed: %w", installErr)
			}
		} else {
			err = exec.Command("yum", "-v").Run()
			if err == nil {
				cmd = exec.Command("sudo", "-u", "root", "/usr/bin/yum", "install", "-y", "masscan")
				installErr = cmd.Run()
				if installErr != nil {
					log.Printf("yum install masscan failed: %v\n", installErr)
					return fmt.Errorf("yum install masscan failed: %w", installErr)
				}
			} else {
				return fmt.Errorf("apt and yum not found, cannot install masscan automatically. Please install manually")
			}
		}
	case "darwin":
		_, brewErr := exec.LookPath("brew")
		if brewErr != nil {
			return fmt.Errorf("brew is not installed, cannot install masscan automatically. Please install manually")
		}

		cmd = exec.Command("brew", "install", "masscan")
		installErr = cmd.Run()
		if installErr != nil {
			return fmt.Errorf("brew install masscan failed: %w", installErr)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s, cannot install masscan automatically. Please install manually", osName)
	}

	log.Println("masscan 安装完成")
	return nil
}

func initCSVWriter() {
	var err error
	csvFile, err = os.Create(*outputFile)
	if err != nil {
		fmt.Printf("⚠️ 创建CSV文件失败: %v\n", err)
		return
	}

	csvWriter = csv.NewWriter(csvFile)
	headers := []string{"IP地址", "模型名称", "状态"}
	if !*disableBench {
		headers = append(headers, "首Token延迟(ms)", "Tokens/s")
	}
	csvWriter.Write(headers)
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		fmt.Println("\n🛑 收到终止信号,正在清理资源...")
		if csvWriter != nil {
			csvWriter.Flush()
		}
		os.Exit(1)
	}()
}

func runScanProcess(ctx context.Context) error {
	if err := validateInput(); err != nil {
		return err
	}

	fmt.Printf("🔍 开始扫描,使用网关: %s\n", *gatewayMAC)
	if err := execMasscan(); err != nil {
		return err
	}

	return processResults(ctx)
}

func validateInput() error {
	if *gatewayMAC == "" {
		return fmt.Errorf("必须指定网关MAC地址")
	}

	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		return fmt.Errorf("输入文件不存在: %s", *inputFile)
	}

	return nil
}

func execMasscan() error {
	cmd := exec.Command("masscan",
		"-p", fmt.Sprintf("%d", port),
		"--rate", fmt.Sprintf("%d", *masscanRate),
		"-iL", *inputFile,
		"-oG", *outputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func processResults(ctx context.Context) error {
	file, err := os.Open(*outputFile)
	if err != nil {
		return fmt.Errorf("打开结果文件失败: %w", err)
	}
	defer file.Close()

	ips := make(chan string, maxWorkers*2)
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, ips)
	}

	var rhWg sync.WaitGroup
	rhWg.Add(1)
	go func() {
		defer rhWg.Done()
		resultHandler()
	}()

	go func() {
		defer close(ips)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Host:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ip := parts[1]
					if net.ParseIP(ip) != nil {
						ips <- ip
					}
				}
			}
		}
	}()

	wg.Wait()
	close(resultsChan)
	rhWg.Wait()
	csvWriter.Flush()

	fmt.Printf("\n✅ 结果已保存至 %s\n", *outputFile)
	return nil
}

func resultHandler() {
	for res := range resultsChan {
		printResult(res)
		writeCSV(res)
	}
}

func printResult(res ScanResult) {
	fmt.Printf("\nIP地址: %s\n", res.IP)
	fmt.Println(strings.Repeat("-", 50))
	for _, model := range res.Models {
		fmt.Printf("├─ 模型: %-25s\n", model.Name)
		if !*disableBench {
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
	for _, model := range res.Models {
		record := []string{res.IP, model.Name, model.Status}
		if !*disableBench {
			record = append(record,
				fmt.Sprintf("%.0f", model.FirstTokenDelay.Seconds()*1000),
				fmt.Sprintf("%.1f", model.TokensPerSec))
		}
		err := csvWriter.Write(record)
		if err != nil {
			fmt.Printf("⚠️ 写入CSV失败: %v\n", err)
		}
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, ips <-chan string) {
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
						if !*disableBench {
							latency, tps, status := benchmarkModel(ip, model)
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

func checkPort(ip string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func checkOllama(ip string) bool {
	resp, err := httpClient.Get(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	return strings.Contains(string(buf[:n]), "Ollama is running")
}

func getModels(ip string) []string {
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

func benchmarkModel(ip, model string) (time.Duration, float64, string) {
	if *disableBench {
		return 0, 0, "未测试"
	}

	start := time.Now()
	payload := map[string]interface{}{
		"model":  model,
		"prompt": *benchPrompt,
		"stream": true,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("http://%s:%d/api/generate", ip, port),
		bytes.NewReader(body))
	client := &http.Client{Timeout: benchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, "连接失败"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Sprintf("HTTP %d", resp.StatusCode)
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
		return 0, 0, "无有效响应"
	}

	totalTime := lastToken.Sub(start)
	return firstToken.Sub(start), float64(tokenCount)/totalTime.Seconds(), "成功"
}
