package template

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// globalCounter 用于生成连续IP地址的全局计数器
var globalCounter int64

// VariableParser 变量解析器
type VariableParser struct {
	random          *rand.Rand
	customVariables map[string]CustomVariable
	verbose bool
}

// NewVariableParser 创建新的变量解析器
func NewVariableParser(verbose bool) *VariableParser {
	return &VariableParser{
		customVariables: make(map[string]CustomVariable),
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		verbose: verbose,
	}
}

// RegisterCustomVariable 注册自定义变量
func (p *VariableParser) RegisterCustomVariable(name string, variable CustomVariable) error {
	// 验证变量配置
	switch variable.Type {
	case "random_choice":
		if len(variable.Values) == 0 {
			return fmt.Errorf("random_choice类型变量必须提供values列表")
		}
	case "random_int":
		if variable.Min >= variable.Max {
			return fmt.Errorf("random_int类型变量的min必须小于max")
		}
	case "random_string":
		if variable.Length <= 0 {
			return fmt.Errorf("random_string类型变量的length必须大于0")
		}
	default:
		return fmt.Errorf("不支持的变量类型: %s", variable.Type)
	}

	// 存储变量配置
	name = strings.ToUpper(name)
	p.customVariables[name] = variable
	if p.verbose {
		fmt.Printf("注册自定义变量: %s, 类型: %s\n", name, variable.Type)
	}
	return nil
}

// newRandom 创建新的随机数生成器
func (p *VariableParser) newRandom() *rand.Rand {
	// 使用crypto/rand生成真随机数作为种子
	seed := make([]byte, 8)
	_, err := cryptorand.Read(seed)
	if err == nil {
		// 直接使用crypto/rand生成的随机字节作为种子
		seedInt := int64(binary.LittleEndian.Uint64(seed))
		return rand.New(rand.NewSource(seedInt))
	}

	// 如果获取随机种子失败，回退到使用时间戳和纳秒级随机性
	now := time.Now().UnixNano()
	seedInt := now ^ atomic.AddInt64(&globalCounter, 1) ^ rand.Int63()
	return rand.New(rand.NewSource(seedInt))
}

// Parse 解析变量表达式
func (p *VariableParser) Parse(expr string) (string, error) {
	// 分割变量名和参数
	parts := strings.SplitN(expr, ":", 2)
	varName := strings.TrimSpace(parts[0])
	varName = strings.ToUpper(varName)
	var params string
	if len(parts) > 1 {
		params = strings.TrimSpace(parts[1])
	}

	// 先检查是否是自定义变量
	if variable, ok := p.customVariables[varName]; ok {
		switch variable.Type {
		case "random_choice":
			return variable.Values[p.random.Intn(len(variable.Values))], nil
		case "random_int":
			return fmt.Sprintf("%d", p.random.Intn(variable.Max-variable.Min)+variable.Min), nil
		case "random_string":
			return p.generateRandomString(fmt.Sprintf("%d", variable.Length))
		default:
			return "", fmt.Errorf("不支持的变量类型: %s", variable.Type)
		}
	}

	// 根据变量类型生成值
	switch varName {
	case "RANDOM_STRING":
		return p.generateRandomString(params)
	case "RANDOM_INT":
		return p.generateRandomInt(params)
	case "ENUM":
		return p.generateEnum(params)
	case "MAC":
		return p.generateMAC()
	case "RANGE_IP":
		// 自动识别IPv6地址
		if strings.Contains(params, ":") {
			return p.generateRangeIPv6(params)
		}
		// IPv4地址
		return p.generateRangeIP(params)
	case "RANDOM_IP", "RANDOM_IPV4":
		if params == "internal" {
			return p.generateInternalIP()
		} else if params == "external" {
			return p.generateExternalIP()
		}
		return p.generateRandomIP(params)
	case "RANDOM_IPV6":
		return p.generateRandomIPv6(params)
	case "PROTOCOL":
		return p.generateProtocol()
	case "HTTP_METHOD":
		return p.generateHTTPMethod()
	case "HTTP_STATUS":
		return p.generateHTTPStatus()
	case "EMAIL":
		return p.generateEmail()
	case "DOMAIN":
		return p.generateDomain()
	case "URL_PATH":
		return p.generateURLPath()
	default:
		return "", fmt.Errorf("unsupported variable: %s", varName)
	}
}

