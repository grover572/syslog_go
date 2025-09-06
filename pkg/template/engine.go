package template

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Engine 模板引擎
type Engine struct {
	templateCache map[string]string
	parser       *VariableParser
	configPath   string        // 自定义变量配置文件路径
	verbose     bool          // 是否显示详细日志
}

// NewEngine 创建新的模板引擎
func NewEngine(configPath string, verbose bool) *Engine {
	// 创建变量解析器
	parser := NewVariableParser(verbose)

	e := &Engine{
		templateCache: make(map[string]string),
		parser:       parser,
		configPath:   configPath,
		verbose:     verbose,
	}
	
	// 如果提供了配置文件路径，尝试加载自定义变量
	if configPath != "" {
		if e.verbose {
			fmt.Printf("正在加载配置文件: %s\n", configPath)
		}
		if err := e.loadCustomVariables(configPath); err != nil {
			if e.verbose {
				fmt.Printf("警告: 加载自定义变量配置失败: %v\n", err)
			}
		} else if e.verbose {
			fmt.Printf("成功加载自定义变量配置\n")
		}
	} else if e.verbose {
		fmt.Printf("未提供配置文件路径\n")
	}
	
	return e
}

// LoadTemplate 加载模板
func (e *Engine) LoadTemplate(name, content string) {
	e.templateCache[name] = content
}

// GenerateMessage 生成消息
func (e *Engine) GenerateMessage(templateName string) (string, error) {
	template, ok := e.templateCache[templateName]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	return e.processTemplate(template)
}

// SetVariableParser 设置变量解析器
func (e *Engine) SetVariableParser(parser *VariableParser) {
	e.parser = parser
}

// processTemplate 处理模板
func (e *Engine) processTemplate(template string) (string, error) {
	// 匹配变量表达式 {{变量名:参数}}
	varRegex := regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

	// 替换所有变量
	var lastErr error
	result := varRegex.ReplaceAllStringFunc(template, func(match string) string {
		// 提取变量表达式
		expr := varRegex.FindStringSubmatch(match)[1]

		// 使用当前的变量解析器
		value, err := e.parser.Parse(expr)
		if err != nil {
			// 记录错误
			lastErr = fmt.Errorf("解析变量 %s 失败: %w", expr, err)
			// 如果解析失败，保留原始表达式
			return match
		}

		return value
	})

	if lastErr != nil {
		return "", lastErr
	}

	return strings.TrimSpace(result), nil
}

// CustomVariable 自定义变量配置结构
type CustomVariable struct {
	Type   string   `yaml:"type"`
	Values []string `yaml:"values,omitempty"`
	Min    int      `yaml:"min,omitempty"`
	Max    int      `yaml:"max,omitempty"`
	Length int      `yaml:"length,omitempty"`
}

// CustomVariableConfig 自定义变量配置文件结构
type CustomVariableConfig struct {
	Variables map[string]CustomVariable `yaml:"variables"`
}

// loadCustomVariables 从YAML文件加载自定义变量配置
func (e *Engine) loadCustomVariables(configPath string) error {
	// 读取配置文件
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML配置
	var config CustomVariableConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("解析YAML配置失败: %w", err)
	}

	// 注册自定义变量到解析器
	for name, variable := range config.Variables {
		if err := e.parser.RegisterCustomVariable(name, variable); err != nil {
			if e.verbose {
				fmt.Printf("警告: 注册自定义变量 %s 失败: %v\n", name, err)
			}
		}
	}

	return nil
}
