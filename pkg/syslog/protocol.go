package syslog

import (
	"fmt"
	"strings"
	"time"
)

// SyslogFormat 定义Syslog格式类型
type SyslogFormat string

const (
	RFC3164 SyslogFormat = "rfc3164"
	RFC5424 SyslogFormat = "rfc5424"
)

// Message 表示一个Syslog消息
type Message struct {
	Priority  int       // 优先级 (Facility * 8 + Severity)
	Timestamp time.Time // 时间戳
	Hostname  string    // 主机名
	Tag       string    // 标签/程序名
	PID       string    // 进程ID
	Content   string    // 消息内容
	Format    SyslogFormat // 格式类型
}

// NewMessage 创建新的Syslog消息
func NewMessage(priority int, hostname, tag, content string, format SyslogFormat) *Message {
	return &Message{
		Priority:  priority,
		Timestamp: time.Now(),
		Hostname:  hostname,
		Tag:       tag,
		Content:   content,
		Format:    format,
	}
}

// Format 将消息格式化为指定的Syslog格式
func (m *Message) Format() string {
	switch m.Format {
	case RFC5424:
		return m.formatRFC5424()
	default:
		return m.formatRFC3164()
	}
}

// formatRFC3164 格式化为RFC3164格式
// 格式: <Priority>Timestamp Hostname Tag[PID]: Content
func (m *Message) formatRFC3164() string {
	// RFC3164时间戳格式: Jan 02 15:04:05
	timestamp := m.Timestamp.Format("Jan 02 15:04:05")
	
	// 构建标签部分
	var tagPart string
	if m.PID != "" {
		tagPart = fmt.Sprintf("%s[%s]", m.Tag, m.PID)
	} else {
		tagPart = m.Tag
	}
	
	// 如果标签为空，使用默认标签
	if tagPart == "" {
		tagPart = "syslog_sender"
	}
	
	return fmt.Sprintf("<%d>%s %s %s: %s",
		m.Priority,
		timestamp,
		m.Hostname,
		tagPart,
		m.Content)
}

// formatRFC5424 格式化为RFC5424格式
// 格式: <Priority>Version Timestamp Hostname App-Name ProcID MsgID Structured-Data Msg
func (m *Message) formatRFC5424() string {
	// RFC5424时间戳格式: 2006-01-02T15:04:05.000Z
	timestamp := m.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z")
	
	// 处理各个字段，空值用 "-" 表示
	hostname := m.Hostname
	if hostname == "" {
		hostname = "-"
	}
	
	appName := m.Tag
	if appName == "" {
		appName = "syslog_sender"
	}
	
	procID := m.PID
	if procID == "" {
		procID = "-"
	}
	
	msgID := "-"           // 消息ID，通常为空
	structuredData := "-"  // 结构化数据，暂时为空
	
	return fmt.Sprintf("<%d>1 %s %s %s %s %s %s %s",
		m.Priority,
		timestamp,
		hostname,
		appName,
		procID,
		msgID,
		structuredData,
		m.Content)
}

// SetTimestamp 设置自定义时间戳
func (m *Message) SetTimestamp(t time.Time) {
	m.Timestamp = t
}

// SetPID 设置进程ID
func (m *Message) SetPID(pid string) {
	m.PID = pid
}

// Bytes 返回消息的字节表示
func (m *Message) Bytes() []byte {
	return []byte(m.Format())
}

// String 返回消息的字符串表示
func (m *Message) String() string {
	return m.Format()
}

// ParseFormat 解析格式字符串
func ParseFormat(format string) SyslogFormat {
	switch strings.ToLower(format) {
	case "rfc5424", "5424":
		return RFC5424
	default:
		return RFC3164
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