// generateCustomVariable 生成自定义变量值
func (p *VariableParser) generateCustomVariable(name string) (string, error) {
	variable, ok := p.customVariables[name]
	if !ok {
		return "", fmt.Errorf("未找到自定义变量: %s", name)
	}

	switch variable.Type {
	case "random_choice":
		return variable.Values[p.random.Intn(len(variable.Values))], nil
	case "random_int":
		return fmt.Sprintf("%d", p.random.Intn(variable.Max-variable.Min)+variable.Min), nil
	case "random_string":
		return p.generateRandomString(fmt.Sprintf("%d", variable.Length))
	default:
		return "", fmt.Errorf("不支持的变量类型: %s", variable.Type)
	}
}

// generateRandomString 生成随机字符串
func (p *VariableParser) generateRandomString(params string) (string, error) {
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANDOM_STRING")
	}

	// 创建新的随机数生成器
	random := p.newRandom()

	// 解析选项和权重
	options := strings.Split(params, ",")
	weights := make([]int, len(options))
	totalWeight := 0

	// 处理每个选项
	for i, opt := range options {
		// 检查是否有权重
		parts := strings.Split(strings.TrimSpace(opt), ":")
		options[i] = parts[0]
		weight := 1

		// 如果指定了权重，解析权重值
		if len(parts) > 1 {
			w, err := strconv.Atoi(parts[1])
			if err == nil && w > 0 {
				weight = w
			}
		}

		weights[i] = weight
		totalWeight += weight
	}

	// 根据权重随机选择
	r := random.Intn(totalWeight)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return options[i], nil
		}
	}

	return options[len(options)-1], nil
}

// generateRandomInt 生成随机整数
func (p *VariableParser) generateRandomInt(params string) (string, error) {
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANDOM_INT")
	}

	// 创建新的随机数生成器
	random := p.newRandom()

	// 解析范围
	parts := strings.Split(params, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid range format for RANDOM_INT, expected min-max")
	}

	// 解析最小值和最大值
	min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", fmt.Errorf("invalid minimum value: %s", parts[0])
	}

	max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", fmt.Errorf("invalid maximum value: %s", parts[1])
	}

	if min >= max {
		return "", fmt.Errorf("minimum value must be less than maximum value")
	}

	// 生成随机数
	result := random.Intn(max-min+1) + min
	return strconv.Itoa(result), nil
}

// generateEnum 生成枚举值
func (p *VariableParser) generateEnum(params string) (string, error) {
	if params == "" {
		return "", fmt.Errorf("missing parameters for ENUM")
	}

	// 创建新的随机数生成器
	random := p.newRandom()

	// 分割选项
	options := strings.Split(params, ",")
	for i := range options {
		options[i] = strings.TrimSpace(options[i])
	}

	// 随机选择一个选项
	return options[random.Intn(len(options))], nil
}

// generateMAC 生成MAC地址
func (p *VariableParser) generateMAC() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	mac := make([]byte, 6)
	random.Read(mac)
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
}

// generateRandomIP 生成随机IP地址
func (p *VariableParser) generateRandomIP(params string) (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	if params == "" {
		// 生成任意IP地址
		return fmt.Sprintf("%d.%d.%d.%d",
			random.Intn(256),
			random.Intn(256),
			random.Intn(256),
			random.Intn(256)), nil
	}

	// 解析IP范围
	parts := strings.Split(params, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid IP range format, expected start,end")
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// 这里简化处理，仅支持最后一段可变
	startParts := strings.Split(start, ".")
	endParts := strings.Split(end, ".")

	if len(startParts) != 4 || len(endParts) != 4 {
		return "", fmt.Errorf("invalid IP address format")
	}

	// 确保前三段相同
	for i := 0; i < 3; i++ {
		if startParts[i] != endParts[i] {
			return "", fmt.Errorf("only last octet can be different in IP range")
		}
	}

	// 解析最后一段的范围
	startNum, err := strconv.Atoi(startParts[3])
	if err != nil {
		return "", fmt.Errorf("invalid start IP: %s", start)
	}

	endNum, err := strconv.Atoi(endParts[3])
	if err != nil {
		return "", fmt.Errorf("invalid end IP: %s", end)
	}

	if startNum >= endNum {
		return "", fmt.Errorf("start IP must be less than end IP")
	}

	// 生成随机IP
	lastNum := random.Intn(endNum-startNum+1) + startNum
	return fmt.Sprintf("%s.%s.%s.%d",
		startParts[0], startParts[1], startParts[2], lastNum), nil
}

// generateInternalIP 生成内网IP地址
func (p *VariableParser) generateInternalIP() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	// 随机选择一个内网IP范围
	switch random.Intn(3) {
	case 0: // 192.168.0.0/16
		return fmt.Sprintf("192.168.%d.%d",
			random.Intn(256),    // 第三段: 0-255
			random.Intn(254)+1), // 第四段: 1-254
			nil
	case 1: // 172.16.0.0/12
		return fmt.Sprintf("172.%d.%d.%d",
			16+random.Intn(16), // 第二段: 16-31
			random.Intn(256),    // 第三段: 0-255
			random.Intn(254)+1), // 第四段: 1-254
			nil
	default: // 10.0.0.0/8
		return fmt.Sprintf("10.%d.%d.%d",
			random.Intn(256),    // 第二段: 0-255
			random.Intn(256),    // 第三段: 0-255
			random.Intn(254)+1), // 第四段: 1-254
			nil
	}
}

