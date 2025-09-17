// Package template 提供了一个灵活的模板引擎实现，支持变量解析和自定义变量处理。
// 该包主要用于生成动态的Syslog消息内容，通过预定义和自定义变量实现数据的随机化和定制化。
package template

import (
	// crypto/rand 用于生成加密安全的随机数
	cryptorand "crypto/rand"
	// encoding/binary 用于字节序列的二进制转换
	"encoding/binary"
	// fmt 用于格式化输出和错误处理
	"fmt"
	// math/rand 用于生成伪随机数
	"math/rand"
	// strconv 用于字符串和基本数据类型之间的转换
	"strconv"
	// strings 用于字符串处理
	"strings"
	// sync/atomic 用于原子操作
	"sync/atomic"
	// time 用于时间相关操作
	"time"
)

// globalCounter 用于生成连续IP地址的全局计数器
// 通过原子操作确保在并发环境下的安全性
var globalCounter int64

// VariableParser 变量解析器结构体，负责处理模板中的变量替换
type VariableParser struct {
	// random 随机数生成器，用于生成各种随机值
	random *rand.Rand
	// customVariables 存储注册的自定义变量，键为变量名（大写），值为变量配置
	customVariables map[string]CustomVariable
	// verbose 是否启用详细日志输出
	verbose bool
}

// NewVariableParser 创建并初始化一个新的变量解析器实例
// 参数:
//   - verbose: 是否启用详细日志输出，true表示输出详细日志，false表示只输出关键日志
//
// 返回值:
//   - *VariableParser: 初始化后的变量解析器实例
func NewVariableParser(verbose bool) *VariableParser {
	return &VariableParser{
		// 初始化自定义变量映射
		customVariables: make(map[string]CustomVariable),
		// 使用当前时间戳作为种子初始化随机数生成器
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		// 设置日志输出级别
		verbose: verbose,
	}
}

// RegisterCustomVariable 注册一个自定义变量到解析器中
// 参数:
//   - name: 变量名，将被自动转换为大写
//   - variable: 变量配置，包含类型、值范围等信息
//
// 返回值:
//   - error: 如果变量配置无效则返回错误，否则返回nil
//
// 支持的变量类型:
//   - random_choice: 从给定的值列表中随机选择一个
//   - random_int: 生成指定范围内的随机整数
//   - random_string: 生成指定长度的随机字符串
func (p *VariableParser) RegisterCustomVariable(name string, variable CustomVariable) error {
	// 验证变量配置
	switch variable.Type {
	case "random_choice":
		// 确保random_choice类型变量提供了可选值列表
		if len(variable.Values) == 0 {
			return fmt.Errorf("random_choice类型变量必须提供values列表")
		}
	case "random_int":
		// 确保random_int类型变量的最小值小于最大值
		if variable.Min >= variable.Max {
			return fmt.Errorf("random_int类型变量的min必须小于max")
		}
	case "random_string":
		// 确保random_string类型变量的长度大于0
		if variable.Length <= 0 {
			return fmt.Errorf("random_string类型变量的length必须大于0")
		}
	default:
		// 不支持的变量类型
		return fmt.Errorf("不支持的变量类型: %s", variable.Type)
	}

	// 存储变量配置，变量名统一转换为大写
	name = strings.ToUpper(name)
	p.customVariables[name] = variable
	// 如果启用了详细日志，输出注册信息
	if p.verbose {
		fmt.Printf("注册自定义变量: %s, 类型: %s\n", name, variable.Type)
	}
	return nil
}

// newRandom 创建一个新的随机数生成器
// 该方法通过多重保障机制确保生成的随机数具有足够的随机性：
// 1. 优先使用crypto/rand生成加密安全的随机种子
// 2. 如果crypto/rand失败，使用时间戳、全局计数器和伪随机数的组合作为备选种子
// 返回值:
//   - *rand.Rand: 初始化后的随机数生成器
func (p *VariableParser) newRandom() *rand.Rand {
	// 尝试使用crypto/rand生成真随机数作为种子
	seed := make([]byte, 8)
	_, err := cryptorand.Read(seed)
	if err == nil {
		// 将随机字节转换为int64作为种子
		seedInt := int64(binary.LittleEndian.Uint64(seed))
		return rand.New(rand.NewSource(seedInt))
	}

	// 如果获取随机种子失败，使用备选方案：
	// 1. 当前时间的纳秒级时间戳
	// 2. 原子递增的全局计数器
	// 3. 伪随机数
	// 通过异或运算组合这些值，提高随机性
	now := time.Now().UnixNano()
	seedInt := now ^ atomic.AddInt64(&globalCounter, 1) ^ rand.Int63()
	return rand.New(rand.NewSource(seedInt))
}

