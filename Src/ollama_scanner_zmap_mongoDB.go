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
	"go.mongodb.org/mongo-driver/bson"
)

const (
	port            = 11434
	timeout         = 3 * time.Second
	maxWorkers      = 200
	maxIdleConns    = 100
	idleConnTimeout = 90 * time.Second
	benchTimeout    = 30 * time.Second
	defaultCSVFile  = "results.csv"
	defaultZmapThreads = 10 // zmap 默认线程数
)

var (
	gatewayMAC  = flag.String("gateway-mac", "", "指定网关MAC地址(格式:aa:bb:cc:dd:ee:ff)")
	inputFile   = flag.String("input", "ip.txt", "输入文件路径(CIDR格式列表)")
	outputFile  = flag.String("output", defaultCSVFile, "CSV输出文件路径")
	disableBench = flag.Bool("no-bench", false, "禁用性能基准测试")
	benchPrompt = flag.String("prompt", "为什么太阳会发光？用一句话回答", "性能测试提示词")
	httpClient  *http.Client
	csvWriter   *csv.Writer
	csvFile     *os.File
    zmapThreads *int	// zmap 线程数
	resultsChan chan ScanResult
	allResults  []ScanResult
	mu          sync.Mutex
	mongoClient *mongo.Client
)

type ScanResult struct {
	IP     string      `bson:"ip"`
	Models []ModelInfo `bson:"models"`
}

type ModelInfo struct {
	Name           string        `bson:"name"`
	FirstTokenDelay time.Duration `bson:"first_token_delay"`
	TokensPerSec   float64       `bson:"tokens_per_sec"`
	Status         string        `bson:"status"`
}

func init() {
	flag.Usage = func() {
		helpText := `Ollama节点扫描工具 v2.2 https://t.me/+YfCVhGWyKxoyMDhl
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
%s -gateway-mac aa:bb:cc:dd:ee:ff -T 20
`, os.Args[0], os.Args[0], os.Args[0]) // 添加 -T 参数的示例

	}

	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConns,
			IdleConnTimeout: idleConnTimeout,
		},
		Timeout: timeout,
	}
    zmapThreads = flag.Int("T", defaultZmapThreads, "zmap 线程数 (默认为 10)")
	resultsChan = make(chan ScanResult, 100)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.Parse()

	// 初始化 MongoDB 客户端
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("无法连接到 MongoDB: %v", err)
	}
	mongoClient = client
	defer mongoClient.Disconnect(ctx)

	// 其他初始化逻辑
	initCSVWriter()
	defer csvFile.Close()
	setupSignalHandler(cancel)

	if err := runScanProcess(ctx); err != nil {
		fmt.Printf("❌ 扫描失败: %v\n", err)
	}
}

func resultHandler() {
	collection := mongoClient.Database("ollama_scan").Collection("results")

	for res := range resultsChan {
		printResult(res)
		writeCSV(res)

		// 将结果转换为 BSON 格式
		doc := bson.M{
			"ip":     res.IP,
			"models": res.Models,
		}

		// 插入到 MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := collection.InsertOne(ctx, doc)
		if err != nil {
			log.Printf("⚠️ 插入 MongoDB 失败: %v\n", err)
		}
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
	var cmd *exec.Cmd
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
			cmd = exec.Command("sudo", "-u", "root", "/usr/bin/apt-get", "update")
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
				cmd = exec.Command("sudo", "-u", "root", "/usr/bin/yum", "install", "-y", "zmap")
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
			return fmt.Errorf("brew is not installed, cannot install zmap automatically. Please install manually")
		}

		cmd = exec.Command("brew", "install", "zmap")
		installErr = cmd.Run()
		if installErr != nil {
			return fmt.Errorf("brew install zmap failed: %w", installErr)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s, cannot install zmap automatically. Please install manually", osName)
	}

	log.Println("zmap 安装完成")
	return nil
}


// initCSVWriter 函数用于初始化 CSV 写入器,创建 CSV 文件并写入表头.
func initCSVWriter() {
	var err error
	// 创建一个新的 CSV 文件,文件路径由命令行参数 -output 指定
	csvFile, err = os.Create(*outputFile)
	if err != nil {
		// 如果创建文件失败,打印错误信息
		fmt.Printf("⚠️ 创建CSV文件失败: %v\n", err)
		return
	}

	// 创建一个新的 CSV 写入器,用于将数据写入 CSV 文件
	csvWriter = csv.NewWriter(csvFile)
	// 定义 CSV 文件的表头
	headers := []string{"IP地址", "模型名称", "状态"}
	// 如果未禁用性能基准测试,则在表头中添加额外的列
	if !*disableBench {
		// 添加首Token延迟和Tokens/s列
		headers = append(headers, "首Token延迟(ms)", "Tokens/s")
	}
	// 将表头写入 CSV 文件
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
	if err := execZmap(); err != nil {
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

func execZmap() error {
    threads := *zmapThreads // 获取 zmap 线程数

	cmd := exec.Command("zmap",
		"-p", fmt.Sprintf("%d", port),
		"-G", *gatewayMAC,
		"-w", *inputFile,
		"-o", *outputFile,
		"-T", fmt.Sprintf("%d", threads)) // 设置 zmap 线程数
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

	// 启动 workers 来发现模型
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, ips)
	}

	// 启动 resultHandler 来处理扫描结果
	var rhWg sync.WaitGroup
	rhWg.Add(1)
	go func() {
		defer rhWg.Done()
		resultHandler()
	}()


	// 将 IP 地址发送到 channel
	go func() {
		defer close(ips)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			ip := strings.TrimSpace(scanner.Text())
			if net.ParseIP(ip) != nil {
				ips <- ip
			}
		}
	}()

	wg.Wait()
	close(resultsChan) // 关闭 resultsChan,通知 resultHandler

	rhWg.Wait() // 等待 resultHandler 处理完所有结果
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
			fmt.Printf("⚠️ 写入CSV失败: %v\n", err) // Handle the error appropriately
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
