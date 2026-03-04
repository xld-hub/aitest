# AI Agent 项目

基于OpenAI协议的Go语言AI Agent实现，包含详细的代码注释和学习资料。

## 📁 项目结构

```
test/
├── main.go              # 基础的OpenAI API调试示例
├── types.go             # 公共类型定义（共享数据结构）
├── agent.go             # Agent核心实现（推荐学习）⭐
├── LEARNING_GUIDE.md    # 详细学习指南
└── examples/            # 其他示例文件
    ├── agent_example.go
    └── agent_detailed.go
```

## 🚀 快速开始

### 1. 运行基础Agent示例

```bash
# 方式1: 直接运行
go run agent.go types.go

# 方式2: 先编译再运行
go build -o agent.exe agent.go types.go
./agent.exe
```

### 2. 运行基础API测试

```bash
go run main.go
```

## 📚 文件说明

### agent.go + types.go （推荐）
- **agent.go**: Agent核心实现，包含三种示例
  - SimpleChat(): 简单流式对话
  - ToolUsingAgent(): 工具调用示例
  - CodeAssistant(): 代码助手示例
- **types.go**: 所有共享的数据结构定义

**特点**：
- ✅ 完整的Agent实现
- ✅ 详细的中文注释
- ✅ 流式响应处理
- ✅ 工具调用支持
- ✅ 无编译错误

### main.go
- 基础的OpenAI API调试代码
- 演示HTTP请求和响应处理
- 适合理解API调用流程

### LEARNING_GUIDE.md
- Go语言基础概念讲解
- 代码结构详细解析
- 练习题（初级/中级/高级）
- 调试技巧

## 🎯 学习路径

### 第1步：理解基础（1-2天）
1. 阅读 `LEARNING_GUIDE.md` 的前3节
2. 理解Go的基本概念（变量、函数、结构体）
3. 查看 `types.go` 了解数据结构

### 第2步：阅读代码（2-3天）
1. 仔细阅读 `agent.go` 的所有注释
2. 重点理解：
   - `Agent` 结构体
   - `Chat()` 方法（流式处理）
   - `ChatWithTools()` 方法（工具调用）

### 第3步：动手修改（3-5天）
1. 修改系统提示，改变AI的行为
2. 添加自己的测试问题
3. 自定义流式输出格式

### 第4步：实践项目（1-2周）
1. 添加新工具（如计算器、翻译）
2. 实现对话历史保存
3. 添加重试机制
4. 构建完整的对话应用

## 💡 代码示例

### 创建Agent
```go
agent := NewAgent(
    "your-api-key",
    "https://api.openai.com/v1",
    "gpt-3.5-turbo",
)
```

### 简单对话
```go
agent.AddMessage("system", "你是一个友好的助手")
err := agent.Chat("你好，请介绍一下自己")
```

### 自定义输出
```go
agent.SetStreamCallback(func(content string) {
    fmt.Print("[AI] " + content)
})
```

### 工具调用
```go
agent.RegisterTool("my_tool", "我的工具", myToolFunction)
err := agent.ChatWithTools("使用工具帮我查询", 5)
```

## 🔧 配置说明

### API配置
在 `agent.go` 中修改以下配置：

```go
apiKey := "your-api-key-here"  // 替换为你的API密钥
baseURL := "https://api.openai.com/v1"  // API地址
model := "gpt-3.5-turbo"  // 模型名称
```

### 支持的API
- ✅ OpenAI官方API
- ✅ 智谱AI (bigmodel.cn)
- ✅ Azure OpenAI
- ✅ 任何兼容OpenAI协议的服务

### 切换示例
编辑 `agent.go` 的 `main()` 函数：

```go
func main() {
    SimpleChat()       // 启用简单对话
    // ToolUsingAgent() // 取消注释启用工具调用
    // CodeAssistant()  // 取消注释启用代码助手
}
```

## 📖 核心概念

### 流式 vs 非流式
| 特性 | 流式 ⭐ | 非流式 |
|------|---------|--------|
| 响应速度 | 快 | 慢 |
| 用户体验 | 好 | 差 |
| 实现难度 | 中 | 低 |

**建议**: Agent应该使用流式响应

### 消息历史管理
```go
// Agent自动维护对话历史
agent.messages = []ChatMessage{
    {Role: "system", Content: "你是助手"},
    {Role: "user", Content: "你好"},
    {Role: "assistant", Content: "你好！"},
    // ... 更多消息
}
```

### 工具调用流程
```
用户提问 → AI分析 → 调用工具 → 获取结果 → 继续回答
```

## 🐛 常见问题

### Q1: 编译错误 "redeclared"
**A**: 确保只运行一个文件，不要同时运行多个包含相同类型的文件

```bash
# ✅ 正确
go run agent.go types.go

# ❌ 错误（会导致重复声明）
go run agent.go agent_example.go types.go
```

### Q2: API返回401错误
**A**: 检查API密钥是否正确

### Q3: 流式响应解析失败
**A**: 确保API返回的是SSE格式（每行以`data:`开头）

### Q4: 如何调试？
**A**: 添加调试输出
```go
fmt.Printf("调试: %+v\n", reqBody)
```

## 📚 学习资源

### 官方资源
- [Go官方教程](https://go.dev/tour/)
- [Go官方文档](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)

### 推荐书籍
- 《Go语言圣经》
- 《Go语言实战》

### 在线练习
- [Go Exercises](https://go.dev/tour/)
- [LeetCode Go版](https://leetcode.cn/)

## 🎓 进阶主题

完成基础学习后，可以探索：
1. **并发编程**: goroutine和channel
2. **Web服务**: 使用net/http创建REST API
3. **数据库**: 使用GORM或sqlx
4. **测试**: 编写单元测试和基准测试
5. **部署**: Docker容器化，部署到云平台

## 📝 练习建议

### 初级练习
1. 修改系统提示，改变AI角色
2. 添加更多测试问题
3. 自定义流式输出格式（添加颜色、时间戳）

### 中级练习
1. 添加新工具（计算器、翻译等）
2. 实现对话历史保存到文件
3. 添加重试机制处理网络错误

### 高级练习
1. 实现流式工具调用
2. 添加对话历史优化（只保留最近N条）
3. 实现并发安全的Agent（使用sync.Mutex）
4. 创建Web服务封装Agent功能

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可

MIT License

---

## 👨‍💻 作者说明

这个项目是为了帮助Go初学者理解：
- Go语言的基本语法
- HTTP客户端编程
- JSON数据处理
- Agent的设计模式
- 流式数据处理

希望这个项目能帮助你学习Go和AI Agent开发！

祝你学习愉快！ 🎉
