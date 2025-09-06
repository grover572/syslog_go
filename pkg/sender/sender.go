package sender

import (
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
type Sender struct {
	config      *config.Config
	connPool    *ConnectionPool
	rateLimiter *RateLimiter
	stats       *Statistics
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	templateEngine *template.Engine
}

// Statistics 统计信息
type Statistics struct {
	Sent      int64     `json:"sent"`
	Failed    int64     `json:"failed"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	mutex     sync.RWMutex
}

// NewSender 创建新的发送器
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
func (s *Sender) sendMessage(msg *syslog.Message) error {
	conn, err := s.connPool.Get()
	if err != nil {
		if s.config.Verbose {
			fmt.Printf("获取连接失败: %v\n", err)
		}
		return fmt.Errorf("获取连接失败: %w", err)
	}
	defer s.connPool.Put(conn)

	// 发送消息
	data := msg.Bytes()
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("写入数据失败: %w", err)
	}

	return nil
}

// readFromDataFile 从数据文件读取内容
func (s *Sender) readFromDataFile() (string, error) {
	// 这里简化实现，实际应该支持多种文件格式和随机读取
	data, err := os.ReadFile(s.config.DataFile)
	if err != nil {
		if s.config.Verbose {
			fmt.Printf("读取数据文件失败: %v\n", err)
		}
		return "", err
	}
	return string(data), nil
}

// statsMonitor 统计监控
func (s *Sender) statsMonitor() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.printStats()
		}
	}
}

// printStats 打印统计信息
func (s *Sender) printStats() {
	if !s.config.Verbose {
		return
	}

	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	elapsed := time.Since(s.stats.StartTime)
	sent := atomic.LoadInt64(&s.stats.Sent)
	failed := atomic.LoadInt64(&s.stats.Failed)
	rate := float64(sent) / elapsed.Seconds()

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
func (s *Sender) Stop() {
	s.cancel()
	s.connPool.Close()
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
