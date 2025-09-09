//go:build linux
// +build linux

package sender

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
)

// RawSocketConn Linux版本的原始套接字连接
// 用于实现IP数据包的底层控制，支持源IP地址伪装
// 主要功能：
// 1. IP伪装：支持自定义源IP地址
// 2. 协议支持：同时支持TCP和UDP协议
// 3. 连接管理：维护TCP连接状态和序列号
// 4. 数据封装：手动构建IP、TCP、UDP数据包
// 5. 调试支持：详细的日志输出功能
type RawSocketConn struct {
	// 套接字控制
	fd         int      // 原始套接字文件描述符
	closed     bool     // 连接关闭状态
	verbose    bool     // 是否输出详细日志
	
	// 网络地址
	sourceIP   net.IP   // 源IP地址
	targetIP   net.IP   // 目标IP地址
	targetPort int      // 目标端口
	srcPort    uint16   // 源端口（随机分配）
	
	// 协议控制
	protocol   string   // 使用的协议（tcp/udp）
	connected  bool     // TCP连接状态
	seqNum     uint32   // TCP序列号
	ackNum     uint32   // TCP确认号
}

// newRawSocketConn 创建新的原始套接字连接 (Linux版本)
// 功能：
//   - 创建并配置原始套接字
//   - 解析和验证源IP和目标地址
//   - 设置套接字选项和超时
//   - 支持TCP和UDP协议
// 参数：
//   - sourceIP: 源IP地址字符串
//   - targetAddr: 目标地址字符串（格式：IP:Port）
//   - protocol: 传输协议（tcp/udp）
//   - verbose: 是否输出详细日志
// 返回值：
//   - *RawSocketConn: 原始套接字连接对象
//   - error: 创建过程中的错误
func newRawSocketConn(sourceIP, targetAddr, protocol string, verbose bool) (*RawSocketConn, error) {
	// 解析源IP地址
	srcIP := net.ParseIP(sourceIP)
	if srcIP == nil {
		return nil, fmt.Errorf("无效的源IP地址: %s", sourceIP)
	}
	srcIP = srcIP.To4()
	if srcIP == nil {
		return nil, fmt.Errorf("仅支持IPv4地址")
	}

	// 解析目标地址
	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, fmt.Errorf("无效的目标地址格式: %s", targetAddr)
	}

	// 解析目标IP
	targetIP := net.ParseIP(host)
	if targetIP == nil {
		// 尝试DNS解析
		addrs, err := net.LookupIP(host)
		if err != nil {
			return nil, fmt.Errorf("无法解析主机名 %s: %w", host, err)
		}
		for _, addr := range addrs {
			if ipv4 := addr.To4(); ipv4 != nil {
				targetIP = ipv4
				break
			}
		}
		if targetIP == nil {
			return nil, fmt.Errorf("无法找到主机 %s 的IPv4地址", host)
		}
	} else {
		targetIP = targetIP.To4()
		if targetIP == nil {
			return nil, fmt.Errorf("仅支持IPv4地址")
		}
	}

	// 解析端口
	targetPort := 0
	if _, err := fmt.Sscanf(port, "%d", &targetPort); err != nil {
		return nil, fmt.Errorf("无效的端口号: %s", port)
	}

	// 创建原始套接字
	var proto int
	if protocol == "tcp" {
		proto = syscall.IPPROTO_TCP
	} else if protocol == "udp" {
		proto = syscall.IPPROTO_UDP
	} else {
		return nil, fmt.Errorf("不支持的协议: %s", protocol)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, proto)
	if err != nil {
		return nil, fmt.Errorf("创建原始套接字失败: %w (Linux需要root权限)", err)
	}

	// 设置套接字选项
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("设置IP_HDRINCL选项失败: %w", err)
	}

	return &RawSocketConn{
		fd:         fd,
		sourceIP:   srcIP,
		targetIP:   targetIP,
		targetPort: targetPort,
		protocol:   protocol,
		closed:     false,
		verbose:    verbose,
	}, nil
}