// Parse 解析变量表达式并生成对应的值
// 变量表达式格式: VARIABLE_NAME[:PARAMS]
// 示例:
//   - RANDOM_STRING:10 - 生成长度为10的随机字符串
//   - RANDOM_INT:1,100 - 生成1到100之间的随机整数
//   - ENUM:apple,banana,orange - 从给定列表中随机选择一个值
//   - CUSTOM_VAR - 使用自定义变量配置生成值
//
// 参数:
//   - expr: 变量表达式，格式为"变量名:参数"，参数部分可选
//
// 返回值:
//   - string: 生成的变量值
//   - error: 解析或生成过程中的错误，如果成功则为nil
func (p *VariableParser) Parse(expr string) (string, error) {
	// 分割变量名和参数
	// 使用SplitN确保只在第一个冒号处分割
	parts := strings.SplitN(expr, ":", 2)
	// 提取并标准化变量名（转换为大写）
	varName := strings.TrimSpace(parts[0])
	varName = strings.ToUpper(varName)
	// 提取参数（如果存在）
	var params string
	if len(parts) > 1 {
		params = strings.TrimSpace(parts[1])
	}

	// 优先检查是否是自定义变量
	if variable, ok := p.customVariables[varName]; ok {
		// 根据自定义变量类型生成值
		switch variable.Type {
		case "random_choice":
			// 从预定义的值列表中随机选择一个
			return variable.Values[p.random.Intn(len(variable.Values))], nil
		case "random_int":
			// 生成指定范围内的随机整数
			return fmt.Sprintf("%d", p.random.Intn(variable.Max-variable.Min)+variable.Min), nil
		case "random_string":
			// 生成指定长度的随机字符串
			return p.generateRandomString(fmt.Sprintf("%d", variable.Length))
		default:
			// 不支持的变量类型
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

// generateCustomVariable 根据自定义变量配置生成变量值
// 参数:
//   - name: 自定义变量名，必须已通过RegisterCustomVariable注册
//
// 返回值:
//   - string: 生成的变量值
//   - error: 生成过程中的错误，包括变量未找到或类型不支持
func (p *VariableParser) generateCustomVariable(name string) (string, error) {
	// 查找自定义变量配置
	variable, ok := p.customVariables[name]
	if !ok {
		return "", fmt.Errorf("未找到自定义变量: %s", name)
	}

	// 根据变量类型生成值
	switch variable.Type {
	case "random_choice":
		// 从预定义的值列表中随机选择
		return variable.Values[p.random.Intn(len(variable.Values))], nil
	case "random_int":
		// 生成指定范围内的随机整数
		return fmt.Sprintf("%d", p.random.Intn(variable.Max-variable.Min)+variable.Min), nil
	case "random_string":
		// 生成指定长度的随机字符串
		return p.generateRandomString(fmt.Sprintf("%d", variable.Length))
	default:
		// 不支持的变量类型
		return "", fmt.Errorf("不支持的变量类型: %s", variable.Type)
	}
}

// generateRandomString 生成随机字符串，支持带权重的选项
// 参数格式: "选项1[:权重1],选项2[:权重2],..."
// 示例:
//   - "10" - 生成长度为10的随机字符串
//   - "5:2,10:1" - 生成长度为5或10的随机字符串，5的权重为2，10的权重为1
//
// 参数:
//   - params: 字符串长度选项及其权重，多个选项用逗号分隔
//
// 返回值:
//   - string: 生成的随机字符串
//   - error: 生成过程中的错误，如参数格式错误
func (p *VariableParser) generateRandomString(params string) (string, error) {
	// 验证参数非空
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANDOM_STRING")
	}

	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 解析选项和权重
	// 格式："长度1:权重1,长度2:权重2,..."
	options := strings.Split(params, ",")
	weights := make([]int, len(options))
	totalWeight := 0

	// 处理每个选项及其权重
	for i, opt := range options {
		// 分离选项和权重值
		parts := strings.Split(strings.TrimSpace(opt), ":")
		options[i] = parts[0] // 选项（字符串长度）
		weight := 1           // 默认权重为1

		// 如果指定了权重，解析权重值
		if len(parts) > 1 {
			w, err := strconv.Atoi(parts[1])
			if err == nil && w > 0 {
				weight = w
			}
		}

		// 累加权重
		weights[i] = weight
		totalWeight += weight
	}

	// 根据权重随机选择一个选项
	r := random.Intn(totalWeight)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return options[i], nil
		}
	}

	return options[len(options)-1], nil
}

