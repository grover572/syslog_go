package server

import (
	"fmt"
	"log"
	"net"      // 提供网络操作的核心包
	"strings"  // 字符串处理工具包
	"sync"     // 提供同步原语，如WaitGroup
	"time"     // 时间相关操作

	"syslog_go/pkg/syslog"
)

// Server 表示一个可以同时监听UDP和TCP的syslog服务器
type Server struct {
	host string // 服务器监听的主机地址
	port int    // 服务器监听的端口

	udpListener *net.UDPConn    // UDP连接监听器
	tcpListener net.Listener    // TCP连接监听器

	shutdown chan struct{}     // 用于通知所有goroutine停止的信号通道
	wg       sync.WaitGroup    // 用于等待所有goroutine完成的同步计数器
}

// NewServer 创建一个新的syslog服务器实例
// host: 监听的主机地址，可以是IP或主机名
// port: 监听的端口号
func NewServer(host string, port int) *Server {
	return &Server{
		host:     host,
		port:     port,
		shutdown: make(chan struct{}), // 创建一个无缓冲的通道用于停止信号
	}
}

// Start 初始化并启动UDP和TCP监听器
func (s *Server) Start() error {
	// 启动UDP监听器
	// net.ResolveUDPAddr: 将地址字符串解析为UDP地址结构
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	// net.ListenUDP: 创建一个UDP监听器，开始监听指定地址
	s.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %v", err)
	}

	// 启动TCP监听器
	// net.Listen: 创建一个TCP监听器，开始监听指定地址
	tcpAddr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.tcpListener, err = net.Listen("tcp", tcpAddr)
	if err != nil {
		s.udpListener.Close() // 如果TCP监听失败，关闭UDP监听器
		return fmt.Errorf("failed to start TCP listener: %v", err)
	}

	// 启动UDP处理协程
	s.wg.Add(1) // 增加等待组计数
	go s.handleUDP()

	// 启动TCP处理协程
	s.wg.Add(1) // 增加等待组计数
	go s.handleTCP()

	log.Printf("Syslog server started on %s:%d (UDP & TCP)", s.host, s.port)
	return nil
}

// Stop 优雅地关闭服务器
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
	log.Println("Syslog server stopped")
}

// handleUDP 处理传入的UDP消息
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
					log.Printf("Error reading UDP message: %v", err)
				}
				continue
			}

			// 将接收到的字节转换为字符串并记录
			msg := string(buffer[:n])
			log.Printf("[UDP] From %s: %s", remoteAddr, msg)

			// Try to parse as RFC5424 first, then RFC3164
			if message, err := syslog.ParseRFC5424(msg); err == nil {
				log.Printf("[RFC5424] Priority: %d, Timestamp: %s, Hostname: %s, App: %s, Content: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else if message, err := syslog.ParseRFC3164(msg); err == nil {
				log.Printf("[RFC3164] Priority: %d, Timestamp: %s, Hostname: %s, Tag: %s, Content: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else {
				log.Printf("Failed to parse syslog message: %v", err)
			}
		}
	}
}

// handleTCP 接受并处理传入的TCP连接
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
					log.Printf("Error accepting TCP connection: %v", err)
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
func (s *Server) handleTCPConnection(conn net.Conn) {
	// 确保在函数退出时：
	defer s.wg.Done()     // 1. 减少等待组计数
	defer conn.Close()    // 2. 关闭TCP连接

	// 创建一个缓冲区用于接收TCP数据
	buffer := make([]byte, 65535)
	// RemoteAddr: 获取远程客户端的地址信息
	remoteAddr := conn.RemoteAddr()

	for {
		select {
		case <-s.shutdown: // 检查是否收到停止信号
			return
		default:
			// 设置读取超时以避免永久阻塞
			// SetReadDeadline: 设置下一次读取操作的截止时间
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			// Read: 从TCP连接读取数据，返回读取的字节数和可能的错误
			n, err := conn.Read(buffer)
			if err != nil {
				// 忽略超时错误，但对于其他错误（如连接关闭），终止该连接的处理
				if !strings.Contains(err.Error(), "timeout") {
					log.Printf("Error reading from TCP connection: %v", err)
					return
				}
				continue
			}

			// 将接收到的字节转换为字符串并记录
			msg := string(buffer[:n])
			log.Printf("[TCP] From %s: %s", remoteAddr, msg)

			// Try to parse as RFC5424 first, then RFC3164
			if message, err := syslog.ParseRFC5424(msg); err == nil {
				log.Printf("[RFC5424] Priority: %d, Timestamp: %s, Hostname: %s, App: %s, Content: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else if message, err := syslog.ParseRFC3164(msg); err == nil {
				log.Printf("[RFC3164] Priority: %d, Timestamp: %s, Hostname: %s, Tag: %s, Content: %s",
					message.Priority, message.Timestamp.Format(time.RFC3339),
					message.Hostname, message.Tag, message.Content)
			} else {
				log.Printf("Failed to parse syslog message: %v", err)
			}
		}
	}
}