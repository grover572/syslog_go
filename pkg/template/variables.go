package template

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// initGenerators 初始化内置变量生成器
func (e *Engine) initGenerators() {
	e.generators = map[string]func() string{
		// 网络相关
		"random_ip":      e.genRandomIP,
		"internal_ip":    e.genInternalIP,
		"external_ip":    e.genExternalIP,
		"random_port":    e.genRandomPort,
		"internal_port":  e.genInternalPort,
		"random_mac":     e.genRandomMAC,

		// 时间相关
		"timestamp":        e.genTimestamp,
		"timestamp_apache": e.genTimestampApache,
		"timestamp_iso":    e.genTimestampISO,
		"timestamp_unix":   e.genTimestampUnix,

		// 用户相关
		"username":     e.genUsername,
		"random_user":  e.genRandomUser,
		"hostname":     e.genHostname,
		"random_host":  e.genRandomHost,

		// 系统相关
		"pid":         e.genPID,
		"random_pid":  e.genRandomPID,
		"conn_id":     e.genConnID,
		"session_id":  e.genSessionID,
		"hex_id":      e.genHexID,

		// HTTP相关
		"http_method":   e.genHTTPMethod,
		"http_status":   e.genHTTPStatus,
		"api_status":    e.genAPIStatus,
		"url_path":      e.genURLPath,
		"api_endpoint":  e.genAPIEndpoint,
		"static_file":   e.genStaticFile,
		"user_agent":    e.genUserAgent,
		"api_client":    e.genAPIClient,
		"referer":       e.genReferer,

		// 数据相关
		"response_size": e.genResponseSize,
		"file_size":     e.genFileSize,
		"bytes":         e.genBytes,
		"duration":      e.genDuration,
		"hit_count":     e.genHitCount,

		// 安全相关
		"protocol":      e.genProtocol,
		"action":        e.genAction,
		"random_action": e.genRandomAction,
		"severity":      e.genSeverity,
		"alert_type":    e.genAlertType,

		// 系统监控
		"cpu":           e.genCPU,
		"memory":        e.genMemory,
		"total_memory":  e.genTotalMemory,
		"disk":          e.genDisk,
		"mount":         e.genMount,
		"process":       e.genProcess,
		"score":         e.genScore,
		"uptime":        e.genUptime,
	}
}

// 网络相关生成器
func (e *Engine) genRandomIP() string {
	// 生成常见的私有IP地址
	networks := []string{
		"192.168.%d.%d",
		"10.%d.%d.%d",
		"172.16.%d.%d",
	}
	network := networks[e.random.Intn(len(networks))]
	
	switch network {
	case "192.168.%d.%d":
		return fmt.Sprintf(network, e.random.Intn(256), e.random.Intn(254)+1)
	case "10.%d.%d.%d":
		return fmt.Sprintf(network, e.random.Intn(256), e.random.Intn(256), e.random.Intn(254)+1)
	default:
		return fmt.Sprintf(network, e.random.Intn(256), e.random.Intn(254)+1)
	}
}

func (e *Engine) genInternalIP() string {
	return fmt.Sprintf("192.168.100.%d", e.random.Intn(50)+1)
}

func (e *Engine) genExternalIP() string {
	// 生成公网IP（避免私有地址段）
	for {
		a := e.random.Intn(223) + 1 // 1-223
		b := e.random.Intn(256)
		c := e.random.Intn(256)
		d := e.random.Intn(254) + 1
		
		// 避免私有地址段
		if (a == 10) || (a == 172 && b >= 16 && b <= 31) || (a == 192 && b == 168) {
			continue
		}
		
		return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
	}
}

func (e *Engine) genRandomPort() string {
	return fmt.Sprintf("%d", e.random.Intn(65535-1024)+1024)
}

func (e *Engine) genInternalPort() string {
	ports := []string{"80", "443", "22", "3389", "3306", "5432", "8080", "8443"}
	return ports[e.random.Intn(len(ports))]
}

func (e *Engine) genRandomMAC() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		e.random.Intn(256), e.random.Intn(256), e.random.Intn(256),
		e.random.Intn(256), e.random.Intn(256), e.random.Intn(256))
}

// 时间相关生成器
func (e *Engine) genTimestamp() string {
	return time.Now().Format("Jan 02 15:04:05")
}

func (e *Engine) genTimestampApache() string {
	return time.Now().Format("02/Jan/2006:15:04:05 -0700")
}

func (e *Engine) genTimestampISO() string {
	return time.Now().Format("2006-01-02T15:04:05.000Z")
}

func (e *Engine) genTimestampUnix() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

// 用户相关生成器
func (e *Engine) genUsername() string {
	usernames := []string{"admin", "user1", "test", "guest", "operator", "monitor", "service", "backup"}
	return usernames[e.random.Intn(len(usernames))]
}

func (e *Engine) genRandomUser() string {
	return e.genUsername()
}

func (e *Engine) genHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return e.genRandomHost()
	}
	return hostname
}

func (e *Engine) genRandomHost() string {
	hosts := []string{"web-server-01", "db-server-02", "app-server-03", "proxy-01", "firewall-01", "switch-01"}
	return hosts[e.random.Intn(len(hosts))]
}

// 系统相关生成器
func (e *Engine) genPID() string {
	return fmt.Sprintf("%d", e.random.Intn(99999-1000)+1000)
}

