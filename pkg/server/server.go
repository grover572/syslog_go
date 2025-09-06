// Package server 提供Syslog服务器的实现
package server

import (
	"fmt"
	"log"
	"net"      // 提供网络操作的核心包
	"strings"  // 字符串处理工具包
	"sync"     // 提供同步原语，如WaitGroup
	"time"     // 时间相关操作

	"syslog_go/pkg/syslog" // Syslog消息处理包
)

// Server 表示一个可以同时监听UDP和TCP的syslog服务器
// 它支持以下功能：
// 1. 同时监听UDP和TCP连接
// 2. 解析RFC3164和RFC5424格式的消息
// 3. 优雅关闭，确保所有连接正确处理
type Server struct {
	host string         // 服务器监听的主机地址
	port int            // 服务器监听的端口

	udpListener *net.UDPConn // UDP连接监听器
	tcpListener net.Listener // TCP连接监听器

	shutdown chan struct{}  // 用于通知所有goroutine停止的信号通道
	wg       sync.WaitGroup // 用于等待所有goroutine完成的同步计数器
}

// NewServer 创建一个新的syslog服务器实例
// 参数：
//   - host: 监听的主机地址，可以是IP或主机名
//   - port: 监听的端口号
// 返回值：
//   - *Server: 新创建的服务器实例
func NewServer(host string, port int) *Server {
	return &Server{
		host:     host,
		port:     port,
		shutdown: make(chan struct{}), // 创建一个无缓冲的通道用于停止信号
	}
}

// Start 初始化并启动UDP和TCP监听器
// 该方法会执行以下操作：
// 1. 启动UDP监听器
// 2. 启动TCP监听器
// 3. 启动处理协程
// 返回值：
//   - error: 如果启动过程中发生错误，返回相应的错误信息
func (s *Server) Start() error {
	// 启动UDP监听器
	// net.ResolveUDPAddr: 将地址字符串解析为UDP地址结构
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("解析UDP地址失败: %v", err)
	}

	// net.ListenUDP: 创建一个UDP监听器，开始监听指定地址
	s.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("启动UDP监听失败: %v", err)
	}

	// 启动TCP监听器
	// net.Listen: 创建一个TCP监听器，开始监听指定地址
	tcpAddr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.tcpListener, err = net.Listen("tcp", tcpAddr)
	if err != nil {
		s.udpListener.Close() // 如果TCP监听失败，关闭UDP监听器
		return fmt.Errorf("启动TCP监听失败: %v", err)
	}

	// 启动UDP处理协程
	s.wg.Add(1) // 增加等待组计数
	go s.handleUDP()

	// 启动TCP处理协程
	s.wg.Add(1) // 增加等待组计数
	go s.handleTCP()

	log.Printf("Syslog服务器已启动，监听地址: %s:%d (UDP & TCP)", s.host, s.port)
	return nil
}

// Stop 优雅地关闭服务器
// 该方法会执行以下操作：
// 1. 通知所有处理协程停止
// 2. 关闭所有网络监听器
// 3. 等待所有处理协程完成
func (s *Server) Stop() {
	// 通过关闭通道来通知所有goroutine停止
	// close: 关闭通道，所有从该通道接收数据的goroutine都会收到通知
	close(s.shutdown)

	// 关闭所有监听器
	if s.udpListener != nil {
		s.udpListener.Close() // 关闭UDP监听器，停止接收新的UDP数据包
	}
	if s.tcpListener != nil {
		s.tcpListener.Close() // 关闭TCP监听器，停止接收新的TCP连接
	}

	// 等待所有goroutine完成
	s.wg.Wait() // 阻塞直到所有goroutine都调用Done
	log.Println("Syslog服务器已停止")
}