// generateExternalIP 生成外网IP地址
func (p *VariableParser) generateExternalIP() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	for {
		// 生成第一段，避免0和127（保留地址）
		a := random.Intn(223) + 1
		if a == 127 {
			continue
		}

		// 生成剩余段
		b := random.Intn(256)
		c := random.Intn(256)
		d := random.Intn(254) + 1

		// 避免内网地址段
		if (a == 10) || // 10.0.0.0/8
			(a == 172 && b >= 16 && b <= 31) || // 172.16.0.0/12
			(a == 192 && b == 168) { // 192.168.0.0/16
			continue
		}

		return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d), nil
	}
}

// generateRangeIP 生成指定范围内的IPv4地址
func (p *VariableParser) generateRangeIP(params string) (string, error) {
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANGE_IP")
	}

	// 支持两种格式：
	// 1. 起始IP-结束IP，如：192.168.1.1-192.168.1.100
	// 2. CIDR格式，如：192.168.1.0/24
	if strings.Contains(params, "/") {
		// CIDR格式
		return p.generateIPFromCIDR(params)
	}

	// IP范围格式
	parts := strings.Split(params, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid IP range format, expected start-end or CIDR notation")
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// 解析起始IP
	startIP := strings.Split(start, ".")
	endIP := strings.Split(end, ".")

	if len(startIP) != 4 || len(endIP) != 4 {
		return "", fmt.Errorf("invalid IP address format")
	}

	// 将IP地址转换为32位整数
	startNum := 0
	endNum := 0
	for i := 0; i < 4; i++ {
		s, err := strconv.Atoi(startIP[i])
		if err != nil || s < 0 || s > 255 {
			return "", fmt.Errorf("invalid start IP: %s", start)
		}
		e, err := strconv.Atoi(endIP[i])
		if err != nil || e < 0 || e > 255 {
			return "", fmt.Errorf("invalid end IP: %s", end)
		}
		startNum = startNum*256 + s
		endNum = endNum*256 + e
	}

	if startNum >= endNum {
		return "", fmt.Errorf("start IP must be less than end IP")
	}

	// 获取当前计数器值并递增
	counter := atomic.AddInt64(&globalCounter, 1) - 1
	// 计算总地址数
	totalIPs := endNum - startNum + 1
	// 使用计数器值对总地址数取模，实现连续生成
	num := startNum + int(counter%int64(totalIPs))

	return fmt.Sprintf("%d.%d.%d.%d",
		(num>>24)&255,
		(num>>16)&255,
		(num>>8)&255,
		num&255), nil
}

// generateIPFromCIDR 从CIDR格式生成随机IP
func (p *VariableParser) generateIPFromCIDR(cidr string) (string, error) {
	// 分割IP和掩码
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid CIDR format")
	}

	// 解析IP地址
	ip := strings.Split(parts[0], ".")
	if len(ip) != 4 {
		return "", fmt.Errorf("invalid IP address in CIDR")
	}

	// 解析掩码
	mask, err := strconv.Atoi(parts[1])
	if err != nil || mask < 0 || mask > 32 {
		return "", fmt.Errorf("invalid network mask in CIDR")
	}

	// 计算网络地址
	baseIP := 0
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(ip[i])
		if err != nil || n < 0 || n > 255 {
			return "", fmt.Errorf("invalid IP address: %s", parts[0])
		}
		baseIP = baseIP*256 + n
	}

	// 计算可用IP范围
	hostBits := 32 - mask
	if hostBits <= 0 {
		return "", fmt.Errorf("invalid network mask: /%d", mask)
	}

	// 获取当前计数器值并递增
	counter := atomic.AddInt64(&globalCounter, 1) - 1
	maskBits := uint32(0xFFFFFFFF) << uint(hostBits)
	network := uint32(baseIP) & maskBits
	hostMax := uint32(1<<uint(hostBits)) - 1

	// 避免网络地址和广播地址
	if hostBits > 1 {
		// 使用计数器值对可用主机数取模，实现连续生成
		hostNum := uint32(counter%int64(hostMax-1)) + 1
		ip := network | hostNum

		return fmt.Sprintf("%d.%d.%d.%d",
			(ip>>24)&255,
			(ip>>16)&255,
			(ip>>8)&255,
			ip&255), nil
	}

	return "", fmt.Errorf("network mask is too restrictive: /%d", mask)
}