// establishTCPConnection 建立TCP连接（三次握手）
// 功能：
//   - 实现完整的TCP三次握手过程
//   - 管理序列号和确认号
//   - 处理TCP标志位
//   - 支持超时重试机制
//   - 验证数据包的合法性
// 返回值：
//   - error: 连接建立过程中的错误
func (c *RawSocketConn) establishTCPConnection() error {
	if c.protocol != "tcp" {
		return nil
	}

	// 设置源端口和初始序列号
	c.srcPort = uint16(time.Now().UnixNano()&0xFFFF) + 32768
	c.seqNum = uint32(time.Now().UnixNano() & 0xFFFFFFFF)

	fmt.Printf("开始TCP连接建立 [%s:%d -> %s:%d]\n", c.sourceIP, c.srcPort, c.targetIP, c.targetPort)

	// 1. 发送SYN包
	if err := c.sendTCPPacket(0x0002, nil); err != nil { // SYN标志
		return fmt.Errorf("发送SYN包失败: %w", err)
	}
	if c.verbose {
		fmt.Printf("已发送SYN包，序列号: %d\n", c.seqNum)
	}

	// 2. 等待接收SYN+ACK包
	buf := make([]byte, 1500)
	maxRetries := 5 // 增加重试次数到5次
	for i := 0; i < maxRetries; i++ {
		if c.verbose {
			fmt.Printf("等待接收SYN+ACK包，尝试次数: %d\n", i+1)
		}

		// 设置读取超时为5秒
		tv := syscall.Timeval{Sec: 5, Usec: 0}
		if err := syscall.SetsockoptTimeval(c.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); err != nil {
			return fmt.Errorf("设置读取超时失败: %w", err)
		}
		if c.verbose {
			fmt.Printf("设置数据包接收超时为%d秒\n", tv.Sec)
		}

		n, _, err := syscall.Recvfrom(c.fd, buf, 0)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				if c.verbose {
					fmt.Printf("等待超时，将重试\n")
				}
				continue
			}
			return fmt.Errorf("接收数据包失败: %w", err)
		}

		if c.verbose {
			fmt.Printf("收到数据包，长度: %d 字节\n", n)
		}

		// 解析接收到的包
		if n < 40 { // IP头部(20) + TCP头部(20)
			if c.verbose {
				fmt.Printf("数据包长度不足，至少需要40字节，实际长度: %d字节\n", n)
			}
			continue
		}

		// 检查IP头部
		ipVersion := buf[0] >> 4
		if ipVersion != 4 {
			if c.verbose {
				fmt.Printf("非IPv4数据包，版本: %d\n", ipVersion)
			}
			continue
		}

		// 检查是否是TCP协议
		ipProtocol := buf[9]
		if ipProtocol != syscall.IPPROTO_TCP {
			if c.verbose {
				fmt.Printf("非TCP协议，协议号: %d（TCP协议号应为: %d）\n", ipProtocol, syscall.IPPROTO_TCP)
			}
			continue
		}
		if c.verbose {
			fmt.Printf("收到TCP协议数据包\n")
		}

		// 检查源IP和目标IP是否匹配
		// buf[12:16]是源IP，buf[16:20]是目标IP
		// 对于收到的SYN+ACK包，我们需要验证数据包是否与当前连接相关
		srcIP := net.IP(buf[12:16])
		dstIP := net.IP(buf[16:20])
		if c.verbose {
			fmt.Printf("收到的数据包IP信息:\n")
			fmt.Printf("  源IP: %v，目标IP: %v\n", srcIP, dstIP)
			fmt.Printf("  本地配置 - 源IP: %v，目标IP: %v\n", c.sourceIP, c.targetIP)
		}
		
		// 检查数据包是否与当前连接相关
		// 至少目标IP应该是我们发送SYN包时使用的源IP
		if !bytes.Equal(dstIP, c.sourceIP.To4()) {
			if c.verbose {
				fmt.Printf("忽略与当前连接无关的数据包\n")
			}
			continue
		}

		// 检查TCP头部和标志位
		ipHeaderLen := (buf[0] & 0x0F) * 4 // IP头部长度
		fmt.Printf("IP头部长度: %d字节\n", ipHeaderLen)
		tcpOffset := ipHeaderLen
		
		// 检查源端口和目标端口
		srcPort := binary.BigEndian.Uint16(buf[tcpOffset:tcpOffset+2])
		dstPort := binary.BigEndian.Uint16(buf[tcpOffset+2:tcpOffset+4])
		fmt.Printf("收到的数据包端口信息:\n")
		fmt.Printf("  源端口: %d，目标端口: %d\n", srcPort, dstPort)
		fmt.Printf("  本地配置 - 源端口: %d，目标端口: %d\n", c.srcPort, c.targetPort)
		
		// 检查端口匹配
		// 对于收到的SYN+ACK包，源端口应该是目标端口，目标端口应该是源端口
		if srcPort != uint16(c.targetPort) || dstPort != c.srcPort {
			fmt.Printf("端口不匹配:\n")
			fmt.Printf("  收到的包 - 源端口: %d，目标端口: %d\n", srcPort, dstPort)
			fmt.Printf("  期望的值 - 源端口: %d，目标端口: %d\n", c.targetPort, c.srcPort)
			continue
		}

		// 检查TCP标志位
		tcpFlags := buf[tcpOffset+13]
		if c.verbose {
			fmt.Printf("TCP标志位分析:\n")
			fmt.Printf("  收到的标志位: 0x%02x\n", tcpFlags)
			fmt.Printf("  标志位含义:\n")
			fmt.Printf("    FIN: %v\n", tcpFlags&0x01 != 0)
			fmt.Printf("    SYN: %v\n", tcpFlags&0x02 != 0)
			fmt.Printf("    RST: %v\n", tcpFlags&0x04 != 0)
			fmt.Printf("    PSH: %v\n", tcpFlags&0x08 != 0)
			fmt.Printf("    ACK: %v\n", tcpFlags&0x10 != 0)
			fmt.Printf("    URG: %v\n", tcpFlags&0x20 != 0)
		}
		
		// 检查是否包含SYN和ACK标志
		if tcpFlags != 0x12 { // SYN+ACK = 0x12
			if c.verbose {
				fmt.Printf("  警告：期望收到SYN+ACK (0x12)，但收到了不同的标志位组合\n")
			}
			continue
		}
		if c.verbose {
			fmt.Printf("  确认：收到了正确的SYN+ACK标志位组合\n")
		}

		// 获取确认号和对方的序列号
		c.ackNum = binary.BigEndian.Uint32(buf[tcpOffset+8:tcpOffset+12]) + 1
		c.seqNum = binary.BigEndian.Uint32(buf[tcpOffset+4:tcpOffset+8])
		if c.verbose {
			fmt.Printf("收到SYN+ACK包，确认号: %d，序列号: %d\n", c.ackNum, c.seqNum)
		}

		// 3. 发送ACK包
		if err := c.sendTCPPacket(0x0010, nil); err != nil { // ACK标志
			return fmt.Errorf("发送ACK包失败: %w", err)
		}
		if c.verbose {
			fmt.Printf("已发送ACK包\n")
		}

		c.connected = true
		fmt.Printf("TCP连接建立成功 [%s:%d -> %s:%d]\n", c.sourceIP, c.srcPort, c.targetIP, c.targetPort)
		return nil
	}

	return fmt.Errorf("TCP连接建立失败: 未收到SYN+ACK包")
}

