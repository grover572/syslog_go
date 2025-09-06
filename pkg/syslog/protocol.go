// Package syslog 提供Syslog协议的实现
// 支持RFC3164和RFC5424两种格式的消息解析和生成
package syslog

import (
	"fmt"
	"regexp"    // 用于正则表达式匹配
	"strings"   // 字符串处理
	"time"      // 时间处理
)

// SyslogFormat 定义Syslog格式类型
// 用于指定消息使用的格式标准
type SyslogFormat string

// Syslog格式常量
const (
	RFC3164 SyslogFormat = "rfc3164" // BSD Syslog协议（传统格式）
	RFC5424 SyslogFormat = "rfc5424" // Syslog协议（现代格式）
)

// Message 表示一个Syslog消息
// 包含了Syslog消息的所有组成部分
type Message struct {
	Priority     int          // 优先级 (Facility * 8 + Severity)
	Timestamp    time.Time    // 消息生成的时间戳
	Hostname     string       // 生成消息的主机名
	Tag          string       // 生成消息的程序名称
	PID          string       // 生成消息的进程ID
	Content      string       // 消息的实际内容
	SyslogFormat SyslogFormat // 使用的Syslog格式（RFC3164或RFC5424）
}

// NewMessage 创建新的Syslog消息
// 参数：
//   - priority: 优先级值（Facility * 8 + Severity）
//   - hostname: 主机名
//   - tag: 程序名称
//   - content: 消息内容
//   - format: Syslog格式（RFC3164或RFC5424）
// 返回值：
//   - *Message: 新创建的Syslog消息对象
func NewMessage(priority int, hostname, tag, content string, format SyslogFormat) *Message {
	return &Message{
		Priority:     priority,
		Timestamp:    time.Now(),         // 使用当前时间
		Hostname:     hostname,
		Tag:          tag,
		Content:      content,
		SyslogFormat: format,
	}
}

// Format 将消息格式化为指定的Syslog格式字符串
// 根据消息的SyslogFormat字段选择相应的格式化方法
// 返回值：
//   - string: 格式化后的Syslog消息字符串
func (m *Message) Format() string {
	switch m.SyslogFormat {
	case RFC5424:
		return m.formatRFC5424()
	default:
		return m.formatRFC3164()
	}
}

// formatRFC3164 格式化为RFC3164格式
// RFC3164格式规范：
// <Priority>Timestamp Hostname Tag[PID]: Content
// 示例：<34>Oct 11 22:14:15 mymachine su[123]: 'su root' failed
func (m *Message) formatRFC3164() string {
	// RFC3164时间戳格式: Jan 02 15:04:05
	timestamp := m.Timestamp.Format("Jan 02 15:04:05")

	// 构建标签部分
	// 如果有PID，格式为"Tag[PID]"
	// 如果没有PID，只使用Tag
	var tagPart string
	if m.PID != "" {
		tagPart = fmt.Sprintf("%s[%s]", m.Tag, m.PID)
	} else {
		tagPart = m.Tag
	}

	// 如果标签为空，使用默认标签
	if tagPart == "" {
		tagPart = "syslog_go"
	}

	// 组装最终的消息格式
	return fmt.Sprintf("<%d>%s %s %s: %s",
		m.Priority,  // 优先级
		timestamp,   // 时间戳
		m.Hostname,  // 主机名
		tagPart,     // 标签（可能包含PID）
		m.Content)   // 消息内容
}

// formatRFC5424 格式化为RFC5424格式
// RFC5424格式规范：
// <Priority>Version Timestamp Hostname App-Name ProcID MsgID Structured-Data Msg
// 示例：<34>1 2003-10-11T22:14:15.003Z mymachine su - ID47 - 'su root' failed
func (m *Message) formatRFC5424() string {
	// RFC5424时间戳格式: 2006-01-02T15:04:05.000Z
	// 使用UTC时间并格式化为ISO格式
	timestamp := m.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z")

	// 处理各个字段，空值用 "-" 表示
	// RFC5424规定必须字段不能为空，应该用"-"代替
	hostname := m.Hostname
	if hostname == "" {
		hostname = "-"
	}

	appName := m.Tag
	if appName == "" {
		appName = "syslog_go"
	}

	procID := m.PID
	if procID == "" {
		procID = "-"
	}

	msgID := "-"          // 消息ID，通常为空
	structuredData := "-" // 结构化数据，暂时不支持

	// 组装最终的消息格式
	return fmt.Sprintf("<%d>1 %s %s %s %s %s %s %s",
		m.Priority,      // 优先级
		timestamp,       // ISO格式的时间戳
		hostname,        // 主机名
		appName,         // 应用名称
		procID,          // 进程ID
		msgID,           // 消息ID
		structuredData,  // 结构化数据
		m.Content)       // 消息内容
}