// generateRangeIPv6 生成指定范围内的IPv6地址
func (p *VariableParser) generateRangeIPv6(params string) (string, error) {
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANGE_IP v6")
	}

	// 支持CIDR格式，如：2001:db8::/32
	if strings.Contains(params, "/") {
		// 分割IP和掩码
		parts := strings.Split(params, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid IPv6 CIDR format")
		}

		// 解析掩码
		mask, err := strconv.Atoi(parts[1])
		if err != nil || mask < 0 || mask > 128 {
			return "", fmt.Errorf("invalid IPv6 network mask")
		}

		// 解析基础IPv6地址
		base := strings.Split(parts[0], ":")
		if len(base) > 8 {
			return "", fmt.Errorf("invalid IPv6 address format")
		}

		// 处理压缩符号 ::
		for i, part := range base {
			if part == "" {
				// 计算需要插入的0组数量
				zeros := 8 - (len(base) - 1)
				newBase := make([]string, 8)
				copy(newBase[:i], base[:i])
				for j := 0; j < zeros; j++ {
					newBase[i+j] = "0"
				}
				if i+1 < len(base) {
					copy(newBase[i+zeros:], base[i+1:])
				}
				base = newBase
				break
			}
		}

		// 获取当前计数器值并递增
		counter := atomic.AddInt64(&globalCounter, 1) - 1
		result := make([]string, 8)

		// 保持网络部分不变
		networkParts := mask / 16
		for i := 0; i < networkParts; i++ {
			if i < len(base) {
				result[i] = base[i]
			} else {
				result[i] = "0"
			}
		}

		// 生成主机部分，使用计数器值实现顺序生成
		for i := networkParts; i < 8; i++ {
			// 每组16位，使用计数器的不同部分
			shift := uint((7-i) * 16)
			value := (counter >> shift) & 0xFFFF
			result[i] = fmt.Sprintf("%04x", value)
		}

		return strings.Join(result, ":"), nil
	}

	return "", fmt.Errorf("only CIDR notation is supported for IPv6 range")
}

// generateEmail 生成邮箱地址
func (p *VariableParser) generateEmail() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	// 常见邮箱域名列表
	domains := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "protonmail.com", "icloud.com", "juminfo.com"}
	// 用户名字符集
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"

	// 生成随机用户名长度(6-12字符)
	usernameLen := random.Intn(7) + 6
	username := make([]byte, usernameLen)
	for i := range username {
		username[i] = charset[random.Intn(len(charset))]
	}

	// 随机选择域名
	domain := domains[random.Intn(len(domains))]

	return fmt.Sprintf("%s@%s", string(username), domain), nil
}