// generateRandomInt 生成指定范围内的随机整数
// 参数格式: "最小值-最大值"
// 示例:
//   - "1-100" - 生成1到100之间的随机整数
//   - "0-1000" - 生成0到1000之间的随机整数
//
// 参数:
//   - params: 整数范围，格式为"min-max"
//
// 返回值:
//   - string: 生成的随机整数字符串
//   - error: 生成过程中的错误，如参数格式错误或范围无效
func (p *VariableParser) generateRandomInt(params string) (string, error) {
	// 验证参数非空
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANDOM_INT")
	}

	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 解析范围参数
	// 格式："最小值-最大值"
	parts := strings.Split(params, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid range format for RANDOM_INT, expected min-max")
	}

	// 解析并验证最小值
	min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", fmt.Errorf("invalid minimum value: %s", parts[0])
	}

	// 解析并验证最大值
	max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", fmt.Errorf("invalid maximum value: %s", parts[1])
	}

	// 确保最小值小于最大值
	if min >= max {
		return "", fmt.Errorf("minimum value must be less than maximum value")
	}

	// 生成指定范围内的随机数
	// Intn(n)生成[0,n)范围的随机数，通过加上min调整到目标范围
	result := random.Intn(max-min+1) + min
	return strconv.Itoa(result), nil
}

// generateEnum 从给定的选项列表中随机选择一个值
// 参数格式: "选项1,选项2,选项3,..."
// 示例:
//   - "apple,banana,orange" - 随机选择一个水果名
//   - "error,warn,info,debug" - 随机选择一个日志级别
//
// 参数:
//   - params: 以逗号分隔的选项列表
//
// 返回值:
//   - string: 随机选择的选项
//   - error: 生成过程中的错误，如参数为空
func (p *VariableParser) generateEnum(params string) (string, error) {
	// 验证参数非空
	if params == "" {
		return "", fmt.Errorf("missing parameters for ENUM")
	}

	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 分割并处理选项列表
	// 移除每个选项两端的空白字符
	options := strings.Split(params, ",")
	for i := range options {
		options[i] = strings.TrimSpace(options[i])
	}

	// 随机选择一个选项
	// 使用Intn确保选择范围在有效索引内
	return options[random.Intn(len(options))], nil
}

// generateMAC 生成随机的MAC地址
// 格式: XX:XX:XX:XX:XX:XX，其中X为十六进制数字
// 示例: 12:34:56:78:9a:bc
// 返回值:
//   - string: 生成的MAC地址，格式为六组由冒号分隔的两位十六进制数
//   - error: 生成过程中的错误，一般不会发生错误
func (p *VariableParser) generateMAC() (string, error) {
	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 生成6字节的随机数据作为MAC地址
	mac := make([]byte, 6)
	random.Read(mac)

	// 格式化为标准MAC地址格式
	// 使用%02x确保每个字节都被格式化为两位十六进制数
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
}