func (e *Engine) genRandomPID() string {
	return e.genPID()
}

func (e *Engine) genConnID() string {
	return fmt.Sprintf("%d", e.random.Intn(999999-100000)+100000)
}

func (e *Engine) genSessionID() string {
	return fmt.Sprintf("%x", e.random.Uint64())
}

func (e *Engine) genHexID() string {
	return fmt.Sprintf("%08x", e.random.Uint32())
}

// HTTP相关生成器
func (e *Engine) genHTTPMethod() string {
	// 加权随机选择
	rand := e.random.Intn(100)
	switch {
	case rand < 70:
		return "GET"
	case rand < 90:
		return "POST"
	case rand < 95:
		return "PUT"
	case rand < 98:
		return "DELETE"
	default:
		return "HEAD"
	}
}

func (e *Engine) genHTTPStatus() string {
	// 加权随机选择
	rand := e.random.Intn(100)
	switch {
	case rand < 80:
		return "200"
	case rand < 90:
		return "404"
	case rand < 95:
		return "500"
	case rand < 98:
		return "403"
	default:
		return "302"
	}
}

func (e *Engine) genAPIStatus() string {
	statuses := []string{"200", "201", "400", "401", "403", "404", "500", "502"}
	return statuses[e.random.Intn(len(statuses))]
}

func (e *Engine) genURLPath() string {
	paths := []string{"/index.html", "/api/users", "/login", "/dashboard", "/admin", "/api/data", "/health"}
	return paths[e.random.Intn(len(paths))]
}

func (e *Engine) genAPIEndpoint() string {
	endpoints := []string{"/api/v1/users", "/api/v1/auth", "/api/v1/data", "/api/v2/reports", "/api/v1/config"}
	return endpoints[e.random.Intn(len(endpoints))]
}

func (e *Engine) genStaticFile() string {
	files := []string{"/css/style.css", "/js/app.js", "/images/logo.png", "/favicon.ico", "/robots.txt"}
	return files[e.random.Intn(len(files))]
}

func (e *Engine) genUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		"curl/7.68.0",
		"PostmanRuntime/7.28.0",
	}
	return agents[e.random.Intn(len(agents))]
}

func (e *Engine) genAPIClient() string {
	clients := []string{"curl/7.68.0", "PostmanRuntime/7.28.0", "Python-requests/2.25.1", "Go-http-client/1.1"}
	return clients[e.random.Intn(len(clients))]
}

func (e *Engine) genReferer() string {
	refs := []string{
		"https://www.google.com/",
		"https://www.baidu.com/",
		"https://github.com/",
		"-",
	}
	return refs[e.random.Intn(len(refs))]
}

// 数据相关生成器
func (e *Engine) genResponseSize() string {
	return fmt.Sprintf("%d", e.random.Intn(10000)+100)
}

func (e *Engine) genFileSize() string {
	return fmt.Sprintf("%d", e.random.Intn(1000000)+1000)
}

func (e *Engine) genBytes() string {
	return fmt.Sprintf("%d", e.random.Intn(1000000)+1000)
}

func (e *Engine) genDuration() string {
	return fmt.Sprintf("%d:%02d:%02d", e.random.Intn(24), e.random.Intn(60), e.random.Intn(60))
}

func (e *Engine) genHitCount() string {
	return fmt.Sprintf("%d", e.random.Intn(1000)+1)
}

// 安全相关生成器
func (e *Engine) genProtocol() string {
	protocols := []string{"TCP", "UDP", "ICMP", "HTTP", "HTTPS", "SSH", "FTP"}
	return protocols[e.random.Intn(len(protocols))]
}

func (e *Engine) genAction() string {
	actions := []string{"login", "logout", "access", "modify", "delete", "create", "view", "download"}
	return actions[e.random.Intn(len(actions))]
}

func (e *Engine) genRandomAction() string {
	return e.genAction()
}

func (e *Engine) genSeverity() string {
	severities := []string{"low", "medium", "high", "critical"}
	return severities[e.random.Intn(len(severities))]
}

func (e *Engine) genAlertType() string {
	types := []string{"intrusion", "malware", "policy_violation", "anomaly", "brute_force"}
	return types[e.random.Intn(len(types))]
}

// 系统监控生成器
func (e *Engine) genCPU() string {
	return fmt.Sprintf("%d", e.random.Intn(100))
}

func (e *Engine) genMemory() string {
	return fmt.Sprintf("%d", e.random.Intn(8192)+1024)
}

func (e *Engine) genTotalMemory() string {
	return fmt.Sprintf("%d", 8192+e.random.Intn(8192))
}

func (e *Engine) genDisk() string {
	return fmt.Sprintf("%d", e.random.Intn(100))
}

func (e *Engine) genMount() string {
	mounts := []string{"/", "/home", "/var", "/tmp", "/opt", "C:\\", "D:\\"}
	return mounts[e.random.Intn(len(mounts))]
}

func (e *Engine) genProcess() string {
	processes := []string{"apache2", "nginx", "mysql", "postgres", "redis", "docker", "java", "python"}
	return processes[e.random.Intn(len(processes))]
}

func (e *Engine) genScore() string {
	return fmt.Sprintf("%d", e.random.Intn(1000))
}

func (e *Engine) genUptime() string {
	return fmt.Sprintf("%d.%06d", e.random.Intn(1000000), e.random.Intn(1000000))
}