// generateDomain 生成域名
func (p *VariableParser) generateDomain() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	// 常见域名前缀
	prefixes := []string{
		"cloud", "api", "dev", "test", "stage", "prod", "admin", "portal",
		"app", "web", "mobile", "cdn", "static", "media", "auth", "login",
		"mail", "smtp", "ftp", "git", "wiki", "docs", "support", "help",
	}

	// 常见域名中间部分
	middles := []string{
		"service", "platform", "system", "network", "security", "server",
		"data", "storage", "compute", "analytics", "monitor", "backup",
		"proxy", "gateway", "cluster", "node", "host", "client",
	}

	// 常见公司或组织名称
	companies := []string{
		"google", "amazon", "microsoft", "apple", "meta", "oracle", "ibm",
		"cisco", "intel", "amd", "nvidia", "dell", "hp", "lenovo", "huawei",
		"github", "gitlab", "bitbucket", "docker", "kubernetes", "linux",
	}

	// 顶级域名列表
	tlds := []string{
		// 通用顶级域名
		"com", "org", "net", "edu", "gov", "mil", "int",
		// 新通用顶级域名
		"io", "cloud", "tech", "dev", "app", "ai", "co", "me",
		// 国家和地区顶级域名
		"cn", "us", "uk", "eu", "de", "fr", "jp", "kr",
		// 安全相关顶级域名
		"security", "protection", "defense", "secure", "trust",
	}

	// 生成域名的方式
	domainType := random.Intn(4)
	var domain string

	switch domainType {
	case 0:
		// 公司域名: company.tld
		domain = fmt.Sprintf("%s.%s",
			companies[random.Intn(len(companies))],
			tlds[random.Intn(len(tlds))])
	case 1:
		// 服务域名: prefix.company.tld
		domain = fmt.Sprintf("%s.%s.%s",
			prefixes[random.Intn(len(prefixes))],
			companies[random.Intn(len(companies))],
			tlds[random.Intn(len(tlds))])
	case 2:
		// 功能域名: prefix-middle.tld
		domain = fmt.Sprintf("%s-%s.%s",
			prefixes[random.Intn(len(prefixes))],
			middles[random.Intn(len(middles))],
			tlds[random.Intn(len(tlds))])
	default:
		// 随机字符域名
		charset := "abcdefghijklmnopqrstuvwxyz0123456789"
		domainLen := random.Intn(6) + 5
		domainBytes := make([]byte, domainLen)
		for i := range domainBytes {
			domainBytes[i] = charset[random.Intn(len(charset))]
		}
		domain = fmt.Sprintf("%s.%s", string(domainBytes), tlds[random.Intn(len(tlds))])
	}

	return domain, nil
}

// generateURLPath 生成URL路径
func (p *VariableParser) generateURLPath() (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	// 常见路径段
	pathSegments := []string{
		"api", "v1", "v2", "admin", "user", "profile", "settings",
		"login", "logout", "register", "dashboard", "public", "private",
		"upload", "download", "images", "files", "docs", "help",
	}

	// 特殊字符和编码字符
	specialChars := []string{
		"%20", "%2F", "%3F", "%3D", "%26", "%25", "%2B", "%23",
		"!", "@", "$", "^", "&", "(", ")", "[", "]", "{", "}",
	}

	// 查询参数名
	queryParams := []string{
		"id", "user", "token", "page", "size", "sort", "order",
		"type", "format", "lang", "version", "timestamp", "callback",
	}

	// 查询参数值
	queryValues := []string{
		"asc", "desc", "true", "false", "json", "xml", "html",
		"en-US", "zh-CN", "1.0.0", "latest", "admin", "guest",
	}

	// 生成2-4个路径段
	segmentCount := random.Intn(3) + 2
	path := make([]string, segmentCount)

	// 随机选择路径段并可能添加特殊字符
	for i := range path {
		pathSegment := pathSegments[random.Intn(len(pathSegments))]
		// 20%概率添加特殊字符
		if random.Float64() < 0.2 {
			pathSegment += specialChars[random.Intn(len(specialChars))]
		}
		path[i] = pathSegment
	}

	// 基础路径
	url := "/" + strings.Join(path, "/")

	// 50%概率添加查询参数
	if random.Float64() < 0.5 {
		// 添加1-3个查询参数
		paramCount := random.Intn(3) + 1
		params := make([]string, paramCount)

		for i := range params {
			paramName := queryParams[random.Intn(len(queryParams))]
			paramValue := queryValues[random.Intn(len(queryValues))]

			// 30%概率对参数值进行URL编码
			if random.Float64() < 0.3 {
				paramValue = specialChars[random.Intn(len(specialChars))] + paramValue
			}

			params[i] = paramName + "=" + paramValue
		}

		url += "?" + strings.Join(params, "&")
	}

	return url, nil
}

// generateProtocol 生成网络协议名称
func (p *VariableParser) generateProtocol() (string, error) {
	// 常见网络协议列表
	protocols := []string{
		"HTTP", "HTTPS", "FTP", "SFTP", "SSH", "TELNET",
		"SMTP", "POP3", "IMAP", "DNS", "DHCP", "LDAP",
		"SMB", "NFS", "SNMP", "MQTT", "AMQP", "RTSP",
		"RDP", "VNC", "IRC", "XMPP", "SIP", "RADIUS",
	}

	// 创建新的随机数生成器
	random := p.newRandom()
	return protocols[random.Intn(len(protocols))], nil
}