// sendTCPPacket 发送TCP数据包
// 功能：
//   - 构建完整的TCP/IP数据包
//   - 支持各种TCP标志位组合
//   - 自动计算IP和TCP校验和
//   - 维护TCP序列号和确认号
// 参数：
//   - flags: TCP标志位（如SYN、ACK、PSH等）
//   - data: 要发送的数据（可选）
// 返回值：
//   - error: 发送过程中的错误
func (c *RawSocketConn) sendTCPPacket(flags uint16, data []byte) error {
	fmt.Printf("准备发送TCP数据包，标志位: 0x%02x\n", flags)

	// 构建IP头部
	ipHeader := make([]byte, 20)
	ipHeader[0] = 0x45                                  // 版本(4)和头部长度(5)
	ipHeader[1] = 0x00                                  // 服务类型
	ipHeaderLen := 20

	// TCP头部
	tcpHeader := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpHeader[0:2], c.srcPort)
	binary.BigEndian.PutUint16(tcpHeader[2:4], uint16(c.targetPort))
	binary.BigEndian.PutUint32(tcpHeader[4:8], c.seqNum)
	binary.BigEndian.PutUint32(tcpHeader[8:12], c.ackNum)
	tcpHeader[12] = 5 << 4                                  // 数据偏移
	tcpHeader[13] = byte(flags)
	binary.BigEndian.PutUint16(tcpHeader[14:16], 65535)    // 窗口大小
	binary.BigEndian.PutUint16(tcpHeader[16:18], 0)        // 校验和
	binary.BigEndian.PutUint16(tcpHeader[18:20], 0)        // 紧急指针

	fmt.Printf("TCP头部 - 源端口: %d, 目标端口: %d, 序列号: %d, 确认号: %d\n", 
		c.srcPort, c.targetPort, c.seqNum, c.ackNum)

	// 计算TCP校验和
	tcpChecksum := calculateTCPChecksum(c.sourceIP, c.targetIP, tcpHeader, data)
	binary.BigEndian.PutUint16(tcpHeader[16:18], tcpChecksum)

	// 设置IP头部其他字段
	totalLen := uint16(ipHeaderLen + len(tcpHeader) + len(data))
	binary.BigEndian.PutUint16(ipHeader[2:4], totalLen)
	binary.BigEndian.PutUint16(ipHeader[4:6], uint16(time.Now().UnixNano()&0xFFFF))
	binary.BigEndian.PutUint16(ipHeader[6:8], 0)
	ipHeader[8] = 64
	ipHeader[9] = syscall.IPPROTO_TCP
	binary.BigEndian.PutUint16(ipHeader[10:12], 0)
	copy(ipHeader[12:16], c.sourceIP.To4())
	copy(ipHeader[16:20], c.targetIP.To4())

	// 计算IP校验和
	ipChecksum := calculateIPChecksum(ipHeader)
	binary.BigEndian.PutUint16(ipHeader[10:12], ipChecksum)

	// 组装完整的数据包
	packet := make([]byte, len(ipHeader)+len(tcpHeader)+len(data))
	copy(packet[0:len(ipHeader)], ipHeader)
	copy(packet[len(ipHeader):len(ipHeader)+len(tcpHeader)], tcpHeader)
	if len(data) > 0 {
		copy(packet[len(ipHeader)+len(tcpHeader):], data)
	}

	// 发送数据包
	addr := syscall.SockaddrInet4{
		Port: c.targetPort,
		Addr: [4]byte{c.targetIP[0], c.targetIP[1], c.targetIP[2], c.targetIP[3]},
	}

	fmt.Printf("IP头部 - 源IP: %v, 目标IP: %v\n", net.IP(c.sourceIP), net.IP(c.targetIP))
	fmt.Printf("准备发送到地址: %v:%d\n", net.IP(addr.Addr[:]), addr.Port)

	err := syscall.Sendto(c.fd, packet, 0, &addr)
	if err != nil {
		if c.verbose {
			fmt.Printf("发送数据包失败: %v\n", err)
		}
		return err
	}
	if c.verbose {
		fmt.Printf("数据包发送成功，长度: %d字节\n", len(packet))
	}
	return nil
}

