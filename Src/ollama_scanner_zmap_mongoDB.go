// v2.2.1 增加断点续扫功能 支持进度条显示
// 自动获取 eth0 网卡的 MAC 地址
// 在以下情况下尝试自动获取 MAC 地址：
// 配置文件不存在时
// 配置文件中的 MAC 地址为空时
// 命令行参数未指定 MAC 地址时
// 获取失败时给出相应的错误提示
// 保存配置文件时更新 MAC 地址
// 保存配置文件时更新 zmap 线程数
// 保存配置文件时更新输入文件路径
// 保存配置文件时更新输出文件路径
// 保存配置文件时更新端口号
// 保存配置文件时更新禁用性能基准测试选项
// 增加mongoDB驱动,并实现插入数据功能

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
	"net"
	"go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"
)
// 在 const 声明之前添加配置结构体
type Config struct {
    Port       int    `json:"port"`
    GatewayMAC string `json:"gateway_mac"`
    InputFile  string `json:"input_file"`
    OutputFile string `json:"output_file"`
    ZmapThreads int   `json:"zmap_threads"`
}

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
	//mongoDB变量
	mongoClient *mongo.Client
    mongoURI    = flag.String("mongo-uri", "mongodb://localhost:27017", "MongoDB 连接URI")
    
// 移除 useDB 标志，因为现在是强制性的

)

// 修改 loadConfig 函数
func loadConfig() error {
    data, err := os.ReadFile(".env.json")
    if err != nil {
        if os.IsNotExist(err) {
            // 如果配置文件不存在,尝试获取 eth0 的 MAC 地址
            mac, err := getEth0MAC()
            if err != nil {
                log.Printf("自动获取MAC地址失败: %v", err)
            }
            
            // 创建默认配置
            config = Config{
                Port:       11434,
                GatewayMAC: mac, // 使用获取到的 MAC 地址
                InputFile:  "ip.txt",
                OutputFile: defaultCSVFile,
                ZmapThreads: defaultZmapThreads,
            }
            // 保存默认配置
            return saveConfig()
        }
        return fmt.Errorf("读取配置文件失败: %w", err)
    }

    if err := json.Unmarshal(data, &config); err != nil {
        return fmt.Errorf("解析配置文件失败: %w", err)
    }

    // 如果配置中的 GatewayMAC 为空,尝试获取 eth0 的 MAC 地址
    if config.GatewayMAC == "" {
        mac, err := getEth0MAC()
        if err != nil {
            log.Printf("自动获取MAC地址失败: %v", err)
        } else {
            config.GatewayMAC = mac
            // 保存更新后的配置
            if err := saveConfig(); err != nil {
                log.Printf("保存更新后的配置失败: %v", err)
            }
        }
    }

    // 使用配置更新相关变量
    port = config.Port
    *gatewayMAC = config.GatewayMAC
    *inputFile = config.InputFile
    *outputFile = config.OutputFile
    *zmapThreads = config.ZmapThreads

    return nil
}

func saveConfig() error {
    // 更新配置对象
    config.Port = port
    config.GatewayMAC = *gatewayMAC
    config.InputFile = *inputFile
    config.OutputFile = *outputFile
    config.ZmapThreads = *zmapThreads

    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("序列化配置失败: %w", err)
    }

    if err := os.WriteFile(".env", data, 0644); err != nil {
        return fmt.Errorf("保存配置文件失败: %w", err)
    }

    return nil
}

type ScanResult struct {
    IP     string      `json:"ip" bson:"ip"`
    Models []ModelInfo `json:"models" bson:"models"`
}

type ModelInfo struct {
    Name           string        `json:"name" bson:"name"`
    FirstTokenDelay time.Duration `json:"first_token_delay" bson:"first_token_delay"`
    TokensPerSec   float64       `json:"tokens_per_sec" bson:"tokens_per_sec"`
    Status         string        `json:"status" bson:"status"`
}