// ParseRFC3164 解析RFC3164格式的syslog消息
// RFC3164格式规范：
// <Priority>Timestamp Hostname Tag[PID]: Content
// 示例：<34>Oct 11 22:14:15 mymachine su[123]: 'su root' failed
//
// 参数：
//   - msg: 要解析的Syslog消息字符串
// 返回值：
//   - *Message: 解析成功后的消息对象
//   - error: 解析过程中的错误，如果格式不正确则返回错误
func ParseRFC3164(msg string) (*Message, error) {
	// 使用正则表达式匹配RFC3164格式
	// 正则表达式分组：
	// 1. Priority: 优先级数字
	// 2. Timestamp: MMM DD HH:MM:SS 格式的时间戳
	// 3. Hostname: 主机名
	// 4. Tag: 程序名称
	// 5. PID: 可选的进程ID
	// 6. Content: 消息内容
	pattern := regexp.MustCompile(`^<(\d+)>([A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+([^\s]+)\s+([^:\[]+)(?:\[(\d+)\])?:\s+(.+)$`)
	matches := pattern.FindStringSubmatch(msg)
	if matches == nil {
		return nil, fmt.Errorf("无效的RFC3164格式")
	}

	// 解析优先级（Priority = Facility * 8 + Severity）
	var priority int
	fmt.Sscanf(matches[1], "%d", &priority)

	// 解析时间戳
	// RFC3164的时间戳不包含年份，需要添加当前年份
	currentYear := time.Now().Year()
	timestamp, err := time.Parse("Jan 2 15:04:05 2006", matches[2]+fmt.Sprintf(" %d", currentYear))
	if err != nil {
		return nil, fmt.Errorf("无效的时间戳格式: %v", err)
	}

	// 创建并返回消息对象
	message := &Message{
		Priority:     priority,   // 优先级
		Timestamp:    timestamp,  // 解析后的时间戳
		Hostname:     matches[3], // 主机名
		Tag:          matches[4], // 程序名称
		PID:          matches[5], // 进程ID（可能为空）
		Content:      matches[6], // 消息内容
		SyslogFormat: RFC3164,    // 标记为RFC3164格式
	}

	return message, nil
}

// ParseRFC5424 解析RFC5424格式的syslog消息
// RFC5424格式规范：
// <Priority>Version Timestamp Hostname App-Name ProcID MsgID Structured-Data Msg
// 示例：<34>1 2003-10-11T22:14:15.003Z mymachine su - ID47 - 'su root' failed
//
// 参数：
//   - msg: 要解析的Syslog消息字符串
// 返回值：
//   - *Message: 解析成功后的消息对象
//   - error: 解析过程中的错误，如果格式不正确则返回错误
func ParseRFC5424(msg string) (*Message, error) {
	// 使用正则表达式匹配RFC5424格式
	// 正则表达式分组：
	// 1. Priority: 优先级数字
	// 2. Timestamp: ISO格式的时间戳
	// 3. Hostname: 主机名
	// 4. App-Name: 应用名称
	// 5. ProcID: 进程ID
	// 6. MsgID: 消息ID
	// 7. Structured-Data: 结构化数据
	// 8. Msg: 消息内容
	pattern := regexp.MustCompile(`^<(\d+)>1\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`)
	matches := pattern.FindStringSubmatch(msg)
	if matches == nil {
		return nil, fmt.Errorf("无效的RFC5424格式")
	}

	// 解析优先级（Priority = Facility * 8 + Severity）
	var priority int
	fmt.Sscanf(matches[1], "%d", &priority)

	// 解析时间戳（RFC5424使用ISO格式的时间戳）
	timestamp, err := time.Parse(time.RFC3339, matches[2])
	if err != nil {
		return nil, fmt.Errorf("无效的时间戳格式: %v", err)
	}

	// 处理特殊值（RFC5424使用"-"表示空值）
	hostname := matches[3]
	if hostname == "-" {
		hostname = ""
	}

	appName := matches[4]
	if appName == "-" {
		appName = ""
	}

	procID := matches[5]
	if procID == "-" {
		procID = ""
	}

	// 创建并返回消息对象
	message := &Message{
		Priority:     priority,   // 优先级
		Timestamp:    timestamp,  // 解析后的时间戳
		Hostname:     hostname,   // 主机名
		Tag:          appName,    // 应用名称
		PID:          procID,     // 进程ID
		Content:      matches[8], // 消息内容
		SyslogFormat: RFC5424,    // 标记为RFC5424格式
	}

	return message, nil
}