// Write 发送数据
// 功能：
//   - 支持TCP和UDP协议的数据发送
//   - 自动处理TCP连接建立
//   - 手动构建IP和传输层数据包
//   - 计算校验和确保数据完整性
// 参数：
//   - data: 要发送的数据
// 返回值：
//   - int: 发送的字节数
//   - error: 发送过程中的错误
func (c *RawSocketConn) Write(data []byte) (int, error) {
	// 检查连接状态
	if c.closed {
		return 0, fmt.Errorf("连接已关闭")
	}

	// TCP协议特殊处理：确保连接已建立
	if c.protocol == "tcp" && !c.connected {
		if err := c.establishTCPConnection(); err != nil {
			return 0, err
		}
	}

	switch c.protocol {
	case "tcp":
		// 发送数据包
		if err := c.sendTCPPacket(0x0018, data); err != nil { // PSH+ACK标志
			return 0, err
		}
		// 更新序列号
		c.seqNum += uint32(len(data))
		return len(data), nil
	case "udp":
		// 构建IP头部
		ipHeader := make([]byte, 20)
		ipHeader[0] = 0x45                                  // 版本(4)和头部长度(5)
		ipHeader[1] = 0x00                                  // 服务类型
		ipHeaderLen := 20

		// UDP头部
		udpHeader := make([]byte, 8)
		srcPort := uint16(time.Now().UnixNano()&0xFFFF) + 32768 // 随机源端口
		dstPort := uint16(c.targetPort)

		binary.BigEndian.PutUint16(udpHeader[0:2], srcPort)
		binary.BigEndian.PutUint16(udpHeader[2:4], dstPort)
		binary.BigEndian.PutUint16(udpHeader[4:6], uint16(8+len(data))) // UDP长度
		// 校验和字段先设为0
		binary.BigEndian.PutUint16(udpHeader[6:8], 0)

		// 计算UDP校验和
		udpChecksum := calculateUDPChecksum(c.sourceIP, c.targetIP, udpHeader, data)
		binary.BigEndian.PutUint16(udpHeader[6:8], udpChecksum)

		// 设置IP头部其他字段
		totalLen := uint16(ipHeaderLen + len(udpHeader) + len(data))
		binary.BigEndian.PutUint16(ipHeader[2:4], totalLen)
		binary.BigEndian.PutUint16(ipHeader[4:6], uint16(time.Now().UnixNano()&0xFFFF)) // ID字段
		binary.BigEndian.PutUint16(ipHeader[6:8], 0)                                     // 标志和片偏移
		ipHeader[8] = 64                                                                  // TTL
		ipHeader[9] = syscall.IPPROTO_UDP                                               // 协议
		// 校验和字段先设为0
		binary.BigEndian.PutUint16(ipHeader[10:12], 0)
		copy(ipHeader[12:16], c.sourceIP.To4())
		copy(ipHeader[16:20], c.targetIP.To4())

		// 计算IP头部校验和
		ipChecksum := calculateIPChecksum(ipHeader)
		binary.BigEndian.PutUint16(ipHeader[10:12], ipChecksum)

		// 组装完整的数据包
		packet := make([]byte, len(ipHeader)+len(udpHeader)+len(data))
		copy(packet[0:len(ipHeader)], ipHeader)
		copy(packet[len(ipHeader):len(ipHeader)+len(udpHeader)], udpHeader)
		copy(packet[len(ipHeader)+len(udpHeader):], data)

		// 构建目标地址结构
		addr := syscall.SockaddrInet4{
			Port: c.targetPort,
			Addr: [4]byte{c.targetIP[0], c.targetIP[1], c.targetIP[2], c.targetIP[3]},
		}

		// 发送数据包
		if err := syscall.Sendto(c.fd, packet, 0, &addr); err != nil {
			return 0, fmt.Errorf("发送数据包失败: %w", err)
		}

		return len(data), nil
	default:
		return 0, fmt.Errorf("不支持的协议: %s", c.protocol)
	}
}

