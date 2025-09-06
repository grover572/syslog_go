package template

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Engine 模板引擎结构体，负责处理消息模板和变量替换
type Engine struct {
	templateCache map[string]string    // 模板缓存，存储已加载的模板内容
	parser       *VariableParser      // 变量解析器，用于解析和替换模板中的变量
	configPath   string              // 自定义变量配置文件路径
	verbose     bool                // 是否显示详细日志信息
}

// NewEngine 创建新的模板引擎实例
// 参数：
//   - configPath: 自定义变量配置文件路径
//   - verbose: 是否启用详细日志输出
// 返回值：
//   - *Engine: 创建的模板引擎实例
func NewEngine(configPath string, verbose bool) *Engine {
	// 创建变量解析器实例
	parser := NewVariableParser(verbose)

	// 初始化引擎实例
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

// LoadTemplate 加载模板到缓存
// 参数：
//   - name: 模板名称，用于标识模板
//   - content: 模板内容
func (e *Engine) LoadTemplate(name, content string) {
	e.templateCache[name] = content
}

// GenerateMessage 根据模板名称生成消息
// 参数：
//   - templateName: 模板名称
// 返回值：
//   - string: 生成的消息内容
//   - error: 生成过程中的错误，如果生成成功则为nil
func (e *Engine) GenerateMessage(templateName string) (string, error) {
	template, ok := e.templateCache[templateName]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	return e.processTemplate(template)
}

// SetVariableParser 设置变量解析器
// 参数：
//   - parser: 新的变量解析器实例
// 说明：
//   此方法允许在运行时更换变量解析器，用于支持不同的变量解析策略
func (e *Engine) SetVariableParser(parser *VariableParser) {
	e.parser = parser
}

// processTemplate 处理模板内容，替换变量表达式
// 参数：
//   - template: 要处理的模板字符串
// 返回值：
//   - string: 处理后的字符串，所有变量表达式都被替换为实际值
//   - error: 处理过程中的错误，如果处理成功则为nil
// 说明：
//   变量表达式格式：{{变量名:参数}}
//   示例：
//   - {{timestamp}}
//   - {{random_int:1,100}}
//   - {{random_string:10}}
func (e *Engine) processTemplate(template string) (string, error) {
	// 匹配变量表达式 {{变量名:参数}}
	varRegex := regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

	// 替换所有变量表达式
	var lastErr error
	result := varRegex.ReplaceAllStringFunc(template, func(match string) string {
		// 提取变量表达式（去除{{}}和空白字符）
		expr := varRegex.FindStringSubmatch(match)[1]

		// 使用变量解析器生成实际值
		value, err := e.parser.Parse(expr)
		if err != nil {
			// 记录错误信息
			lastErr = fmt.Errorf("解析变量[%s]失败: %w", expr, err)
			// 解析失败时保留原始表达式
			return match
		}

		return value
	})

	// 如果处理过程中出现错误，返回错误信息
	if lastErr != nil {
		return "", lastErr
	}

	// 去除结果中的首尾空白字符
	return strings.TrimSpace(result), nil
}

// CustomVariable 自定义变量配置结构
type CustomVariable struct {
	Type   string   `yaml:"type"`              // 变量类型（如random_int、random_string等）
	Values []string `yaml:"values,omitempty"`  // 可选值列表，用于random_choice类型
	Min    int      `yaml:"min,omitempty"`     // 最小值，用于random_int类型
	Max    int      `yaml:"max,omitempty"`     // 最大值，用于random_int类型
	Length int      `yaml:"length,omitempty"`  // 字符串长度，用于random_string类型
}

// CustomVariableConfig 自定义变量配置文件结构
type CustomVariableConfig struct {
	Variables map[string]CustomVariable `yaml:"variables"` // 变量名到配置的映射
}

// loadCustomVariables 从YAML文件加载自定义变量配置
// 参数：
//   - configPath: 配置文件路径
// 返回值：
//   - error: 加载过程中的错误，如果加载成功则为nil
// 说明：
//   配置文件格式（YAML）：
//   variables:
//     变量名:
//       type: 变量类型
//       values: [可选值列表]  # 用于random_choice类型
//       min: 最小值          # 用于random_int类型
//       max: 最大值          # 用于random_int类型
//       length: 字符串长度    # 用于random_string类型
func (e *Engine) loadCustomVariables(configPath string) error {
	// 读取配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML格式的配置内容
	var config CustomVariableConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("解析YAML配置失败: %w", err)
	}

	// 注册所有自定义变量到解析器
	for name, variable := range config.Variables {
		if err := e.parser.RegisterCustomVariable(name, variable); err != nil {
			if e.verbose {
				fmt.Printf("警告: 注册自定义变量[%s]失败: %v\n", name, err)
			}
		}
	}

	return nil
}