// generateRandomIP 生成随机IPv4地址
// 参数格式:
//   - 空字符串: 生成完全随机的IP地址
//   - "start,end": 生成指定范围内的IP地址，目前仅支持最后一段可变
//
// 示例:
//   - "" - 生成任意IP地址，如"192.168.1.1"
//   - "192.168.1.1,192.168.1.100" - 生成192.168.1.1到192.168.1.100之间的IP
//
// 参数:
//   - params: IP地址范围，为空时生成任意IP，否则需要提供起始和结束IP
//
// 返回值:
//   - string: 生成的IP地址
//   - error: 生成过程中的错误，如参数格式错误
func (p *VariableParser) generateRandomIP(params string) (string, error) {
	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 无参数时生成完全随机的IP地址
	if params == "" {
		// 每段取值范围为[0,255]
		return fmt.Sprintf("%d.%d.%d.%d",
			random.Intn(256),
			random.Intn(256),
			random.Intn(256),
			random.Intn(256)), nil
	}

	// 解析IP范围参数
	// 格式："起始IP,结束IP"
	parts := strings.Split(params, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid IP range format, expected start,end")
	}

	// 移除IP地址两端的空白字符
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// 分割IP地址的各个段
	// 目前实现仅支持最后一段可变，前三段必须相同
	startParts := strings.Split(start, ".")
	endParts := strings.Split(end, ".")

	// 验证IP地址格式
	if len(startParts) != 4 || len(endParts) != 4 {
		return "", fmt.Errorf("invalid IP address format")
	}

	// 确保前三段IP地址相同
	// 这是当前实现的限制，未来可能会支持更复杂的范围
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

// generateInternalIP 生成随机的内网IP地址
// 支持三种内网地址段：
//   - 192.168.0.0/16 (192.168.0.0 - 192.168.255.255)
//   - 172.16.0.0/12 (172.16.0.0 - 172.31.255.255)
//   - 10.0.0.0/8 (10.0.0.0 - 10.255.255.255)
//
// 返回值:
//   - string: 生成的内网IP地址
//   - error: 生成过程中的错误，一般不会发生错误
func (p *VariableParser) generateInternalIP() (string, error) {
	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 随机选择一个内网IP范围
	switch random.Intn(3) {
	case 0: // 192.168.0.0/16 私有网络地址段
		return fmt.Sprintf("192.168.%d.%d",
				random.Intn(256),    // 第三段: 0-255
				random.Intn(254)+1), // 第四段: 1-254，避免使用0和255
			nil
	case 1: // 172.16.0.0/12 私有网络地址段
		return fmt.Sprintf("172.%d.%d.%d",
				16+random.Intn(16),  // 第二段: 16-31，确保在172.16-172.31范围内
				random.Intn(256),    // 第三段: 0-255
				random.Intn(254)+1), // 第四段: 1-254，避免使用0和255
			nil
	default: // 10.0.0.0/8 私有网络地址段
		return fmt.Sprintf("10.%d.%d.%d",
				random.Intn(256),    // 第二段: 0-255
				random.Intn(256),    // 第三段: 0-255
				random.Intn(254)+1), // 第四段: 1-254，避免使用0和255
			nil
	}
}

// generateExternalIP 生成随机的外网IP地址
// 生成规则：
//  1. 第一段: 1-223，排除0(保留)和127(回环地址)
//  2. 排除私有网络地址段：
//     - 10.0.0.0/8
//     - 172.16.0.0/12
//     - 192.168.0.0/16
//  3. 最后一段: 1-254，排除0和255
//
// 返回值:
//   - string: 生成的外网IP地址
//   - error: 生成过程中的错误，一般不会发生错误
func (p *VariableParser) generateExternalIP() (string, error) {
	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 循环生成直到得到有效的外网IP地址
	for {
		// 生成第一段，范围1-223
		// 排除0(保留地址)和127(回环地址)
		a := random.Intn(223) + 1
		if a == 127 {
			continue
		}

		// 生成剩余段
		b := random.Intn(256)     // 第二段: 0-255
		c := random.Intn(256)     // 第三段: 0-255
		d := random.Intn(254) + 1 // 第四段: 1-254，避免使用0和255

		// 排除所有私有网络地址段
		if (a == 10) || // 10.0.0.0/8 私有网络
			(a == 172 && b >= 16 && b <= 31) || // 172.16.0.0/12 私有网络
			(a == 192 && b == 168) { // 192.168.0.0/16 私有网络
			continue
		}

		// 返回有效的外网IP地址
		return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d), nil
	}
}

