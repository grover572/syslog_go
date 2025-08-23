package sender

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ConnectionPool 连接池
type ConnectionPool struct {
	address     string
	protocol    string
	maxSize     int
	timeout     time.Duration
	connections chan net.Conn
	mutex       sync.RWMutex
	closed      bool
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(address, protocol string, maxSize int, timeout time.Duration) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		address:     address,
		protocol:    protocol,
		maxSize:     maxSize,
		timeout:     timeout,
		connections: make(chan net.Conn, maxSize),
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
func (p *ConnectionPool) createConnection() (net.Conn, error) {
	switch p.protocol {
	case "tcp":
		return net.DialTimeout("tcp", p.address, p.timeout)
	case "udp":
		return net.DialTimeout("udp", p.address, p.timeout)
	default:
		return nil, fmt.Errorf("不支持的协议: %s", p.protocol)
	}
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

// Size 获取连接池当前大小
func (p *ConnectionPool) Size() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.connections)
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
	if now.Sub(rl.lastTime) >= rl.interval {
		rl.lastTime = now
		return true
	}
	return false
}

// Wait 等待直到允许请求
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	now := time.Now()
	next := rl.lastTime.Add(rl.interval)
	rl.mutex.Unlock()

	if next.After(now) {
		time.Sleep(next.Sub(now))
	}

	rl.mutex.Lock()
	rl.lastTime = time.Now()
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