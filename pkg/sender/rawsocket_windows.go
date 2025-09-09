//go:build windows
// +build windows

package sender

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
)

// Windows系统的常量定义
const (
	AF_INET     = 2
	SOCK_RAW    = 3
	IPPROTO_TCP = 6
	IPPROTO_UDP = 17
	IPPROTO_IP  = 0
	IP_HDRINCL  = 2
)

// RawSocketConn Windows版本的原始套接字连接
type RawSocketConn struct {
	fd       syscall.Handle
	sourceIP net.IP
	targetIP net.IP
	targetPort int
	protocol string
	closed   bool
	verbose  bool     // 是否输出详细日志
}

// NewRawSocketConn 创建新的原始套接字连接 (Windows版本)
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
	var fd syscall.Handle
	if protocol == "tcp" {
		fd, err = syscall.Socket(AF_INET, SOCK_RAW, IPPROTO_TCP)
	} else if protocol == "udp" {
		fd, err = syscall.Socket(AF_INET, SOCK_RAW, IPPROTO_UDP)
	} else {
		return nil, fmt.Errorf("不支持的协议: %s", protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("创建原始套接字失败: %w (Windows需要管理员权限)", err)
	}

	// 设置IP_HDRINCL选项，允许我们自己构造IP头
	if err := syscall.SetsockoptInt(fd, IPPROTO_IP, IP_HDRINCL, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("设置IP_HDRINCL失败: %w", err)
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

// Write 发送数据
func (c *RawSocketConn) Write(data []byte) (int, error) {
	if c.closed {
		return 0, fmt.Errorf("连接已关闭")
	}

	// 构造完整的数据包
	var packet []byte
	if c.protocol == "tcp" {
		packet = c.buildTCPPacket(data)
	} else if c.protocol == "udp" {
		packet = c.buildUDPPacket(data)
	} else {
		return 0, fmt.Errorf("不支持的协议: %s", c.protocol)
	}

	// 构造目标地址
	addr := &syscall.SockaddrInet4{
		Port: c.targetPort,
	}
	copy(addr.Addr[:], c.targetIP.To4())

	// 发送数据包
	err := syscall.Sendto(c.fd, packet, 0, addr)
	if err != nil {
		if c.verbose {
			fmt.Printf("发送数据包失败: %v\n", err)
		}
		return 0, fmt.Errorf("发送数据包失败: %w", err)
	}

	if c.verbose {
		fmt.Printf("数据包发送成功，长度: %d字节\n", len(packet))
	}

	return len(data), nil
}

// Read 读取数据 (原始套接字通常不用于读取)
func (c *RawSocketConn) Read(b []byte) (int, error) {
	return 0, fmt.Errorf("原始套接字不支持读取操作")
}

// Close 关闭连接
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

// buildIPHeader 构造IP头
func (c *RawSocketConn) buildIPHeader(protocol uint8, dataLen int) []byte {
	header := make([]byte, 20) // IP头固定20字节

	// 版本(4位) + 头长度(4位)
	header[0] = 0x45 // IPv4, 头长度20字节

	// 服务类型
	header[1] = 0x00

	// 总长度
	totalLen := 20 + dataLen
	binary.BigEndian.PutUint16(header[2:4], uint16(totalLen))

	// 标识
	binary.BigEndian.PutUint16(header[4:6], 0x1234)

	// 标志(3位) + 片偏移(13位)
	binary.BigEndian.PutUint16(header[6:8], 0x4000) // Don't Fragment

	// TTL
	header[8] = 64

	// 协议
	header[9] = protocol

	// 校验和 (先设为0)
	header[10] = 0
	header[11] = 0

	// 源IP地址
	copy(header[12:16], c.sourceIP.To4())

	// 目标IP地址
	copy(header[16:20], c.targetIP.To4())

	// 计算校验和
	checksum := c.calculateChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	return header
}

// buildTCPPacket 构造TCP数据包
func (c *RawSocketConn) buildTCPPacket(data []byte) []byte {
	// 构造TCP头 (20字节)
	tcpHeader := make([]byte, 20)

	// 源端口 (使用随机端口)
	srcPort := uint16(32768 + time.Now().UnixNano()%32768) // 使用32768-65535范围的随机端口
	binary.BigEndian.PutUint16(tcpHeader[0:2], srcPort)

	// 目标端口
	binary.BigEndian.PutUint16(tcpHeader[2:4], uint16(c.targetPort))

	// 序列号 (使用时间戳作为随机值)
	seqNum := uint32(time.Now().UnixNano() & 0xFFFFFFFF)
	binary.BigEndian.PutUint32(tcpHeader[4:8], seqNum)

	// 确认号 (对于第一个包设为0)
	binary.BigEndian.PutUint32(tcpHeader[8:12], 0)

	// 数据偏移(4位) + 保留(3位) + 标志(9位)
	tcpHeader[12] = 0x50 // 数据偏移20字节
	tcpHeader[13] = 0x18 // PSH+ACK 标志，表示这是一个数据包且已建立连接

	// 窗口大小
	binary.BigEndian.PutUint16(tcpHeader[14:16], 8192)

	// 校验和 (先设为0)
	tcpHeader[16] = 0
	tcpHeader[17] = 0

	// 紧急指针
	tcpHeader[18] = 0
	tcpHeader[19] = 0

	// 计算TCP校验和
	checksum := c.calculateTCPChecksum(tcpHeader, data)
	binary.BigEndian.PutUint16(tcpHeader[16:18], checksum)

	// 构造IP头
	ipHeader := c.buildIPHeader(IPPROTO_TCP, len(tcpHeader)+len(data))

	// 组合完整数据包
	packet := make([]byte, len(ipHeader)+len(tcpHeader)+len(data))
	copy(packet, ipHeader)
	copy(packet[len(ipHeader):], tcpHeader)
	copy(packet[len(ipHeader)+len(tcpHeader):], data)

	return packet
}

// buildUDPPacket 构造UDP数据包
func (c *RawSocketConn) buildUDPPacket(data []byte) []byte {
	// 构造UDP头 (8字节)
	udpHeader := make([]byte, 8)

	// 源端口 (使用随机端口)
	srcPort := uint16(32768 + time.Now().UnixNano()%32768) // 使用32768-65535范围的随机端口
	binary.BigEndian.PutUint16(udpHeader[0:2], srcPort)

	// 目标端口
	binary.BigEndian.PutUint16(udpHeader[2:4], uint16(c.targetPort))

	// 长度
	udpLen := len(udpHeader) + len(data)
	binary.BigEndian.PutUint16(udpHeader[4:6], uint16(udpLen))

	// 校验和 (先设为0)
	udpHeader[6] = 0
	udpHeader[7] = 0

	// 计算UDP校验和
	checksum := c.calculateUDPChecksum(udpHeader, data)
	binary.BigEndian.PutUint16(udpHeader[6:8], checksum)

	// 构造IP头
	ipHeader := c.buildIPHeader(IPPROTO_UDP, len(udpHeader)+len(data))

	// 组合完整数据包
	packet := make([]byte, len(ipHeader)+len(udpHeader)+len(data))
	copy(packet, ipHeader)
	copy(packet[len(ipHeader):], udpHeader)
	copy(packet[len(ipHeader)+len(udpHeader):], data)

	return packet
}

// calculateChecksum 计算IP头校验和
func (c *RawSocketConn) calculateChecksum(data []byte) uint16 {
	sum := uint32(0)
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(data[i])<<8 + uint32(data[i+1])
		} else {
			sum += uint32(data[i]) << 8
		}
	}
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return uint16(^sum)
}

// calculateTCPChecksum 计算TCP校验和
func (c *RawSocketConn) calculateTCPChecksum(tcpHeader, data []byte) uint16 {
	// TCP伪头部
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], c.sourceIP.To4())
	copy(pseudoHeader[4:8], c.targetIP.To4())
	pseudoHeader[8] = 0
	pseudoHeader[9] = IPPROTO_TCP
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpHeader)+len(data)))

	// 组合数据进行校验和计算
	combined := append(pseudoHeader, tcpHeader...)
	combined = append(combined, data...)

	return c.calculateChecksum(combined)
}

// calculateUDPChecksum 计算UDP校验和
func (c *RawSocketConn) calculateUDPChecksum(udpHeader, data []byte) uint16 {
	// UDP伪头部
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], c.sourceIP.To4())
	copy(pseudoHeader[4:8], c.targetIP.To4())
	pseudoHeader[8] = 0
	pseudoHeader[9] = IPPROTO_UDP
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(udpHeader)+len(data)))

	// 组合数据进行校验和计算
	combined := append(pseudoHeader, udpHeader...)
	combined = append(combined, data...)

	return c.calculateChecksum(combined)
}