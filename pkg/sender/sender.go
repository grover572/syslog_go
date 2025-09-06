package sender

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"syslog_go/pkg/config"
	"syslog_go/pkg/syslog"
	"syslog_go/pkg/template"
)

// Sender Syslog发送器
// 负责管理消息的生成、发送和统计信息收集
type Sender struct {
	config      *config.Config      // 配置信息
	connPool    *ConnectionPool     // 连接池，管理与目标服务器的连接
	rateLimiter *RateLimiter       // 速率限制器，控制消息发送速率
	stats       *Statistics        // 统计信息，记录发送状态和性能指标
	ctx         context.Context     // 上下文，用于控制发送器的生命周期
	cancel      context.CancelFunc  // 取消函数，用于停止发送器
	wg          sync.WaitGroup      // 等待组，用于优雅关闭
	templateEngine *template.Engine // 模板引擎，用于生成消息内容
	dataFile    *os.File           // 数据文件句柄
	dataScanner *bufio.Scanner     // 数据文件扫描器
}

// Statistics 统计信息
// 记录发送器的运行状态和性能指标
type Statistics struct {
	Sent      int64     `json:"sent"`      // 已成功发送的消息数量
	Failed    int64     `json:"failed"`    // 发送失败的消息数量
	StartTime time.Time `json:"start_time"` // 统计开始时间
	EndTime   time.Time `json:"end_time"`   // 统计结束时间
	mutex     sync.RWMutex                  // 读写锁，保护统计数据的并发访问
}

// NewSender 创建新的发送器实例
// 参数：
//   - cfg: 发送器配置信息，包含连接、模板、速率限制等配置
// 返回值：
//   - *Sender: 创建的发送器实例
//   - error: 创建过程中的错误，如果创建成功则为nil
func NewSender(cfg *config.Config) (*Sender, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)

	s := &Sender{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		stats:  &Statistics{StartTime: time.Now()},
	}

	// 初始化连接池
	if err := s.initConnectionPool(); err != nil {
		return nil, fmt.Errorf("初始化连接池失败: %w", err)
	}

	// 初始化速率限制器
	s.rateLimiter = NewRateLimiter(cfg.EPS)

	return s, nil
}

// initConnectionPool 初始化连接池
func (s *Sender) initConnectionPool() error {
	var err error
	s.connPool, err = NewConnectionPool(
		s.config.Target,
		s.config.Protocol,
		s.config.Concurrency,
		s.config.Timeout,
	)
	return err
}

// Start 开始发送
// 功能：
//   - 启动统计监控协程（如果启用）
//   - 启动多个发送工作协程
//   - 等待所有协程完成或超时
// 返回值：
//   - error: 启动过程中的错误，如果启动成功则为nil
func (s *Sender) Start() error {
	if s.config.Verbose {
		fmt.Printf("开始发送，目标: %s, 协议: %s, EPS: %d\n",
			s.config.Target, s.config.Protocol, s.config.EPS)
	}

	// 启动统计监控
	if s.config.EnableStats {
		s.wg.Add(1)
		go s.statsMonitor()
	}

	// 启动发送协程
	for i := 0; i < s.config.Concurrency; i++ {
		s.wg.Add(1)
		go s.sendWorker(i)
	}

	// 等待完成或超时
	s.wg.Wait()
	s.stats.EndTime = time.Now()

	// 打印最终统计
	s.printFinalStats()
	return nil
}

// sendWorker 发送工作协程
func (s *Sender) sendWorker(workerID int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 等待直到允许发送
			s.rateLimiter.Wait()

			// 生成消息
			message, err := s.generateMessage()
			if err != nil {
				if s.config.Verbose {
					fmt.Printf("生成消息失败: %v\n", err)
				}
				atomic.AddInt64(&s.stats.Failed, 1)
				continue
			}

			// 发送消息
			if s.config.Protocol == "udp" {
				_ = s.sendMessage(message)
				atomic.AddInt64(&s.stats.Sent, 1)
				if s.config.Verbose {
					fmt.Printf("发送消息: %s\n", message.Content)
				}
			} else if err = s.sendMessage(message); err != nil {
				atomic.AddInt64(&s.stats.Failed, 1)
				if s.config.Verbose {
					fmt.Printf("发送消息失败: %v\n", err)
				}
			} else {
				atomic.AddInt64(&s.stats.Sent, 1)
				if s.config.Verbose {
					fmt.Printf("成功发送消息: %s\n", message.Content)
				}
			}
		}
	}
}

// generateMessage 生成Syslog消息
// 功能：
//   - 根据配置生成消息内容
//   - 支持从命令行参数、模板文件或数据文件生成消息
//   - 自动处理消息格式和变量替换
// 返回值：
//   - *syslog.Message: 生成的Syslog消息对象
//   - error: 生成过程中的错误，如果生成成功则为nil
func (s *Sender) generateMessage() (*syslog.Message, error) {
	var content string
	var err error

	// 优先使用命令行指定的消息内容
	if s.config.Message != "" {
		// 使用共享的模板引擎
		if s.templateEngine == nil {
			// 检查当前目录下是否存在template.yml
			configPath := "template.yml"
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				configPath = "" // 如果文件不存在，使用空字符串
			}
			s.templateEngine = template.NewEngine(configPath, s.config.Verbose)
			s.templateEngine.LoadTemplate("message", s.config.Message)
		}
		
		// 处理消息中的变量
		content, err = s.templateEngine.GenerateMessage("message")
		if err != nil {
			return nil, fmt.Errorf("处理消息变量失败: %w", err)
		}
	} else if s.config.DataFile != "" {
		// 如果有数据文件，从文件读取
		content, err = s.readFromDataFile()
		if err != nil {
			return nil, err
		}
	} else {
		// 使用默认消息
		content = fmt.Sprintf("Test message from syslog_go by saturn at %s", time.Now().Format(time.RFC3339))
	}

	// 获取主机名
	hostname := "localhost"
	if h, err := os.Hostname(); err == nil {
		hostname = h
	}

	// 创建Syslog消息
	msg := syslog.NewMessage(
		s.config.GetPriority(),
		hostname,
		"syslog_go",
		content,
		syslog.ParseFormat(s.config.Format),
	)

	return msg, nil
}