func init() {
    // 命令行参数仍然保留,但作为覆盖配置文件的选项
    gatewayMAC = flag.String("gateway-mac", "", "指定网关MAC地址(格式:aa:bb:cc:dd:ee:ff)")
    inputFile = flag.String("input", "ip.txt", "输入文件路径(CIDR格式列表)")
    outputFile = flag.String("output", defaultCSVFile, "CSV输出文件路径")
    disableBench = flag.Bool("no-bench", false, "禁用性能基准测试")
    benchPrompt = flag.String("prompt", "为什么太阳会发光？用一句话回答", "性能测试提示词")
    zmapThreads = flag.Int("T", defaultZmapThreads, "zmap 线程数 (默认为 10)")

    flag.Usage = func() {
		helpText := fmt.Sprintf(`Ollama节点扫描工具 v2.2.1 https://t.me/Ollama_Scanner
		默认功能:
		- 自动执行性能测试
		- 结果导出到%s和MongoDB
		
		使用方法:
		%s [参数]
		
		MongoDB配置:
		  -mongo-uri    MongoDB连接URI (默认: mongodb://localhost:27017)
		必须配置有效的MongoDB连接才能运行程序
		
		参数说明:
		`, defaultCSVFile, os.Args[0])

        fmt.Fprintf(os.Stderr, helpText)
        flag.PrintDefaults()

        examples := fmt.Sprintf(`
基础使用示例:
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -T 20

MongoDB支持示例:
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -use-db
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -use-db -mongo-uri mongodb://user:pass@host:port
`, os.Args[0])

        fmt.Fprintf(os.Stderr, examples)
    }

    // 加载配置文件
    if err := loadConfig(); err != nil {
        log.Printf("加载配置文件失败: %v, 使用默认配置", err)
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

type Progress struct {
    mu sync.Mutex
    total int
    current int
    startTime time.Time
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
    fmt.Printf("\r进度: %.1f%% (%d/%d) 已用时间: %v 预计剩余: %v", 
        percentage, p.current, p.total, elapsed.Round(time.Second), remainingTime.Round(time.Second))
}

// 增加断点续扫功能
const (
    // ...existing code...
    stateFile = "scan_state.json"  // 状态文件名
)

var (
    // ...existing code...
    resumeScan = flag.Bool("resume", false, "从上次中断处继续扫描")
)

// ScanState 结构体用于保存扫描状态
type ScanState struct {
    ScannedIPs  map[string]bool    `json:"scanned_ips"`
    LastScanTime time.Time         `json:"last_scan_time"`
    TotalIPs     int              `json:"total_ips"`
    Config       ScanConfig        `json:"config"`
}

type ScanConfig struct {
    GatewayMAC  string `json:"gateway_mac"`
    InputFile   string `json:"input_file"`
    OutputFile  string `json:"output_file"`
    DisableBench bool  `json:"disable_bench"`
}

// saveState 函数用于保存扫描状态到文件中
func saveState(state *ScanState) error {
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("序列化状态失败: %w", err)
    }

    if err := os.WriteFile(stateFile, data, 0644); err != nil {
        return fmt.Errorf("保存状态文件失败: %w", err)
    }

    return nil
}

func loadState() (*ScanState, error) {
    data, err := os.ReadFile(stateFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, fmt.Errorf("读取状态文件失败: %w", err)
    }

    var state ScanState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, fmt.Errorf("解析状态文件失败: %w", err)
    }

    return &state, nil
}

func validateStateConfig(state *ScanState) bool {
    return state.Config.GatewayMAC == *gatewayMAC &&
           state.Config.InputFile == *inputFile &&
           state.Config.OutputFile == *outputFile &&
           state.Config.DisableBench == *disableBench
}