// SetTimestamp 设置自定义时间戳
// 参数：
//   - t: 要设置的新时间戳
func (m *Message) SetTimestamp(t time.Time) {
	m.Timestamp = t
}

// SetPID 设置进程ID
// 参数：
//   - pid: 要设置的进程ID字符串
func (m *Message) SetPID(pid string) {
	m.PID = pid
}

// SetHostname 设置主机名
// 参数：
//   - hostname: 要设置的主机名字符串
func (m *Message) SetHostname(hostname string) {
	m.Hostname = hostname
}

// SetTag 设置标签（程序名称或应用名称）
// 参数：
//   - tag: 要设置的标签字符串
func (m *Message) SetTag(tag string) {
	m.Tag = tag
}

// SetContent 设置消息内容
// 参数：
//   - content: 要设置的消息内容字符串
func (m *Message) SetContent(content string) {
	m.Content = content
}

// SetPriority 设置优先级
// 参数：
//   - priority: 要设置的优先级值（Priority = Facility * 8 + Severity）
func (m *Message) SetPriority(priority int) {
	m.Priority = priority
}

// GetFacility 获取设施值
// 返回值：
//   - int: 通过右移3位从优先级中提取的设施值（0-23）
func (m *Message) GetFacility() int {
	return m.Priority >> 3
}

// GetSeverity 获取严重性值
// 返回值：
//   - int: 通过与操作从优先级中提取的严重性值（0-7）
func (m *Message) GetSeverity() int {
	return m.Priority & 0x07
}

// GetSyslogFormat 获取消息格式
// 返回值：
//   - SyslogFormat: 消息使用的Syslog格式（RFC3164或RFC5424）
func (m *Message) GetSyslogFormat() SyslogFormat {
	return m.SyslogFormat
}

// Bytes 返回消息的字节表示
// 返回值：
//   - []byte: 消息的字节数组表示
func (m *Message) Bytes() []byte {
	return []byte(m.Format())
}

// String 返回消息的字符串表示
// 返回值：
//   - string: 消息的字符串表示
func (m *Message) String() string {
	return m.Format()
}

// ParseFormat 解析格式字符串
// 参数：
//   - format: 要解析的格式字符串，支持"rfc3164"、"rfc5424"和"5424"（不区分大小写）
// 返回值：
//   - SyslogFormat: 解析后的Syslog格式，默认返回RFC3164格式
// 说明：
//   该函数将输入的格式字符串转换为对应的SyslogFormat类型
//   如果输入的格式不能识别，将默认返回RFC3164格式
func ParseFormat(format string) SyslogFormat {
	switch strings.ToLower(format) {
	case "rfc5424", "5424":
		return RFC5424 // 新格式
	default:
		return RFC3164 // 默认使用RFC3164格式
	}
}

// GetFacilityName 获取Facility名称
func GetFacilityName(facility int) string {
	facilities := map[int]string{
		0:  "kernel",
		1:  "user",
		2:  "mail",
		3:  "daemon",
		4:  "auth",
		5:  "syslog",
		6:  "lpr",
		7:  "news",
		8:  "uucp",
		9:  "cron",
		10: "authpriv",
		11: "ftp",
		16: "local0",
		17: "local1",
		18: "local2",
		19: "local3",
		20: "local4",
		21: "local5",
		22: "local6",
		23: "local7",
	}

	if name, ok := facilities[facility]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", facility)
}

// GetSeverityName 获取Severity名称
func GetSeverityName(severity int) string {
	severities := map[int]string{
		0: "emerg",
		1: "alert",
		2: "crit",
		3: "err",
		4: "warning",
		5: "notice",
		6: "info",
		7: "debug",
	}

	if name, ok := severities[severity]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", severity)
}
