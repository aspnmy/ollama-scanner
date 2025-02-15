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
	"go.mongodb.org/mongo-driver/mongo/readpref"
)
// 在 const 声明之前添加配置结构体
type Config struct {
    Port         int    `json:"port"`
    GatewayMAC   string `json:"gateway_mac"`
    InputFile    string `json:"input_file"`
    OutputFile   string `json:"output_file"`
    ZmapThreads  int    `json:"zmap_threads"`
    MasscanRate  int    `json:"masscan_rate"`
    DisableBench bool   `json:"disable_bench"`
    BenchPrompt  string `json:"bench_prompt"`
    MongoDBURI   string `json:"mongodb_uri"`
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
    defaultMasscanRate = 1000 // masscan 默认扫描速率
    defaultBenchPrompt = "为什么太阳会发光？用一句话回答"
)

var (
    gatewayMAC  = flag.String("gateway-mac", "", "指定网关MAC地址(格式:aa:bb:cc:dd:ee:ff)")
    inputFile   = flag.String("input", "ip.txt", "输入文件路径(CIDR格式列表)")
    outputFile  = flag.String("output", defaultCSVFile, "CSV输出文件路径")
    disableBench = flag.Bool("no-bench", false, "禁用性能基准测试")
    benchPrompt = flag.String("prompt", defaultBenchPrompt, "性能测试提示词")
    httpClient  *http.Client
    csvWriter   *csv.Writer
    csvFile     *os.File
    zmapThreads *int    // zmap 线程数
    resultsChan chan ScanResult
    allResults  []ScanResult
    mu          sync.Mutex
    scannerType string  // 扫描器类型 (zmap/masscan)
    masscanRate = flag.Int("rate", defaultMasscanRate, "masscan 扫描速率 (每秒扫描的包数)")
    config      Config
    mongoClient *mongo.Client
    useMongoDB  bool
)

// 选择合适的扫描器并初始化
func initScanner() error {
    osName := runtime.GOOS
    if osName == "windows" {
        scannerType = "masscan"
        log.Printf("Windows 系统，使用 masscan 扫描器")
    } else {
        scannerType = "zmap"
        log.Printf("Unix/Linux 系统，使用 zmap 扫描器")
    }

    return checkAndInstallScanner()
}

// 检查并安装扫描器
func checkAndInstallScanner() error {
    if scannerType == "masscan" {
        return checkAndInstallMasscan()
    }
    return checkAndInstallZmap()
}

// 添加 masscan 安装函数
func checkAndInstallMasscan() error {
    _, err := exec.LookPath("masscan")
    if err == nil {
        log.Println("masscan 已安装")
        return nil
    }

    log.Println("masscan 未安装, 尝试自动安装...")
    osName := runtime.GOOS

    switch osName {
    case "linux":
        // 尝试使用 apt
        if err := exec.Command("apt", "-v").Run(); err == nil {
            cmd := exec.Command("sudo", "apt-get", "update")
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("apt-get update 失败: %w", err)
            }
            cmd = exec.Command("sudo", "apt-get", "install", "-y", "masscan")
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("安装 masscan 失败: %w", err)
            }
        } else {
            // 尝试使用 yum
            if err := exec.Command("yum", "-v").Run(); err == nil {
                cmd := exec.Command("sudo", "yum", "install", "-y", "masscan")
                if err := cmd.Run(); err != nil {
                    return fmt.Errorf("安装 masscan 失败: %w", err)
                }
            } else {
                return fmt.Errorf("无法找到包管理器")
            }
        }
    default:
        return fmt.Errorf("不支持在 %s 系统上自动安装 masscan", osName)
    }

    log.Println("masscan 安装完成")
    return nil
}