// generateRangeIP 生成指定范围内的IPv4地址
// 支持两种格式：
//  1. 起始IP-结束IP，如：192.168.1.1-192.168.1.100
//  2. CIDR格式，如：192.168.1.0/24
//
// 参数:
//   - params: IP地址范围，支持IP范围格式或CIDR格式
//
// 返回值:
//   - string: 生成的IP地址
//   - error: 生成过程中的错误，如参数格式错误或范围无效
func (p *VariableParser) generateRangeIP(params string) (string, error) {
	// 验证参数非空
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANGE_IP")
	}

	// 支持两种格式：
	// 1. 起始IP-结束IP，如：192.168.1.1-192.168.1.100
	// 2. CIDR格式，如：192.168.1.0/24
	if strings.Contains(params, "/") {
		// CIDR格式，交由专门的方法处理
		return p.generateIPFromCIDR(params)
	}

	// 处理IP范围格式
	parts := strings.Split(params, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid IP range format, expected start-end or CIDR notation")
	}

	// 移除IP地址两端的空白字符
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// 分割并验证IP地址格式
	startIP := strings.Split(start, ".")
	endIP := strings.Split(end, ".")

	if len(startIP) != 4 || len(endIP) != 4 {
		return "", fmt.Errorf("invalid IP address format")
	}

	// 将IP地址转换为32位整数以便比较和计算
	startNum := 0
	endNum := 0
	for i := 0; i < 4; i++ {
		// 解析并验证起始IP的每个段
		s, err := strconv.Atoi(startIP[i])
		if err != nil || s < 0 || s > 255 {
			return "", fmt.Errorf("invalid start IP: %s", start)
		}
		// 解析并验证结束IP的每个段
		e, err := strconv.Atoi(endIP[i])
		if err != nil || e < 0 || e > 255 {
			return "", fmt.Errorf("invalid end IP: %s", end)
		}
		// 将IP地址转换为32位整数
		startNum = startNum*256 + s
		endNum = endNum*256 + e
	}

	// 确保起始IP小于结束IP
	if startNum >= endNum {
		return "", fmt.Errorf("start IP must be less than end IP")
	}

	// 使用全局计数器实现连续生成
	// 获取当前计数器值并递增
	counter := atomic.AddInt64(&globalCounter, 1) - 1
	// 计算IP地址范围内的总地址数
	totalIPs := endNum - startNum + 1
	// 使用计数器值对总地址数取模，确保生成的IP在范围内循环
	num := startNum + int(counter%int64(totalIPs))

	// 将32位整数转换回点分十进制格式
	return fmt.Sprintf("%d.%d.%d.%d",
			(num>>24)&255, // 提取第一段
			(num>>16)&255, // 提取第二段
			(num>>8)&255,  // 提取第三段
			num&255),      // 提取第四段
		nil
}