// handleUDP 处理传入的UDP消息
// 该方法在独立的goroutine中运行，负责：
// 1. 接收UDP数据包
// 2. 解析Syslog消息
// 3. 记录消息内容
func (s *Server) handleUDP() {
	defer s.wg.Done() // 确保在函数退出时减少等待组计数

	// 创建一个缓冲区用于接收UDP数据包
	// UDP数据包的最大大小是65535字节（包括IP头和UDP头）
	buffer := make([]byte, 65535)

	for {
		select {
		case <-s.shutdown: // 检查是否收到停止信号
			return
		default:
			// 设置读取超时以避免永久阻塞
			// SetReadDeadline: 设置下一次读取操作的截止时间
			s.udpListener.SetReadDeadline(time.Now().Add(1 * time.Second))

			// ReadFromUDP: 从UDP连接读取数据，返回读取的字节数、发送者地址和可能的错误
			n, remoteAddr, err := s.udpListener.ReadFromUDP(buffer)
			if err != nil {
				// 忽略超时错误，它是正常的
				if !strings.Contains(err.Error(), "timeout") {
					log.Printf("读取UDP消息失败: %v", err)
				}
				continue
			}

			// 将接收到的字节转换为字符串并记录
			msg := string(buffer[:n])
			log.Printf("[UDP] 来自 %s 的消息: %s", remoteAddr, msg)

			// 尝试按RFC5424格式解析，如果失败则尝试RFC3164格式
			if message, err := syslog.ParseRFC5424(msg); err == nil {
				log.Printf("[RFC5424] 优先级: %d, 时间: %s, 主机: %s, 应用: %s, 内容: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else if message, err := syslog.ParseRFC3164(msg); err == nil {
				log.Printf("[RFC3164] 优先级: %d, 时间: %s, 主机: %s, 标签: %s, 内容: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else {
				log.Printf("解析Syslog消息失败: %v", err)
			}
		}
	}
}

// handleTCP 接受并处理传入的TCP连接
// 该方法在独立的goroutine中运行，负责：
// 1. 接受新的TCP连接
// 2. 为每个连接启动独立的处理协程
// 3. 处理服务器关闭时的清理工作
func (s *Server) handleTCP() {
	defer s.wg.Done() // 确保在函数退出时减少等待组计数

	for {
		select {
		case <-s.shutdown: // 检查是否收到停止信号
			return
		default:
			// 接受新的TCP连接
			// net.Listener接口不支持SetDeadline，我们通过检查错误类型来处理关闭情况
			conn, err := s.tcpListener.Accept()
			if err != nil {
				// 检查是否是由于服务器关闭导致的错误
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("接受TCP连接失败: %v", err)
				}
				continue
			}

			// 为每个新连接启动一个独立的goroutine处理
			s.wg.Add(1) // 增加等待组计数
			go s.handleTCPConnection(conn)
		}
	}
}

// handleTCPConnection 处理单个TCP连接的消息
// 该方法在独立的goroutine中运行，负责：
// 1. 读取并解析TCP连接中的数据
// 2. 处理Syslog消息
// 3. 管理连接的生命周期
// 参数：
//   - conn: 需要处理的TCP连接
func (s *Server) handleTCPConnection(conn net.Conn) {
	// 确保在函数退出时执行清理操作：
	defer s.wg.Done()     // 1. 减少等待组计数
	defer conn.Close()    // 2. 关闭TCP连接

	// 创建一个缓冲区用于接收TCP数据
	// TCP没有数据包大小限制，但我们使用与UDP相同的缓冲区大小
	buffer := make([]byte, 65535)

	// RemoteAddr: 获取远程客户端的地址信息
	// 用于日志记录和调试
	remoteAddr := conn.RemoteAddr()

	for {
		select {
		case <-s.shutdown: // 检查是否收到停止信号
			return
		default:
			// 设置读取超时以避免永久阻塞
			// SetReadDeadline: 设置下一次读取操作的截止时间
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			// Read: 从TCP连接读取数据
			// 返回值：
			//   - n: 读取的字节数
			//   - err: 可能的错误
			n, err := conn.Read(buffer)
			if err != nil {
				// 忽略超时错误，但对于其他错误（如连接关闭），终止该连接的处理
				if !strings.Contains(err.Error(), "timeout") {
					log.Printf("读取TCP连接数据失败: %v", err)
					return
				}
				continue
			}

			// 将接收到的字节转换为字符串并记录
			msg := string(buffer[:n])
			log.Printf("[TCP] 来自 %s 的消息: %s", remoteAddr, msg)

			// 尝试解析Syslog消息
			// 1. 首先尝试RFC5424格式（更新的格式）
			// 2. 如果失败，尝试RFC3164格式（传统格式）
			// 3. 如果两种格式都解析失败，记录错误
			if message, err := syslog.ParseRFC5424(msg); err == nil {
				// 成功解析为RFC5424格式
				log.Printf("[RFC5424] 优先级: %d, 时间: %s, 主机: %s, 应用: %s, 内容: %s",
					message.Priority, // 优先级（Facility * 8 + Severity）
					message.Timestamp.Format(time.RFC3339), // 标准化的时间格式
					message.Hostname, // 发送消息的主机名
					message.Tag,     // 应用程序名称
					message.Content) // 消息内容
			} else if message, err := syslog.ParseRFC3164(msg); err == nil {
				// 成功解析为RFC3164格式
				log.Printf("[RFC3164] 优先级: %d, 时间: %s, 主机: %s, 标签: %s, 内容: %s",
					message.Priority, // 优先级
					message.Timestamp.Format(time.RFC3339), // 转换为标准时间格式
					message.Hostname, // 主机名
					message.Tag,     // 进程/应用标签
					message.Content) // 消息内容
			} else {
				// 两种格式都解析失败
				log.Printf("解析Syslog消息失败: %v", err)
			}
		}
	}
}