// 修改 loadConfig 函数
func loadConfig() error {
    data, err := os.ReadFile(".env.json")
    if err != nil {
        if (os.IsNotExist(err)) {
            // 如果配置文件不存在,尝试获取 eth0 的 MAC 地址
            mac, err := getEth0MAC()
            if err != nil {
                log.Printf("自动获取MAC地址失败: %v", err)
            }
            
            // 创建默认配置
            config = Config{
                Port:         port,
                GatewayMAC:   mac, // 使用获取到的 MAC 地址
                InputFile:    "ip.txt",
                OutputFile:   defaultCSVFile,
                ZmapThreads:  defaultZmapThreads,
                MasscanRate:  defaultMasscanRate,
                DisableBench: false,
                BenchPrompt:  defaultBenchPrompt,
                MongoDBURI:   "",
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
    *masscanRate = config.MasscanRate
    *disableBench = config.DisableBench
    *benchPrompt = config.BenchPrompt

    return nil
}

func saveConfig() error {
    // 更新配置对象
    config.Port = port
    config.GatewayMAC = *gatewayMAC
    config.InputFile = *inputFile
    config.OutputFile = *outputFile
    config.ZmapThreads = *zmapThreads
    config.MasscanRate = *masscanRate
    config.DisableBench = *disableBench
    config.BenchPrompt = *benchPrompt

    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return fmt.Errorf("序列化配置失败: %w", err)
    }

    if err := os.WriteFile(".env.json", data, 0644); err != nil {
        return fmt.Errorf("保存配置文件失败: %w", err)
    }

    return nil
}

type ScanResult struct {
	IP     string
	Models []ModelInfo
}

type ModelInfo struct {
	Name          string
	FirstTokenDelay time.Duration
	TokensPerSec  float64
	Status        string
}

func init() {
    // 命令行参数仍然保留,但作为覆盖配置文件的选项
    gatewayMAC = flag.String("gateway-mac", "", "指定网关MAC地址(格式:aa:bb:cc:dd:ee:ff)")
    inputFile = flag.String("input", "ip.txt", "输入文件路径(CIDR格式列表)")
    outputFile = flag.String("output", defaultCSVFile, "CSV输出文件路径")
    disableBench = flag.Bool("no-bench", false, "禁用性能基准测试")
    benchPrompt = flag.String("prompt", defaultBenchPrompt, "性能测试提示词")
    zmapThreads = flag.Int("T", defaultZmapThreads, "zmap 线程数 (默认为 10)")
    masscanRate = flag.Int("rate", defaultMasscanRate, "masscan 扫描速率 (每秒扫描的包数)")

    flag.Usage = func() {
        helpText := fmt.Sprintf(`Ollama节点扫描工具 v2.2.1 https://t.me/+YfCVhGWyKxoyMDhl
默认功能:
- 自动执行性能测试
- 结果导出到%s
- Windows系统使用masscan，其他系统使用zmap

使用方法:
%s [参数]

参数说明:
`, defaultCSVFile, os.Args[0])

        fmt.Fprintf(os.Stderr, helpText)
        flag.PrintDefaults()

        examples := fmt.Sprintf(`
基础使用示例:
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -no-bench -output custom.csv
  
Zmap参数 (Unix/Linux):
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -T 20

Masscan参数 (Windows):
  %[1]s -gateway-mac aa:bb:cc:dd:ee:ff -rate 2000
`, os.Args[0])

        fmt.Fprintf(os.Stderr, examples)
    }

    // 加载配置文件
    if err := loadConfig(); err != nil {
        log.Printf("加载配置文件失败: %v, 使用默认配置", err)
    }

    // 初始化 MongoDB 客户端
    if config.MongoDBURI != "" {
        // 检查并安装 MongoDB
        if err := checkAndInstallMongoDB(); err != nil {
            log.Printf("MongoDB 安装失败: %v, 使用 CSV 模式", err)
        } else {
            clientOptions := options.Client().ApplyURI(config.MongoDBURI)
            client, err := mongo.Connect(context.TODO(), clientOptions)
            if err == nil {
                err = client.Ping(context.TODO(), readpref.Primary())
                if err == nil {
                    mongoClient = client
                    useMongoDB = true
                    log.Println("成功连接到 MongoDB")
                } else {
                    log.Printf("无法连接到 MongoDB: %v, 使用 CSV 模式", err)
                }
            } else {
                log.Printf("无法连接到 MongoDB: %v, 使用 CSV 模式", err)
            }
        }
    } else {
        log.Println("未配置 MongoDB URI, 使用 CSV 模式")
    }

    httpClient = &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:    maxIdleConns,
            MaxIdleConnsPerHost: maxIdleConns,
            IdleConnTimeout: idleConnTimeout,
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
        return fmt.Errorf("解析状态文件失败: %w", err)
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
func main() {
    // 解析命令行参数
    flag.Parse()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 初始化扫描器
    if err := initScanner(); err != nil {
        log.Printf("❌ 初始化扫描器失败: %v\n", err)
        fmt.Printf("是否继续执行程序？(y/n): ")
        var answer string
        fmt.Scanln(&answer)
        if strings.ToLower(answer) != "y" {
            os.Exit(1)
        }
    }

    // 初始化 CSV 写入器或 MongoDB
    if useMongoDB {
        log.Println("使用 MongoDB 保存结果")
    } else {
        initCSVWriter()
        defer csvFile.Close()
    }
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
	if (!*disableBench) {
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
    if err := execScan(); err != nil {
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
func execScan() error {
    if scannerType == "masscan" {
        return execMasscan()
    }
    return execZmap()
}

func execMasscan() error {
    cmd := exec.Command("masscan",
        "-p", fmt.Sprintf("%d", port),
        "--rate", fmt.Sprintf("%d", *masscanRate),
        "--interface", "eth0",
        "--source-ip", *gatewayMAC,
        "-iL", *inputFile,
        "-oL", *outputFile)
    
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
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
    for res := range resultsChan {
        printResult(res)
        if useMongoDB {
            writeMongoDB(res)
        } else {
            writeCSV(res)
        }
    }
}

func writeMongoDB(res ScanResult) {
    collection := mongoClient.Database("ollama_scanner").Collection("scan_results")
    _, err := collection.InsertOne(context.TODO(), res)
    if err != nil {
        log.Printf("⚠️ 写入MongoDB失败: %v\n", err)
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

// 添加 MongoDB 安装检查函数
func checkAndInstallMongoDB() error {
    // 检查 mongod 是否已安装
    _, err := exec.LookPath("mongod")
    if err == nil {
        log.Println("MongoDB 已安装")
        return nil
    }

    log.Println("MongoDB 未安装, 尝试自动安装...")
    osName := runtime.GOOS

    switch osName {
    case "linux":
        // Debian/Ubuntu
        if err := exec.Command("apt", "-v").Run(); err == nil {
            // 添加 MongoDB 源
            cmd := exec.Command("sudo", "bash", "-c", `
                curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | \
                sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor && \
                echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | \
                sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
            `)
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("添加 MongoDB 源失败: %w", err)
            }

            cmd = exec.Command("sudo", "apt-get", "update")
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("apt-get update 失败: %w", err)
            }

            cmd = exec.Command("sudo", "apt-get", "install", "-y", "mongodb-org")
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("安装 MongoDB 失败: %w", err)
            }

            // 启动 MongoDB 服务
            cmd = exec.Command("sudo", "systemctl", "start", "mongod")
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("启动 MongoDB 服务失败: %w", err)
            }

        } else {
            // CentOS/RHEL
            if err := exec.Command("yum", "-v").Run(); err == nil {
                // 添加 MongoDB 源
                cmd := exec.Command("sudo", "bash", "-c", `
                    echo '[mongodb-org-7.0]
                    name=MongoDB Repository
                    baseurl=https://repo.mongodb.org/yum/redhat/$releasever/mongodb-org/7.0/x86_64/
                    gpgcheck=1
                    enabled=1
                    gpgkey=https://www.mongodb.org/static/pgp/server-7.0.asc' | \
                    sudo tee /etc/yum.repos.d/mongodb-org-7.0.repo
                `)
                if err := cmd.Run(); err != nil {
                    return fmt.Errorf("添加 MongoDB 源失败: %w", err)
                }

                cmd = exec.Command("sudo", "yum", "install", "-y", "mongodb-org")
                if err := cmd.Run(); err != nil {
                    return fmt.Errorf("安装 MongoDB 失败: %w", err)
                }

                // 启动 MongoDB 服务
                cmd = exec.Command("sudo", "systemctl", "start", "mongod")
                if err := cmd.Run(); err != nil {
                    return fmt.Errorf("启动 MongoDB 服务失败: %w", err)
                }
            } else {
                return fmt.Errorf("无法找到包管理器")
            }
        }
    case "darwin":
        // macOS
        _, brewErr := exec.LookPath("brew")
        if brewErr != nil {
            return fmt.Errorf("未安装 Homebrew，无法自动安装 MongoDB")
        }

        cmd := exec.Command("brew", "tap", "mongodb/brew")
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("添加 MongoDB tap 失败: %w", err)
        }

        cmd = exec.Command("brew", "install", "mongodb-community")
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("安装 MongoDB 失败: %w", err)
        }

        cmd = exec.Command("brew", "services", "start", "mongodb-community")
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("启动 MongoDB 服务失败: %w", err)
        }
    default:
        return fmt.Errorf("不支持在 %s 系统上自动安装 MongoDB", osName)
    }

    log.Println("MongoDB 安装完成")
    return nil
}