// main 函数是程序的入口点,负责初始化程序、检查并安装 zmap、设置信号处理和启动扫描过程.
// 在 main 函数中修改 MongoDB 初始化逻辑
func main() {
    flag.Parse()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // MongoDB 强制性初始化
    clientOptions := options.Client().ApplyURI(*mongoURI)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatalf("❌ MongoDB连接失败: %v\n必须配置有效的MongoDB连接才能继续运行", err)
    }
    mongoClient = client
    defer mongoClient.Disconnect(ctx)

    // 测试连接
    err = mongoClient.Ping(ctx, nil)
    if err != nil {
        log.Fatalf("❌ MongoDB连接测试失败: %v\n必须配置有效的MongoDB连接才能继续运行", err)
    }
    log.Printf("✅ MongoDB连接成功: %s", *mongoURI)


	// 检查并安装 zmap,如果未安装则尝试自动安装
	// Check and install zmap if it's not already installed
	if err := checkAndInstallZmap(); err != nil {
		// 打印无法安装 zmap 的错误信息
		log.Printf("❌ 无法安装 zmap: %v\n 请手动安装 zmap 后重试\n", err)
		// 提示用户手动安装 zmap 的链接
        fmt.Printf("请确认已安装 zmap,或手动安装后重试 (https://github.com/zmap/zmap)\n")
		// 询问用户是否跳过自动安装 zmap 并继续执行程序
		fmt.Printf("是否跳过自动安装 zmap 并继续执行程序？ (y/n): ")
		var answer string
		// 读取用户输入
		fmt.Scanln(&answer)
		// 如果用户输入不是 'y',则退出程序
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
        // 保存配置
        if err := saveConfig(); err != nil {
            log.Printf("保存配置失败: %v", err)
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
    // 如果命令行参数中未指定 MAC 地址,尝试获取 eth0 的 MAC 地址
    if *gatewayMAC == "" {
        mac, err := getEth0MAC()
        if err != nil {
            return fmt.Errorf("必须指定网关MAC地址,自动获取失败: %v", err)
        }
        *gatewayMAC = mac
        log.Printf("自动使用 eth0 网卡 MAC 地址: %s", mac)
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

// v2.2.1支持断点续扫功能
func processResults(ctx context.Context) error {
    file, err := os.Open(*outputFile)
    if err != nil {
        return fmt.Errorf("打开结果文件失败: %w", err)
    }
    defer file.Close()

    // 加载之前的扫描状态
    var state *ScanState
    if *resumeScan {
        state, err = loadState()
        if err != nil {
            return fmt.Errorf("加载扫描状态失败: %w", err)
        }
        if state != nil && !validateStateConfig(state) {
            return fmt.Errorf("扫描配置已更改,无法继续之前的扫描")
        }
    }

    if state == nil {
        state = &ScanState{
            ScannedIPs: make(map[string]bool),
            Config: ScanConfig{
                GatewayMAC:   *gatewayMAC,
                InputFile:    *inputFile,
                OutputFile:   *outputFile,
                DisableBench: *disableBench,
            },
        }
    }

    // 计算总IP数并更新进度
    scanner := bufio.NewScanner(file)
    if state.TotalIPs == 0 {
        for scanner.Scan() {
            if net.ParseIP(strings.TrimSpace(scanner.Text())) != nil {
                state.TotalIPs++
            }
        }
        file.Seek(0, 0)
    }

    progress := &Progress{}
    progress.Init(state.TotalIPs)
    progress.current = len(state.ScannedIPs)

    ips := make(chan string, maxWorkers*2)
    var wg sync.WaitGroup

    // 定期保存扫描状态
    stopSaving := make(chan struct{})
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                state.LastScanTime = time.Now()
                if err := saveState(state); err != nil {
                    log.Printf("保存扫描状态失败: %v", err)
                }
            case <-stopSaving:
                return
            }
        }
    }()

    // 修改 worker 函数以支持断点续扫
    workerWithProgress := func(ctx context.Context, wg *sync.WaitGroup, ips <-chan string) {
        defer wg.Done()
        for ip := range ips {
            select {
            case <-ctx.Done():
                return
            default:
                if state.ScannedIPs[ip] {
                    progress.Increment()
                    continue
                }

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
                state.ScannedIPs[ip] = true
                progress.Increment()
            }
        }
    }

    // ...existing worker startup code...

    wg.Wait()
    close(resultsChan)
    rhWg.Wait()
    csvWriter.Flush()

    // 保存最终状态
    close(stopSaving)
    state.LastScanTime = time.Now()
    if err := saveState(state); err != nil {
        log.Printf("保存最终扫描状态失败: %v", err)
    }

    fmt.Printf("\n✅ 扫描完成,结果已保存至 %s\n", *outputFile)
    return nil
}




func resultHandler() {
    collection := mongoClient.Database("ollama_scan").Collection("results")

    for res := range resultsChan {
        printResult(res)
        writeCSV(res)

     // MongoDB存储（现在是强制性的）
		doc := bson.M{
			"ip":           res.IP,
			"models":       res.Models,
			"scan_time":    time.Now(),
			"gateway_mac":  *gatewayMAC,
			"bench_status": !*disableBench,
			"scan_config": bson.M{
				"input_file":  *inputFile,
				"output_file": *outputFile,
				"threads":     *zmapThreads,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := collection.InsertOne(ctx, doc)
		cancel()

		if err != nil {
			log.Printf("❌ MongoDB存储失败 [%s]: %v", res.IP, err)
			// 如果存储失败，终止程序
			log.Fatalf("MongoDB存储失败，程序终止")
		}
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
    d := net.Dialer{Timeout: timeout}
    conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
    if err != nil {
        return false
    }
    conn.Close()
    return true
}

func checkOllama(ip string) bool {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", 
        fmt.Sprintf("http://%s:%d", ip, port), nil)
    if err != nil {
        return false
    }

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