// sendMessage 发送消息
// 功能：
//   - 从连接池获取连接
//   - 将消息序列化并发送
//   - 处理发送过程中的错误
// 参数：
//   - msg: 要发送的Syslog消息对象
// 返回值：
//   - error: 发送过程中的错误，如果发送成功则为nil
func (s *Sender) sendMessage(msg *syslog.Message) error {
	// 从连接池获取连接
	conn, err := s.connPool.Get()
	if err != nil {
		if s.config.Verbose {
			fmt.Printf("获取连接失败: %v\n", err)
		}
		return fmt.Errorf("获取连接失败: %w", err)
	}
	defer s.connPool.Put(conn)

	// 序列化并发送消息
	data := msg.Bytes()
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("写入数据失败: %w", err)
	}

	return nil
}

// readFromDataFile 从数据文件读取内容
// 功能：
//   - 按行读取数据文件
//   - 维护当前读取位置，支持循环读取
//   - 返回下一行数据
// 返回值：
//   - string: 读取的行内容
//   - error: 读取过程中的错误
func (s *Sender) readFromDataFile() (string, error) {
	// 如果文件未打开，则打开文件
	if s.dataFile == nil {
		file, err := os.Open(s.config.DataFile)
		if err != nil {
			if s.config.Verbose {
				fmt.Printf("打开数据文件失败: %v\n", err)
			}
			return "", fmt.Errorf("打开数据文件失败: %w", err)
		}
		s.dataFile = file
		s.dataScanner = bufio.NewScanner(file)
	}

	// 如果已到文件末尾，重新开始读取
	if !s.dataScanner.Scan() {
		if err := s.dataScanner.Err(); err != nil {
			return "", fmt.Errorf("读取数据文件失败: %w", err)
		}
		// 重置文件指针到开头
		if _, err := s.dataFile.Seek(0, 0); err != nil {
			return "", fmt.Errorf("重置文件指针失败: %w", err)
		}
		s.dataScanner = bufio.NewScanner(s.dataFile)
		if !s.dataScanner.Scan() {
			return "", fmt.Errorf("数据文件为空")
		}
	}

	return s.dataScanner.Text(), nil
}

// statsMonitor 统计监控协程
// 功能：
//   - 定期收集和输出发送统计信息
//   - 监控发送性能和错误情况
//   - 在收到停止信号时优雅退出
func (s *Sender) statsMonitor() {
	defer s.wg.Done()

	// 创建定时器，间隔由配置指定
	ticker := time.NewTicker(s.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// 收到停止信号，退出协程
			return
		case <-ticker.C:
			// 定时输出统计信息
			s.printStats()
		}
	}
}

// printStats 打印当前的发送统计信息
// 功能：
//   - 计算并展示实时发送速率
//   - 输出成功、失败、运行时间等统计数据
//   - 仅在verbose模式下输出详细信息
func (s *Sender) printStats() {
	if !s.config.Verbose {
		return
	}

	// 使用读锁保护并发访问
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	// 计算统计指标
	elapsed := time.Since(s.stats.StartTime)
	sent := atomic.LoadInt64(&s.stats.Sent)
	failed := atomic.LoadInt64(&s.stats.Failed)
	rate := float64(sent) / elapsed.Seconds()

	// 格式化输出统计信息
	fmt.Printf("[统计] 已发送: %d, 失败: %d, 速率: %.2f/s, 运行时间: %v\n",
		sent, failed, rate, elapsed.Truncate(time.Second))
}

// printFinalStats 打印最终统计
func (s *Sender) printFinalStats() {
	if !s.config.Verbose {
		return
	}

	elapsed := s.stats.EndTime.Sub(s.stats.StartTime)
	sent := atomic.LoadInt64(&s.stats.Sent)
	failed := atomic.LoadInt64(&s.stats.Failed)
	rate := float64(sent) / elapsed.Seconds()

	fmt.Printf("\n=== 发送完成 ===\n")
	fmt.Printf("总发送数: %d\n", sent)
	fmt.Printf("失败数: %d\n", failed)
	fmt.Printf("成功率: %.2f%%\n", float64(sent)/float64(sent+failed)*100)
	fmt.Printf("平均速率: %.2f/s\n", rate)
	fmt.Printf("总耗时: %v\n", elapsed.Truncate(time.Millisecond))
}

// Stop 停止发送
// 功能：
//   - 通过context取消信号停止所有工作协程
//   - 关闭连接池释放资源
//   - 关闭数据文件
//   - 确保资源完全释放和协程优雅退出
func (s *Sender) Stop() {
	s.cancel()
	s.connPool.Close()
	// 关闭数据文件
	if s.dataFile != nil {
		s.dataFile.Close()
		s.dataFile = nil
		s.dataScanner = nil
	}
}

// GetStats 获取统计信息
func (s *Sender) GetStats() *Statistics {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	return &Statistics{
		Sent:      atomic.LoadInt64(&s.stats.Sent),
		Failed:    atomic.LoadInt64(&s.stats.Failed),
		StartTime: s.stats.StartTime,
		EndTime:   s.stats.EndTime,
	}
}