// generateHTTPMethod 生成HTTP请求方法
func (p *VariableParser) generateHTTPMethod() (string, error) {
	// HTTP请求方法列表
	methods := []string{
		"GET", "POST", "PUT", "DELETE", "HEAD",
		"OPTIONS", "PATCH", "TRACE", "CONNECT",
	}

	// 创建新的随机数生成器
	random := p.newRandom()
	return methods[random.Intn(len(methods))], nil
}

// generateHTTPStatus 生成HTTP状态码
func (p *VariableParser) generateHTTPStatus() (string, error) {
	// HTTP状态码列表
	statuses := []struct {
		code int
		desc string
	}{
		{200, "OK"}, {201, "Created"}, {202, "Accepted"},
		{204, "No Content"}, {301, "Moved Permanently"},
		{302, "Found"}, {304, "Not Modified"},
		{400, "Bad Request"}, {401, "Unauthorized"},
		{403, "Forbidden"}, {404, "Not Found"},
		{405, "Method Not Allowed"}, {408, "Request Timeout"},
		{429, "Too Many Requests"}, {500, "Internal Server Error"},
		{501, "Not Implemented"}, {502, "Bad Gateway"},
		{503, "Service Unavailable"}, {504, "Gateway Timeout"},
	}

	// 创建新的随机数生成器
	random := p.newRandom()
	status := statuses[random.Intn(len(statuses))]
	return fmt.Sprintf("%d %s", status.code, status.desc), nil
}

// generateRandomIPv6 生成随机IPv6地址
func (p *VariableParser) generateRandomIPv6(params string) (string, error) {
	// 创建新的随机数生成器
	random := p.newRandom()

	// 根据参数生成不同类型的IPv6地址
	switch params {
	case "internal": // 生成内网IPv6地址 (fd00::/8)
		groups := make([]string, 8)
		groups[0] = "fd00"
		for i := 1; i < 8; i++ {
			groups[i] = fmt.Sprintf("%04x", random.Intn(65536))
		}
		return strings.Join(groups, ":"), nil

	case "external": // 生成外网IPv6地址 (2000::/3)
		groups := make([]string, 8)
		groups[0] = fmt.Sprintf("2%03x", random.Intn(0x1000))
		for i := 1; i < 8; i++ {
			groups[i] = fmt.Sprintf("%04x", random.Intn(65536))
		}
		return strings.Join(groups, ":"), nil

	case "compressed": // 生成压缩格式的IPv6地址（包含::）
		groups := make([]string, 8)
		// 生成8组数，确保至少有2组连续的0
		compressStart := random.Intn(5) + 1 // 不压缩第一组
		compressLength := random.Intn(2) + 2 // 压缩2-3组连续的0
		for i := range groups {
			if i >= compressStart && i < compressStart+compressLength {
				groups[i] = "0000"
			} else {
				// 确保生成4位十六进制数
				groups[i] = fmt.Sprintf("%04x", random.Intn(65536))
			}
		}

		// 找到最长的连续0序列
		maxZeroStart := -1
		maxZeroLength := 0
		currentZeroStart := -1
		currentZeroLength := 0

		for i, group := range groups {
			if group == "0000" {
				if currentZeroStart == -1 {
					currentZeroStart = i
				}
				currentZeroLength++
				if currentZeroLength > maxZeroLength {
					maxZeroStart = currentZeroStart
					maxZeroLength = currentZeroLength
				}
			} else {
				currentZeroStart = -1
				currentZeroLength = 0
			}
		}

		// 构建压缩格式的地址
		var parts []string
		for i := 0; i < len(groups); i++ {
			if i == maxZeroStart {
				parts = append(parts, "")
				if maxZeroStart == 0 {
					parts = append(parts, "")
				}
				i += maxZeroLength - 1
			} else {
				// 移除前导0但保留至少一位数字
				val := strings.TrimLeft(groups[i], "0")
				if val == "" {
					val = "0"
				}
				parts = append(parts, val)
			}
		}

		// 处理末尾压缩的情况
		if maxZeroStart+maxZeroLength == len(groups) {
			parts = append(parts, "")
		}

		return strings.Join(parts, ":"), nil

	default: // 生成标准格式的IPv6地址
		groups := make([]string, 8)
		for i := range groups {
			groups[i] = fmt.Sprintf("%04x", random.Intn(65536))
		}
		return strings.Join(groups, ":"), nil
	}
}