// generateIPFromCIDR 从CIDR格式生成随机IP地址
// CIDR格式：IP地址/掩码长度
// 示例：
//   - 192.168.1.0/24 表示192.168.1.0-192.168.1.255范围
//   - 10.0.0.0/8 表示10.0.0.0-10.255.255.255范围
//   - 172.16.0.0/12 表示172.16.0.0-172.31.255.255范围
//
// 参数:
//   - cidr: CIDR格式的网络地址范围
//
// 返回值:
//   - string: 生成的IP地址
//   - error: 生成过程中的错误，如CIDR格式错误
func (p *VariableParser) generateIPFromCIDR(cidr string) (string, error) {
	// 分割IP地址和掩码长度
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
// 目前仅支持CIDR格式，格式：IPv6地址/掩码长度
// 示例：
//   - 2001:db8::/32 表示2001:db8::到2001:db8:ffff:ffff:ffff:ffff:ffff:ffff范围
//   - fe80::/64 表示fe80::到fe80::ffff:ffff:ffff:ffff范围
//
// 参数:
//   - params: CIDR格式的IPv6网络地址范围
//
// 返回值:
//   - string: 生成的IPv6地址
//   - error: 生成过程中的错误，如CIDR格式错误或掩码无效
func (p *VariableParser) generateRangeIPv6(params string) (string, error) {
	// 验证参数非空
	if params == "" {
		return "", fmt.Errorf("missing parameters for RANGE_IP v6")
	}

	// 目前仅支持CIDR格式，如：2001:db8::/32
	if strings.Contains(params, "/") {
		// 分割IPv6地址和掩码长度
		parts := strings.Split(params, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid IPv6 CIDR format")
		}

		// 解析并验证掩码长度（0-128）
		mask, err := strconv.Atoi(parts[1])
		if err != nil || mask < 0 || mask > 128 {
			return "", fmt.Errorf("invalid IPv6 network mask")
		}

		// 解析基础IPv6地址
		// IPv6地址由8组16位十六进制数组成，每组用冒号分隔
		base := strings.Split(parts[0], ":")
		if len(base) > 8 {
			return "", fmt.Errorf("invalid IPv6 address format")
		}

		// 处理IPv6地址中的压缩符号 ::
		// 例如：2001:db8::1 实际表示 2001:db8:0:0:0:0:0:1
		for i, part := range base {
			if part == "" {
				// 计算需要插入的0组数量
				zeros := 8 - (len(base) - 1)
				// 创建新的地址数组并填充0
				newBase := make([]string, 8)
				// 复制压缩符号前的部分
				copy(newBase[:i], base[:i])
				// 填充中间的0
				for j := 0; j < zeros; j++ {
					newBase[i+j] = "0"
				}
				// 复制压缩符号后的部分
				if i+1 < len(base) {
					copy(newBase[i+zeros:], base[i+1:])
				}
				base = newBase
				break
			}
		}

		// 使用全局计数器实现连续生成
		counter := atomic.AddInt64(&globalCounter, 1) - 1
		result := make([]string, 8)

		// 保持网络部分不变（由掩码长度决定）
		// 每组占16位，所以将掩码长度除以16得到需要保持不变的组数
		networkParts := mask / 16
		for i := 0; i < networkParts; i++ {
			if i < len(base) {
				result[i] = base[i] // 使用原始值
			} else {
				result[i] = "0" // 补充0
			}
		}

		// 生成主机部分，使用计数器值实现顺序生成
		for i := networkParts; i < 8; i++ {
			// 每组16位，使用计数器的不同部分
			// 通过位移和掩码提取计数器值的不同部分
			shift := uint((7 - i) * 16)
			value := (counter >> shift) & 0xFFFF
			// 格式化为4位十六进制数
			result[i] = fmt.Sprintf("%04x", value)
		}

		// 组合各部分，生成完整的IPv6地址
		return strings.Join(result, ":"), nil
	}

	return "", fmt.Errorf("only CIDR notation is supported for IPv6 range")
}

// generateEmail 生成随机的邮箱地址
// 生成规则：
//  1. 用户名：6-12个字符，仅包含小写字母和数字
//  2. 域名：从预定义的常见邮箱服务商中随机选择
//
// 示例：
//   - user123@gmail.com
//   - test456@outlook.com
//
// 返回值:
//   - string: 生成的邮箱地址
//   - error: 生成过程中的错误，一般不会发生错误
func (p *VariableParser) generateEmail() (string, error) {
	// 创建新的随机数生成器，确保随机性
	random := p.newRandom()

	// 预定义常见邮箱服务商域名
	domains := []string{
		"juminfo.com",    // 企业邮箱示例
		"gmail.com",      // Google邮箱
		"yahoo.com",      // Yahoo邮箱
		"hotmail.com",    // 微软Hotmail
		"outlook.com",    // 微软Outlook
		"protonmail.com", // ProtonMail加密邮箱
		"icloud.com",     // 苹果iCloud邮箱
	}

	// 用户名允许的字符集
	// 仅使用小写字母和数字，避免特殊字符
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"

	// 生成随机长度的用户名（6-12字符）
	usernameLen := random.Intn(7) + 6 // 随机长度范围：[6,12]
	username := make([]byte, usernameLen)
	// 从字符集中随机选择字符
	for i := range username {
		username[i] = charset[random.Intn(len(charset))]
	}

	// 随机选择一个邮箱域名
	domain := domains[random.Intn(len(domains))]

	// 组合用户名和域名，生成完整的邮箱地址
	return fmt.Sprintf("%s@%s", string(username), domain), nil
}

// generateDomain 生成随机的域名
// 生成规则：
//  1. 域名前缀：从预定义的常见前缀中随机选择
//  2. 域名中间部分：从预定义的功能描述词中随机选择
//  3. 公司名称：从预定义的知名公司列表中随机选择
//  4. 顶级域名：从预定义的顶级域名列表中随机选择
//
// 支持四种生成方式：
//  1. 公司域名: company.tld (如 google.com)
//  2. 服务域名: prefix.company.tld (如 api.amazon.com)
//  3. 功能域名: prefix-middle.tld (如 cloud-storage.io)
//  4. 随机字符域名: random.tld (如 abc123.com)
//
// 返回值:
//   - string: 生成的域名
//   - error: 生成过程中的错误，一般不会发生错误
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
		compressStart := random.Intn(5) + 1  // 不压缩第一组
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
