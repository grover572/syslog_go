package template

import (
	"bufio"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Engine 模板引擎
type Engine struct {
	templates  map[string][]string     // 模板内容缓存
	variables  map[string]*Variable   // 变量定义
	generators map[string]func() string // 变量生成器
	mutex      sync.RWMutex           // 读写锁
	random     *rand.Rand             // 随机数生成器
}

// Variable 变量定义
type Variable struct {
	Type    string            `yaml:"type"`
	Values  []string          `yaml:"values"`
	Weights map[string]int    `yaml:"weights"`
	Min     int               `yaml:"min"`
	Max     int               `yaml:"max"`
	Format  string            `yaml:"format"`
	Ranges  []string          `yaml:"ranges"`
}

// VariableConfig 变量配置文件结构
type VariableConfig struct {
	Variables map[string]*Variable `yaml:"variables"`
}

// NewEngine 创建新的模板引擎
func NewEngine() *Engine {
	return &Engine{
		templates:  make(map[string][]string),
		variables:  make(map[string]*Variable),
		generators: make(map[string]func() string),
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// LoadTemplatesFromDir 从目录加载模板文件
func (e *Engine) LoadTemplatesFromDir(dir string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理.log文件
		if d.IsDir() || !strings.HasSuffix(path, ".log") {
			return nil
		}

		// 读取文件内容
		lines, err := e.readTemplateFile(path)
		if err != nil {
			return fmt.Errorf("读取模板文件 %s 失败: %w", path, err)
		}

		// 使用相对路径作为键
		relPath, _ := filepath.Rel(dir, path)
		key := strings.ReplaceAll(relPath, "\\", "/") // 统一使用正斜杠
		e.templates[key] = lines

		return nil
	})
}

// LoadTemplateFromFile 从单个文件加载模板
func (e *Engine) LoadTemplateFromFile(filePath string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	lines, err := e.readTemplateFile(filePath)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %w", err)
	}

	key := filepath.Base(filePath)
	e.templates[key] = lines
	return nil
}

// LoadVariables 加载变量配置
func (e *Engine) LoadVariables(configPath string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取变量配置文件失败: %w", err)
	}

	var config VariableConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析变量配置失败: %w", err)
	}

	e.variables = config.Variables
	e.initGenerators()
	return nil
}

// readTemplateFile 读取模板文件内容
func (e *Engine) readTemplateFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}

// GenerateMessage 生成一条日志消息
func (e *Engine) GenerateMessage(templateKey string) (string, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// 如果指定了模板键，使用指定模板
	if templateKey != "" {
		if templates, ok := e.templates[templateKey]; ok {
			return e.processTemplate(templates), nil
		}
		return "", fmt.Errorf("模板 %s 不存在", templateKey)
	}

	// 随机选择一个模板
	if len(e.templates) == 0 {
		return "", fmt.Errorf("没有可用的模板")
	}

	// 收集所有模板
	var allTemplates []string
	for _, templates := range e.templates {
		allTemplates = append(allTemplates, templates...)
	}

	return e.processTemplate(allTemplates), nil
}

// processTemplate 处理模板，替换占位符
func (e *Engine) processTemplate(templates []string) string {
	if len(templates) == 0 {
		return "Empty template"
	}

	// 随机选择一行模板
	template := templates[e.random.Intn(len(templates))]

	// 查找所有占位符
	re := regexp.MustCompile(`{{\s*([^}]+)\s*}}`)
	matches := re.FindAllStringSubmatch(template, -1)

	// 替换占位符
	result := template
	for _, match := range matches {
		placeholder := match[0] // 完整的占位符 {{variable}}
		variableName := strings.TrimSpace(match[1]) // 变量名

		// 生成变量值
		value := e.generateVariable(variableName)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// generateVariable 生成变量值
func (e *Engine) generateVariable(name string) string {
	// 首先检查是否有自定义生成器
	if generator, ok := e.generators[name]; ok {
		return generator()
	}

	// 检查是否有变量定义
	if variable, ok := e.variables[name]; ok {
		return e.generateFromVariable(variable)
	}

	// 如果没有定义，返回占位符
	return fmt.Sprintf("{{%s}}", name)
}

// generateFromVariable 根据变量定义生成值
func (e *Engine) generateFromVariable(variable *Variable) string {
	switch variable.Type {
	case "random_choice":
		if len(variable.Values) > 0 {
			return variable.Values[e.random.Intn(len(variable.Values))]
		}
	case "weighted_choice":
		return e.generateWeightedChoice(variable)
	case "random_int":
		min := variable.Min
		max := variable.Max
		if max <= min {
			max = min + 100
		}
		return fmt.Sprintf("%d", e.random.Intn(max-min+1)+min)
	case "random_ip":
		return e.generateRandomIP(variable.Ranges)
	case "timestamp":
		format := variable.Format
		if format == "" {
			format = "2006-01-02 15:04:05"
		}
		return time.Now().Format(format)
	}

	return "unknown"
}

// generateWeightedChoice 生成加权随机选择
func (e *Engine) generateWeightedChoice(variable *Variable) string {
	if len(variable.Weights) == 0 {
		if len(variable.Values) > 0 {
			return variable.Values[e.random.Intn(len(variable.Values))]
		}
		return "unknown"
	}

	// 计算总权重
	totalWeight := 0
	for _, weight := range variable.Weights {
		totalWeight += weight
	}

	// 随机选择
	randomValue := e.random.Intn(totalWeight)
	currentWeight := 0

	for value, weight := range variable.Weights {
		currentWeight += weight
		if randomValue < currentWeight {
			return value
		}
	}

	// 默认返回第一个值
	for value := range variable.Weights {
		return value
	}

	return "unknown"
}

// generateRandomIP 生成随机IP地址
func (e *Engine) generateRandomIP(ranges []string) string {
	if len(ranges) == 0 {
		// 默认私有IP范围
		return fmt.Sprintf("192.168.%d.%d",
			e.random.Intn(256),
			e.random.Intn(254)+1)
	}

	// 随机选择一个范围
	rangeStr := ranges[e.random.Intn(len(ranges))]
	return e.parseIPRange(rangeStr)
}

// parseIPRange 解析IP范围并生成随机IP
func (e *Engine) parseIPRange(rangeStr string) string {
	// 简单实现：支持 "192.168.1.1-192.168.1.254" 格式
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return "192.168.1.1"
	}

	// 这里简化处理，实际应该解析IP范围
	// 暂时返回范围内的随机IP
	return fmt.Sprintf("192.168.%d.%d",
		e.random.Intn(256),
		e.random.Intn(254)+1)
}

// GetTemplateKeys 获取所有模板键
func (e *Engine) GetTemplateKeys() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	keys := make([]string, 0, len(e.templates))
	for key := range e.templates {
		keys = append(keys, key)
	}
	return keys
}

// GetTemplateCount 获取模板总数
func (e *Engine) GetTemplateCount() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	count := 0
	for _, templates := range e.templates {
		count += len(templates)
	}
	return count
}