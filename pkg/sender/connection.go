package sender

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// ConnectionPool 连接池结构体
// 用于管理和复用与目标服务器的网络连接
// 主要功能：
// 1. 连接管理：预创建和复用连接，减少连接建立开销
// 2. 并发控制：支持多协程安全地获取和归还连接
// 3. 故障处理：自动检测和重建失效连接
// 4. 资源控制：限制最大连接数，防止资源耗尽
// 5. 源地址模拟：支持指定源IP地址（需要root权限）
type ConnectionPool struct {
	// 基础配置
	address  string        // 目标服务器地址，格式：host:port
	protocol string        // 网络协议，支持tcp和udp
	maxSize  int           // 连接池最大容量
	timeout  time.Duration // 连接超时时间

	// 连接管理
	connections chan net.Conn // 连接通道，用于存储和分发连接
	mutex       sync.RWMutex  // 读写锁，保护并发访问
	closed      bool          // 连接池状态标志

	// 高级功能
	sourceIP string // 源IP地址，用于IP伪装，为空则使用系统默认地址
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(address, protocol string, maxSize int, timeout time.Duration, sourceIP string) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		address:     address,
		protocol:    protocol,
		maxSize:     maxSize,
		timeout:     timeout,
		connections: make(chan net.Conn, maxSize),
		sourceIP:    sourceIP,
	}

	// 预创建连接
	for i := 0; i < maxSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// 如果无法创建连接，关闭已创建的连接
			pool.Close()
			return nil, fmt.Errorf("创建连接失败: %w", err)
		}
		pool.connections <- conn
	}

	return pool, nil
}

// createConnection 创建新连接
// 支持IPv4和IPv6地址格式，支持原始套接字模拟源IP地址
func (p *ConnectionPool) createConnection() (net.Conn, error) {
	// 构建网络地址
	network := p.protocol
	if network == "tcp" || network == "udp" {
		// 检查是否为IPv6地址
		if strings.Contains(p.address, ":") {
			// 如果地址中包含多个冒号，说明是IPv6地址
			// 检查地址是否已包含端口号
			if !strings.HasSuffix(p.address, "]") {
				// 如果地址不是以]结尾，说明需要添加端口号
				// 查找最后一个冒号，它应该是端口号分隔符
				lastColon := strings.LastIndex(p.address, ":")
				if lastColon != -1 {
					// 分离地址和端口
					host := p.address[:lastColon]
					port := p.address[lastColon+1:]
					// 重新组合地址，确保IPv6地址被方括号包围
					if !strings.HasPrefix(host, "[") {
						host = "[" + host + "]"
					}
					p.address = host + ":" + port
				}
			}
		}

		// 如果指定了源IP地址且不是本机IP，尝试使用原始套接字
		if p.sourceIP != "" && !isLocalIP(p.sourceIP) {
			fmt.Printf("尝试使用原始套接字模拟源IP地址: %s\n", p.sourceIP)
			// 尝试创建原始套接字连接
			rawConn, err := newRawSocketConn(p.sourceIP, p.address, network, true) // 启用详细日志
			if err != nil {
				fmt.Printf("警告: 创建原始套接字失败: %v\n", err)
				fmt.Printf("回退到标准连接，使用系统默认地址\n")
				// 回退到标准连接，不设置源IP
				return (&net.Dialer{Timeout: p.timeout}).Dial(network, p.address)
			}
			return rawConn, nil
		}

		// 使用Dialer以支持设置源IP地址
		dialer := &net.Dialer{
			Timeout: p.timeout,
		}

		// 如果指定了源IP地址且为本机IP，设置本地地址
		if p.sourceIP != "" && isLocalIP(p.sourceIP) {
			var localAddr net.Addr
			if network == "tcp" {
				localAddr, _ = net.ResolveTCPAddr(network, p.sourceIP+":0")
			} else if network == "udp" {
				localAddr, _ = net.ResolveUDPAddr(network, p.sourceIP+":0")
			}
			if localAddr != nil {
				dialer.LocalAddr = localAddr
			}
		}

		return dialer.Dial(network, p.address)
	}
	return nil, fmt.Errorf("不支持的协议: %s", p.protocol)
}