// Read 读取数据 (原始套接字通常不用于读取)
// 功能：
//   - 从原始套接字读取数据
//   - 由于原始套接字的特性，通常不支持直接读取
//   - 返回不支持操作的错误
// 参数：
//   - b: 用于存储读取数据的缓冲区
// 返回值：
//   - int: 读取的字节数
//   - error: 读取过程中的错误
func (c *RawSocketConn) Read(b []byte) (int, error) {
	return 0, fmt.Errorf("原始套接字不支持读取操作")
}

// Close 关闭连接
// 功能：
//   - 关闭原始套接字连接
//   - 释放系统资源
//   - 支持幂等操作（多次调用安全）
// 返回值：
//   - error: 关闭过程中的错误
func (c *RawSocketConn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return syscall.Close(c.fd)
}

// LocalAddr 返回本地地址
func (c *RawSocketConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: c.sourceIP, Port: 0}
}

// RemoteAddr 返回远程地址
func (c *RawSocketConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: c.targetIP, Port: c.targetPort}
}

// SetDeadline 设置读写超时
func (c *RawSocketConn) SetDeadline(t time.Time) error {
	return nil // 原始套接字不支持超时设置
}

// SetReadDeadline 设置读超时
func (c *RawSocketConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline 设置写超时
func (c *RawSocketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// calculateIPChecksum 计算IP校验和
// 功能：
//   - 计算IP头部的16位校验和
//   - 采用Internet校验和算法
//   - 支持奇数长度的数据
// 参数：
//   - header: IP头部数据
// 返回值：
//   - uint16: 计算得到的校验和
func calculateIPChecksum(header []byte) uint16 {
	var sum uint32
	count := len(header)

	// 每次处理2个字节（16位）
	for i := 0; i < count-1; i += 2 {
		sum += uint32(header[i])<<8 | uint32(header[i+1])
	}

	// 如果长度为奇数，处理最后一个字节
	if count&1 == 1 {
		sum += uint32(header[count-1]) << 8
	}

	// 将高16位加到低16位，直到高16位为0
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 返回校验和的反码
	return ^uint16(sum)
}

// calculateTCPChecksum 计算TCP校验和
// 功能：
//   - 计算TCP数据包的16位校验和
//   - 包含TCP伪头部、TCP头部和数据部分
//   - 支持奇数长度的数据
// 参数：
//   - srcIP: 源IP地址
//   - dstIP: 目标IP地址
//   - tcpHeader: TCP头部数据
//   - data: TCP数据部分
// 返回值：
//   - uint16: 计算得到的校验和
func calculateTCPChecksum(srcIP, dstIP net.IP, tcpHeader []byte, data []byte) uint16 {
	// 1. 构建TCP伪头部（12字节）
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP.To4())                          // 源IP
	copy(pseudoHeader[4:8], dstIP.To4())                          // 目标IP
	pseudoHeader[8] = 0                                           // 保留字节
	pseudoHeader[9] = syscall.IPPROTO_TCP                        // 协议类型
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpHeader)+len(data))) // TCP长度

	// 2. 计算校验和
	var sum uint32

	// 处理伪头部（按16位累加）
	for i := 0; i < len(pseudoHeader)-1; i += 2 {
		sum += uint32(pseudoHeader[i])<<8 | uint32(pseudoHeader[i+1])
	}

	// 处理TCP头部（按16位累加）
	for i := 0; i < len(tcpHeader)-1; i += 2 {
		sum += uint32(tcpHeader[i])<<8 | uint32(tcpHeader[i+1])
	}

	// 处理数据部分（按16位累加）
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	// 处理奇数长度的数据
	if len(data)&1 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// 将高16位加到低16位，直到高16位为0
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 返回校验和的反码
	return ^uint16(sum)
}

