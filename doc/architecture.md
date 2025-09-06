# 代码架构说明

## 项目结构

```mermaid
graph TD
    A[main.go] --> B[cmd]
    B --> B1[root.go]
    B --> B2[server.go]
    
    A --> C[pkg]
    C --> D[config]
    C --> E[sender]
    C --> F[server]
    C --> G[syslog]
    C --> H[template]
    C --> I[ui]
    
    D --> D1[config.go]
    
    E --> E1[connection.go]
    E --> E2[sender.go]
    
    F --> F1[server.go]
    
    G --> G1[protocol.go]
    
    H --> H1[engine.go]
    H --> H2[parser.go]
    
    I --> I1[interactive.go]
```

## 核心组件调用关系

```mermaid
sequenceDiagram
    participant M as Main
    participant C as Config
    participant T as Template
    participant S as Sender
    participant Sv as Server
    participant U as UI
    
    M->>C: 加载配置
    M->>U: 启动交互界面
    U->>C: 更新配置
    
    alt 发送模式
        M->>T: 初始化模板引擎
        T->>T: 加载模板
        M->>S: 创建发送器
        S->>S: 建立连接
        loop 消息发送
            S->>T: 生成消息
            S->>S: 发送消息
        end
    else 服务器模式
        M->>Sv: 启动服务器
        loop 消息接收
            Sv->>Sv: 接收消息
            Sv->>Sv: 处理消息
        end
    end
```

## 组件职责

1. **Config（配置管理）**
   - 加载和验证配置文件
   - 提供默认配置
   - 支持配置更新

2. **Template（模板引擎）**
   - 模板解析和渲染
   - 变量替换处理
   - 支持自定义变量

3. **Sender（发送器）**
   - 管理网络连接
   - 控制发送速率
   - 处理重试和错误

4. **Server（服务器）**
   - 监听网络端口
   - 接收和处理消息
   - 支持TCP/UDP协议

5. **UI（交互界面）**
   - 提供命令行交互
   - 配置参数管理
   - 显示运行状态

## 数据流

```mermaid
flowchart LR
    A[配置文件] --> B[Config]
    B --> C[Template]
    C --> D[Sender]
    D --> E[目标服务器]
    
    F[模板文件] --> C
    G[变量配置] --> C
    
    H[用户输入] --> I[UI]
    I --> B
```