// Get 从连接池获取连接
func (p *ConnectionPool) Get() (net.Conn, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("连接池已关闭")
	}

	select {
	case conn := <-p.connections:
		// 检查连接是否有效
		if p.isConnectionValid(conn) {
			return conn, nil
		}
		// 连接无效，创建新连接
		conn.Close()
		return p.createConnection()
	default:
		// 连接池为空，创建新连接
		return p.createConnection()
	}
}

// Put 将连接放回连接池
func (p *ConnectionPool) Put(conn net.Conn) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.closed || !p.isConnectionValid(conn) {
		conn.Close()
		return
	}

	select {
	case p.connections <- conn:
		// 成功放回连接池
	default:
		// 连接池已满，关闭连接
		conn.Close()
	}
}

// isConnectionValid 检查连接是否有效
func (p *ConnectionPool) isConnectionValid(conn net.Conn) bool {
	if conn == nil {
		return false
	}

	// 对于UDP连接，总是认为有效
	if p.protocol == "udp" {
		return true
	}

	// 对于TCP连接，尝试设置读取超时来检查连接状态
	conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	conn.SetReadDeadline(time.Time{}) // 清除超时

	// 如果是超时错误，说明连接正常但没有数据
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// 其他错误说明连接有问题
	return err == nil
}

// Close 关闭连接池
func (p *ConnectionPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.connections)

	// 关闭所有连接
	for conn := range p.connections {
		conn.Close()
	}
}

// Size 返回连接池当前大小
func (p *ConnectionPool) Size() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.connections)
}

// isTemporaryError 检查错误是否为临时性错误
func isTemporaryError(err error) bool {
	if err == nil {
		return true
	}
	// 检查错误是否实现了 net.Error 接口
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary()
	}
	return false
}

// isLocalIP 检查IP地址是否为本机IP
func isLocalIP(ip string) bool {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		// 获取接口的地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的所有地址
		for _, addr := range addrs {
			var ipNet *net.IPNet
			var ok bool

			// 检查地址类型
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
				ok = true
			case *net.IPAddr:
				ipNet = &net.IPNet{
					IP:   v.IP,
					Mask: net.CIDRMask(32, 32),
				}
				ok = true
			}

			// 如果是有效的IP地址
			if ok {
				// 将IP地址转换为字符串并比较
				if ipNet.IP.String() == ip {
					return true
				}
			}
		}
	}

	// 特殊处理本地回环地址
	if ip == "127.0.0.1" || ip == "::1" {
		return true
	}

	return false
}

// RateLimiter 速率限制器
type RateLimiter struct {
	rate     int64         // 每秒允许的请求数
	interval time.Duration // 请求间隔
	lastTime time.Time     // 上次请求时间
	mutex    sync.Mutex    // 互斥锁
}

// NewRateLimiter 创建新的速率限制器
func NewRateLimiter(ratePerSecond int) *RateLimiter {
	interval := time.Second / time.Duration(ratePerSecond)
	return &RateLimiter{
		rate:     int64(ratePerSecond),
		interval: interval,
		lastTime: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime)
	if elapsed >= rl.interval {
		// 如果距离上次发送时间超过了多个间隔，调整lastTime以保持期望速率
		intervals := elapsed / rl.interval
		rl.lastTime = rl.lastTime.Add(intervals * rl.interval)
		return true
	}
	return false
}

// Wait 等待直到允许请求
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	now := time.Now()
	elapsed := now.Sub(rl.lastTime)
	if elapsed >= rl.interval {
		// 如果距离上次发送时间超过了多个间隔，调整lastTime以保持期望速率
		intervals := elapsed / rl.interval
		rl.lastTime = rl.lastTime.Add(intervals * rl.interval)
	} else {
		// 计算需要等待的时间
		waitTime := rl.interval - elapsed
		rl.mutex.Unlock()
		time.Sleep(waitTime)
		rl.mutex.Lock()
		rl.lastTime = rl.lastTime.Add(rl.interval)
	}
	rl.mutex.Unlock()
}

// SetRate 设置新的速率
func (rl *RateLimiter) SetRate(ratePerSecond int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.rate = int64(ratePerSecond)
	rl.interval = time.Second / time.Duration(ratePerSecond)
}

// GetRate 获取当前速率
func (rl *RateLimiter) GetRate() int64 {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.rate
}
