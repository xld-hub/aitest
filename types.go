package main

import (
	"encoding/json" // JSON编码和解码包
)

// ==================== 公共类型定义 ====================
// 这个文件包含所有共享的数据结构定义
//
// 💡 什么是结构体（struct）？
// 结构体是Go中用于组织相关数据的方式，类似于其他语言的"类"或"对象"
// 例如：一个人有姓名、年龄、地址等属性，可以用结构体表示

// ============================================================================
// 第一部分：聊天相关的数据结构
// ============================================================================

// ChatRequest 聊天请求结构体
//
// 💡 用途：发送给OpenAI API的请求数据格式
//
// 📝 示例JSON：
//
//	{
//	  "model": "gpt-3.5-turbo",
//	  "messages": [{"role": "user", "content": "你好"}],
//	  "stream": false
//	}
type ChatRequest struct {
	// Model 字段：指定要使用的AI模型名称
	// 类型：string（字符串）
	// 示例值："gpt-3.5-turbo", "gpt-4", "glm-5"
	//
	// 💡 `json:"model"` 是结构体标签（struct tag）
	//    告诉JSON库：Go字段名"Model"对应JSON中的"model"键
	//    这样JSON序列化时会自动转换
	Model string `json:"model"`

	// Messages 字段：对话消息列表
	// 类型：[]ChatMessage（ChatMessage类型的切片/数组）
	//
	// 💡 切片（slice）是Go的动态数组，可以自动增长
	//    []string 表示字符串切片
	//    []ChatMessage 表示ChatMessage结构体切片
	//
	// 📝 包含整个对话历史，例如：
	//    [
	//      {"role": "system", "content": "你是一个助手"},
	//      {"role": "user", "content": "你好"},
	//      {"role": "assistant", "content": "你好！"}
	//    ]
	Messages []ChatMessage `json:"messages"`

	// Stream 字段：是否使用流式响应
	// 类型：bool（布尔值，只能是true或false）
	//
	// 💡 true=流式响应（逐字返回，像打字机效果）
	//    false=普通响应（一次性返回完整内容）
	//
	// 📝 Agent通常使用true，提供更好的用户体验
	Stream bool `json:"stream"`

	// Tools 字段：可用的工具列表（可选）
	// 类型：[]Tool（Tool结构体切片）
	//
	// 💡 omitempty标签的含义：
	//    "omitempty" = "如果为空则省略"
	//    如果Tools为空，序列化JSON时会忽略这个字段
	//
	// 📝 用于Function Calling功能，定义AI可以调用的工具
	//    例如：天气查询工具、计算器工具等
	Tools []Tool `json:"tools,omitempty"`
}