// calculateUDPChecksum 计算UDP校验和
// 功能：
//   - 计算UDP数据包的16位校验和
//   - 包含UDP伪头部、UDP头部和数据部分
//   - 支持奇数长度的数据
// 参数：
//   - srcIP: 源IP地址
//   - dstIP: 目标IP地址
//   - udpHeader: UDP头部数据
//   - data: UDP数据部分
// 返回值：
//   - uint16: 计算得到的校验和
func calculateUDPChecksum(srcIP, dstIP net.IP, udpHeader []byte, data []byte) uint16 {
	// 1. 构建UDP伪头部（12字节）
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP.To4())                          // 源IP
	copy(pseudoHeader[4:8], dstIP.To4())                          // 目标IP
	pseudoHeader[8] = 0                                           // 保留字节
	pseudoHeader[9] = syscall.IPPROTO_UDP                        // 协议类型
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(udpHeader)+len(data))) // UDP长度

	// 2. 计算校验和
	var sum uint32

	// 处理伪头部（按16位累加）
	for i := 0; i < len(pseudoHeader)-1; i += 2 {
		sum += uint32(pseudoHeader[i])<<8 | uint32(pseudoHeader[i+1])
	}

	// 处理UDP头部（按16位累加）
	for i := 0; i < len(udpHeader)-1; i += 2 {
		sum += uint32(udpHeader[i])<<8 | uint32(udpHeader[i+1])
	}

	// 处理数据部分（按16位累加）
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	// 处理奇数长度的数据
	if len(data)&1 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// 将高16位加到低16位，直到高16位为0
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 返回校验和的反码
	return ^uint16(sum)
}