// ChatMessage 聊天消息结构体
//
// 💡 用途：表示对话中的一条消息
//
// 📝 示例：
// {"role": "user", "content": "什么是AI？"}
type ChatMessage struct {
	// Role 字段：消息发送者的角色
	// 类型：string
	//
	// 📝 可选值：
	//    "system"      - 系统消息，设定AI的行为和角色
	//                    例如："你是一个专业的编程助手"
	//    "user"        - 用户消息，用户说的话
	//                    例如："帮我写一个快速排序"
	//    "assistant"   - AI助手的回复
	//                    例如："好的，这是快速排序代码..."
	//    "tool"        - 工具执行结果（用于工具调用场景）
	Role string `json:"role"`

	// Content 字段：消息的文本内容
	// 类型：string
	//
	// 📝 示例：
	//    "你好，请介绍一下Go语言"
	//    "Go是一门开源的编程语言..."
	Content string `json:"content"`

	// ToolCalls 字段：AI发起的工具调用列表（可选）
	// 类型：[]ToolCall
	//
	// 💡 当AI需要使用工具时会填充这个字段
	//    例如：AI想查询天气，会包含一个"get_weather"工具调用
	//    omitempty表示如果没有工具调用，这个字段不会出现在JSON中
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID 字段：工具调用的唯一标识符（可选）
	// 类型：string
	//
	// 💡 用途：关联工具调用和工具结果
	//    当role="tool"时，用这个ID告诉AI这是哪个工具的执行结果
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ============================================================================
// 第二部分：响应相关的数据结构
// ============================================================================

// ChatResponse 聊天响应结构体
//
// 💡 用途：表示从OpenAI API返回的完整响应（非流式）
//
// 📝 示例JSON：
//
//	{
//	  "id": "chatcmpl-123",
//	  "choices": [{
//	    "message": {"role": "assistant", "content": "你好！"},
//	    "finish_reason": "stop"
//	  }],
//	  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
//	}
type ChatResponse struct {
	// ID 字段：响应的唯一标识符
	// 类型：string
	//
	// 📝 示例值："chatcmpl-123", "chatcmpl-456"
	//    每次请求都会生成不同的ID，用于追踪和调试
	ID string `json:"id"`

	// Choices 字段：AI生成的候选回复列表
	// 类型：[]Choice（Choice结构体切片）
	//
	// 💡 虽然是复数，但通常只有一个元素（choices[0]）
	//    某些模型可能返回多个候选供选择
	//
	// 📝 示例：包含AI的回复内容、结束原因等
	Choices []Choice `json:"choices"`

	// Usage 字段：Token使用统计信息
	// 类型：Usage结构体
	//
	// 💡 告诉你这次请求消耗了多少Token
	//    用于计算成本和监控使用量
	Usage Usage `json:"usage"`
}

// Choice 选择项结构体
//
// 💡 用途：表示AI生成的一个候选回复
type Choice struct {
	// Message 字段：AI返回的消息内容
	// 类型：ChatMessage
	//
	// 📝 包含：
	//    - Role: "assistant"
	//    - Content: AI的回复文本
	//    - ToolCalls: 可能包含的工具调用
	Message ChatMessage `json:"message"`

	// FinishReason 字段：生成结束的原因
	// 类型：string
	//
	// 📝 可能的值：
	//    "stop"       - 正常结束，AI完成了回复
	//    "length"     - 达到最大长度限制
	//    "tool_calls" - AI需要调用工具
	//    "content_filter" - 内容被过滤
	FinishReason string `json:"finish_reason"`
}

// Usage Token使用统计结构体
//
// 💡 用途：记录这次请求的Token消耗情况
//
// 💡 什么是Token？
//
//	Token是AI模型处理文本的基本单位
//	大约：1个Token ≈ 0.75个英文单词 ≈ 1个汉字
//	例如："Hello, world!" 大约是4个Token
type Usage struct {
	// PromptTokens 字段：输入（提示词）消耗的Token数
	// 类型：int（整数）
	//
	// 📝 包括：系统提示 + 所有历史消息 + 当前用户消息
	//    例如：你发送了100字的消息，可能消耗约150个Token
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens 字段：AI回复（输出）消耗的Token数
	// 类型：int
	//
	// 📝 AI生成的回复内容消耗的Token
	//    例如：AI回复了50字，可能消耗约75个Token
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens 字段：总计消耗的Token数
	// 类型：int
	//
	// 📝 计算公式：TotalTokens = PromptTokens + CompletionTokens
	//    这是本次请求的总成本，用于计费
	TotalTokens int `json:"total_tokens"`
}

// ============================================================================
// 第三部分：工具调用相关的数据结构
// ============================================================================

// Tool 工具定义结构体
//
// 💡 用途：定义一个AI可以调用的工具（函数）
//
// 📝 示例：定义一个"查询天气"的工具
type Tool struct {
	// Type 字段：工具类型
	// 类型：string
	//
	// 📝 当前固定值为 "function"
	//    未来可能支持其他类型（如 "code_interpreter"）
	Type string `json:"type"`

	// Function 字段：函数的详细定义
	// 类型：FunctionDef结构体
	//
	// 📝 包含函数名、描述、参数定义等
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义结构体
//
// 💡 用途：描述一个函数的元数据（不是函数本身，是函数的"说明书"）
//
// 📝 告诉AI：
//   - 这个函数叫什么名字
//   - 这个函数是做什么的
//   - 需要什么参数
type FunctionDef struct {
	// Name 字段：函数名称
	// 类型：string
	//
	// 📝 示例："get_weather", "calculate", "search"
	//    命名规范：通常用动词开头，描述函数的动作
	Name string `json:"name"`

	// Description 字段：函数功能的文字描述
	// 类型：string
	//
	// 📝 示例："获取指定城市的实时天气信息"
	//    这个描述会被AI看到，帮助AI决定何时调用这个函数
	//
	// 💡 描述越详细越好，包括：
	//    - 函数做什么
	//    - 什么时候使用
	//    - 需要什么参数
	Description string `json:"description"`

	// Parameters 字段：函数参数的JSON Schema定义
	// 类型：map[string]interface{}
	//
	// 💡 interface{} 是Go的"任意类型"
	//    map[string]interface{} 表示键为string，值可以是任意类型的map
	//
	// 📝 定义参数的结构、类型、是否必需等
	//    示例：
	//    {
	//      "type": "object",
	//      "properties": {
	//        "city": {"type": "string", "description": "城市名称"}
	//      },
	//      "required": ["city"]
	//    }
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolCall 工具调用结构体
//
// 💡 用途：表示AI发起的一次具体工具调用
//
// 📝 当AI决定使用工具时，会生成这个结构
type ToolCall struct {
	// ID 字段：这次调用的唯一标识符
	// 类型：string
	//
	// 📝 示例："call_abc123"
	//    用于关联工具调用和后续的工具结果
	ID string `json:"id"`

	// Type 字段：调用类型
	// 类型：string
	//
	// 📝 当前固定值为 "function"
	Type string `json:"type"`

	// Function 字段：具体的函数调用信息
	// 类型：FunctionCall结构体
	//
	// 📝 包含：要调用的函数名、传入的参数
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用结构体
//
// 💡 用途：包含函数调用的具体信息
type FunctionCall struct {
	// Name 字段：要调用的函数名
	// 类型：string
	//
	// 📝 必须匹配已注册的某个工具函数名
	//    示例："get_weather"
	Name string `json:"name"`

	// Arguments 字段：函数参数的JSON字符串
	// 类型：string
	//
	// 📝 示例：'{"city": "北京", "unit": "celsius"}'
	//    注意：这是字符串，需要解析后才能使用
	//
	// 💡 为什么是字符串不是对象？
	//    因为JSON序列化时，任意对象都会转为字符串
	//    使用时需要用 json.Unmarshal() 解析
	Arguments string `json:"arguments"`
}

// ============================================================================
// 第四部分：辅助函数（工具函数）
// ============================================================================

// BuildRequest 构建请求对象并序列化为JSON
//
// 💡 这是一个辅助函数，简化请求构建过程
//
// 📝 参数说明：
//
//	model   - 模型名称（如 "gpt-3.5-turbo"）
//	messages - 对话消息列表
//	stream  - 是否流式（true/false）
//	tools   - 可用工具列表
//
// 📝 返回值：
//
//	[]byte  - JSON字节数组（序列化后的数据）
//	error   - 错误信息（如果序列化失败）
//
// 📝 使用示例：
//
//	data, err := BuildRequest("gpt-3.5-turbo", messages, true, tools)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// 现在data是[]byte类型，可以直接发送给API
func BuildRequest(model string, messages []ChatMessage, stream bool, tools []Tool) ([]byte, error) {
	// 步骤1: 创建ChatRequest结构体并赋值
	req := ChatRequest{
		Model:    model,    // 设置模型
		Messages: messages, // 设置消息列表
		Stream:   stream,   // 设置是否流式
		Tools:    tools,    // 设置工具列表
	}

	// 步骤2: 将结构体序列化为JSON字节数组
	// json.Marshal() 返回 ([]byte, error)
	// 例如：{"model":"gpt-3.5-turbo","messages":[...]}
	return json.Marshal(req)
}

// ParseResponse 解析JSON响应数据
//
// 💡 这是一个辅助函数，简化响应解析过程
//
// 📝 参数说明：
//
//	data - API返回的JSON字节数组
//
// 📝 返回值：
//
//	*ChatResponse - 解析后的响应结构体指针
//	error         - 错误信息（如果解析失败）
//
// 📝 使用示例：
//
//	resp, err := ParseResponse(jsonData)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Choices[0].Message.Content)
//
// 💡 为什么要返回指针 (*ChatResponse)？
//
//	指针更高效，避免复制整个结构体
//	结构体可能很大，复制指针（8字节）比复制结构体快得多
func ParseResponse(data []byte) (*ChatResponse, error) {
	// 步骤1: 创建一个空的ChatResponse结构体
	var resp ChatResponse
	// var 关键字声明变量
	// ChatResponse{} 是零值（所有字段都是默认值）

	// 步骤2: 将JSON数据解析到结构体中
	// &resp 表示resp的内存地址（指针）
	// json.Unmarshal会填充这个地址指向的结构体
	err := json.Unmarshal(data, &resp)

	// 步骤3: 检查是否有错误
	if err != nil {
		// 如果解析失败，返回nil和错误信息
		return nil, err
	}

	// 步骤4: 返回解析结果
	// &resp 取resp的地址（指针）
	// nil 表示没有错误
	return &resp, nil
}

// ============================================================================
// 📚 给Go小白的知识点总结
// ============================================================================

/*
1️⃣ 结构体（Struct）
   - 结构体是相关数据的集合
   - 类似其他语言的"类"或"对象"
   - 例如：Person结构体包含姓名、年龄等属性

2️⃣ 结构体标签（Struct Tag）
   - 格式：`json:"字段名"`
   - 告诉JSON库如何序列化/反序列化
   - omitempty: 如果为空则省略

3️⃣ 切片（Slice）
   - Go的动态数组
   - 可以自动增长
   - []Type表示Type类型的切片

4️⃣ 指针（Pointer）
   - 存储内存地址
   - *Type表示Type类型的指针
   - &var 取var的地址
   - *ptr 解引用（获取指针指向的值）

5️⃣ interface{}
   - Go的"任意类型"
   - 可以存储任何值
   - 类似Java的Object或C++的void*

6️⃣ map
   - 键值对集合
   - map[K]V表示键类型为K，值类型为V的map

7️⃣ 返回值
   - Go支持多返回值
   - 通常最后一个是error类型
   - nil表示"无"或"空"

8️⃣ 变量声明
   - var name string  // 声明变量
   - name := "value"  // 短变量声明（自动推断类型）

9️⃣ 函数定义
   - func FuncName(param Type) (ReturnType, error) { ... }
   - 如果有多个返回值，用括号括起来

🔟 JSON处理
   - json.Marshal(v)    - Go对象 → JSON
   - json.Unmarshal(data, &v) - JSON → Go对象
*/
