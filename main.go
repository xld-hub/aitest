package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ==================== 示例：基础HTTP请求 ====================

// sendChatRequest 发送普通的（非流式）聊天请求
// 这是最基础的API调用示例
func sendChatRequest(apiKey, baseURL, model, userMessage string) (*ChatResponse, error) {
	// 1. 构建请求体
	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Stream: false, // 非流式
	}

	// 2. 序列化为JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("JSON编码失败: %v", err)
	}

	// 3. 创建HTTP请求
	url := baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 4. 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 5. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 6. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 7. 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 8. 解析JSON响应
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &chatResp, nil
}

// ==================== 示例：流式响应处理 ====================

// sendStreamChatRequest 发送流式聊天请求
// 演示如何处理Server-Sent Events (SSE)格式的响应
func sendStreamChatRequest(apiKey, baseURL, model, userMessage string) error {
	// 1. 构建请求体
	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Stream: true, // 启用流式
	}

	// 2. 序列化
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	// 3. 创建请求
	url := baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 4. 设置请求头（注意Accept头）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream") // 告诉服务器我们期望SSE格式

	// 5. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 6. 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 7. 读取流式响应（SSE格式）
	fmt.Println("流式响应:")
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式：每行以 "data:" 开头
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		// 检查结束标记
		if line == "data: [DONE]" {
			break
		}

		// 提取JSON数据
		jsonStr := strings.TrimPrefix(line, "data:")
		jsonStr = strings.TrimSpace(jsonStr)

		// 解析每一行的JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			// 跳过无法解析的行
			continue
		}

		// 8. 提取并显示内容
		if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok {
						fmt.Print(content) // 实时打印
					}
				}
			}
		}
	}
	fmt.Println() // 换行

	return nil
}

// ==================== 示例：多轮对话 ====================

// multiTurnConversation 多轮对话示例
// 演示如何维护对话历史
// proxyAddr: 代理地址，如 "http://127.0.0.1:7890"，为空则不使用代理
func multiTurnConversation(apiKey, baseURL, model, proxyAddr string) error {
	// ========== 创建带代理的HTTP客户端 ==========
	// 步骤1: 解析代理URL
	var client *http.Client
	if proxyAddr != "" {
		// 有代理：创建自定义Transport
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			return fmt.Errorf("解析代理URL失败: %v", err)
		}

		// 步骤2: 创建Transport，设置代理
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL), // ← 设置代理函数
			// 跳过TLS证书验证（解决代理证书问题）
			// ⚠️  注意：这会降低安全性，仅在开发环境使用
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // ← 跳过证书验证
			},
		}

		// 步骤3: 创建Client，使用自定义Transport
		client = &http.Client{
			Transport: transport,
		}

		fmt.Printf("✅ 使用代理: %s\n", proxyAddr)
		fmt.Println("⚠️  已跳过TLS证书验证（仅开发环境）\n")
	} else {
		// 无代理：使用默认Client
		client = &http.Client{}
		fmt.Println("ℹ️  未设置代理，直连API\n")
	}
	// ==============================================

	// 对话历史
	messages := []ChatMessage{
		{Role: "system", Content: "你是一个知识渊博的老师。"},
	}

	questions := []string{
		"什么是HTTP?",
		"我上一个问题是什么",
	}

	for i, question := range questions {
		fmt.Printf("\n=== 第%d轮对话 ===\n", i+1)
		fmt.Printf("问题: %s\n", question)

		// 添加用户消息
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: question,
		})

		// 构建请求
		reqBody := ChatRequest{
			Model:    model,
			Messages: messages, // 包含历史消息
			Stream:   false,
		}

		jsonData, _ := json.Marshal(reqBody)

		// 发送请求
		url := baseURL + "/chat/completions"
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// ========== 使用带代理的client发送请求 ==========
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("请求失败: %v", err)
		}
		// ==============================================

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var respData ChatResponse
		json.Unmarshal(body, &respData)

		// 显示回答
		if len(respData.Choices) > 0 {
			answer := respData.Choices[0].Message.Content
			fmt.Printf("回答: %s\n", answer)

			// 添加助手回答到历史
			messages = append(messages, ChatMessage{
				Role:    "assistant",
				Content: answer,
			})
		}
	}

	return nil
}

// ==================== 主函数 ====================

func main() {
	// 检查是否有命令行参数
	if len(os.Args) > 1 {
		// 命令行模式
		RunCLI()
		return
	}

	// 默认演示模式
	runDemoMode()
}

// runDemoMode 运行演示模式
func runDemoMode() {
	// 配置参数
	apiKey := ""
	baseURL := "https://open.bigmodel.cn/api/coding/paas/v4"
	model := "glm-4.7"
	seen := make(map[string]bool)
	seen["1"] = true
	fmt.Println(seen)
	fmt.Printf("API地址: %s\n", baseURL)
	fmt.Printf("模型: %s\n\n", model)

	//// 测试消息
	//testMessage := "你好，请用一句话介绍一下Go语言。"
	//
	//// ========== 测试1: 普通请求 ==========
	//fmt.Println("【测试1: 普通请求】")
	//fmt.Printf("发送消息: %s\n", testMessage)
	//
	//response, err := sendChatRequest(apiKey, baseURL, model, testMessage)
	//if err != nil {
	//	fmt.Printf("❌ 错误: %v\n\n", err)
	//} else {
	//	fmt.Printf("✅ 成功!\n")
	//	fmt.Printf("响应ID: %s\n", response.ID)
	//	if len(response.Choices) > 0 {
	//		fmt.Printf("回复: %s\n", response.Choices[0].Message.Content)
	//	}
	//	if response.Usage.TotalTokens > 0 {
	//		fmt.Printf("令牌使用: %d (提示) + %d (完成) = %d (总计)\n",
	//			response.Usage.PromptTokens,
	//			response.Usage.CompletionTokens,
	//			response.Usage.TotalTokens)
	//	}
	//	fmt.Println()
	//}

	//// ========== 测试2: 流式请求 ==========
	//fmt.Println("【测试2: 流式请求】")
	//fmt.Printf("发送消息: %s\n", testMessage)
	//
	//err := sendStreamChatRequest(apiKey, baseURL, model, testMessage)
	//if err != nil {
	//	fmt.Printf("❌ 错误: %v\n", err)
	//}
	//fmt.Println()

	// ========== 测试3: 浏览器搜索工具 ==========
	//browserSearchExample() // 取消注释运行浏览器搜索示例

	// ========== 测试4: GitHub搜索CVE-2026 ==========
	SearchCVE2026Demo(apiKey, baseURL, model) // 运行CVE-2026搜索演示

	// ========== 测试5: 多轮对话（已注释）==========
	// fmt.Println("【测试3: 多轮对话（带代理开关）】")
	// fmt.Println("演示如何维护对话历史，让AI记住上下文...\n")
	// ...（注释掉的代理配置代码）
}

// ==================== GitHub Search API - CVE-2026搜索 ====================

// GitHubSearchResult GitHub搜索结果
type GitHubSearchResult struct {
	TotalCount int                `json:"total_count"` // 总结果数
	Incomplete bool               `json:"incomplete"`  // 是否结果不完整
	Items      []GitHubRepository `json:"items"`       // 仓库列表
}

// GitHubRepository GitHub仓库信息
type GitHubRepository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description string `json:"description"`
	URL         string `json:"html_url"`
	Language    string `json:"language"`
	Forks       int    `json:"forks"`
	Stars       int    `json:"stargazers_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// SearchCVE2026 搜索CVE-2026相关的GitHub仓库
// 参数：
//   - token: GitHub Personal Access Token（可选，为空则每小时限制60次）
//   - query: 搜索查询，例如 "CVE-2026 POC"
//   - sortBy: 排序方式 (stars, forks, updated)
//   - page: 页码（从1开始）
//
// 返回：搜索结果和错误
func SearchCVE2026(token, query, sortBy string, page int) (*GitHubSearchResult, error) {
	// GitHub Search API endpoint
	searchURL := "https://api.github.com/search/repositories"

	// 构建查询参数
	params := url.Values{}

	// 添加搜索查询
	if query == "" {
		query = "CVE-2026" // 默认搜索
	}

	// 添加高级筛选
	searchQuery := fmt.Sprintf("%s language:python stars:>5", query)
	params.Add("q", searchQuery)

	// 排序
	if sortBy == "" {
		sortBy = "stars" // 默认按stars排序
	}
	params.Add("sort", sortBy)
	params.Add("order", "desc") // 降序

	// 分页
	params.Add("per_page", "100") // 每页最多100个结果
	if page < 1 {
		page = 1
	}
	params.Add("page", fmt.Sprintf("%d", page))

	// 完整URL
	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// 创建HTTP请求
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result GitHubSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// FormatGitHubRepository 格式化仓库信息
func FormatGitHubRepository(repo GitHubRepository) string {
	return fmt.Sprintf(
		"📌 %s\n"+
			"   描述: %s\n"+
			"   语言: %s | ⭐ %d | 🍴 %d\n"+
			"   URL: %s\n",
		repo.FullName,
		truncateString(repo.Description, 100),
		repo.Language,
		repo.Stars,
		repo.Forks,
		repo.URL,
	)
}

// truncateString 辅助函数：截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GitHubFile GitHub文件内容
type GitHubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`    // "file" or "dir"
	Content     string `json:"content"` // Base64编码的文件内容
	Encoding    string `json:"encoding"`
	Size        int    `json:"size"`
	DownloadURL string `json:"download_url"`
}

// GetRepoFiles 获取GitHub仓库的文件列表（仅顶层目录和文件）
func GetRepoFiles(owner, repoName, token string) ([]GitHubFile, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/", owner, repoName)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var contents []GitHubFile
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, err
	}

	return contents, nil
}

// GetFileContent 获取GitHub文件的实际内容
func GetFileContent(owner, repoName, path, token string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repoName, path)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &fileContent); err != nil {
		return "", err
	}

	// GitHub API返回的是Base64编码的内容
	// 这里直接返回，让调用方决定是否需要解码
	return fileContent.Content, nil
}

// GetPOCFiles 获取仓库中可能包含POC代码的文件内容
func GetPOCFiles(owner, repoName, token string) (map[string]string, error) {
	files, err := GetRepoFiles(owner, repoName, token)
	if err != nil {
		return nil, err
	}

	pocFiles := make(map[string]string)
	pocExtensions := []string{".py", ".js", ".go", ".c", ".cpp", ".java", ".rb", ".php", ".sh", ".ps1", ".yml", ".yaml"}

	for _, file := range files {
		// 跳过目录
		if file.Type == "dir" {
			continue
		}

		// 检查是否是代码文件
		isCodeFile := false
		for _, ext := range pocExtensions {
			if strings.HasSuffix(strings.ToLower(file.Name), ext) {
				isCodeFile = true
				break
			}
		}

		// 或者是特定的POC相关文件名
		isPOCFile := strings.Contains(strings.ToLower(file.Name), "poc") ||
			strings.Contains(strings.ToLower(file.Name), "exploit") ||
			strings.Contains(strings.ToLower(file.Name), "payload") ||
			file.Name == "README.md"

		if isCodeFile || isPOCFile {
			// 限制文件大小，避免获取过大的文件
			if file.Size > 100000 { // 大于100KB的文件跳过
				fmt.Printf("  ⏭️  跳过过大文件: %s (%d bytes)\n", file.Name, file.Size)
				continue
			}

			fmt.Printf("  📥 正在获取文件: %s (%d bytes)\n", file.Name, file.Size)
			content, err := GetFileContent(owner, repoName, file.Path, token)
			if err != nil {
				fmt.Printf("  ⚠️  获取文件失败 %s: %v\n", file.Name, err)
				continue
			}

			// 存储文件路径和内容
			pocFiles[file.Path] = content
		}
	}

	return pocFiles, nil
}

// AnalyzeRepoWithAI 使用AI分析仓库是否包含可利用的POC，并识别漏洞类型
func AnalyzeRepoWithAI(apiKey, baseURL, model, token string, repo GitHubRepository) (string, error) {
	fmt.Println("  ══════════════════════════════════════════")
	fmt.Println("  🔍 开始AI深度分析流程（包含文件内容）")
	fmt.Println("  ══════════════════════════════════════════")

	// 步骤1: 获取仓库文件列表
	fmt.Println("  📂 步骤1: 获取仓库文件结构...")
	files, err := GetRepoFiles(repo.Owner.Login, repo.Name, token)
	var fileList string
	if err != nil {
		fmt.Printf("  ⚠️  获取文件列表失败: %v\n", err)
		fileList = "(无法获取文件列表)"
	} else {
		var fileNames []string
		for _, file := range files {
			icon := "📄"
			if file.Type == "dir" {
				icon = "📁"
			}
			fileNames = append(fileNames, fmt.Sprintf("%s %s", icon, file.Name))
		}
		fileList = strings.Join(fileNames, "\n    ")
		fmt.Printf("  ✅ 成功获取 %d 个文件/目录\n", len(files))
	}

	// 步骤2: 获取POC相关文件的内容
	fmt.Println("\n  📥 步骤2: 获取POC文件内容...")
	pocFiles, err := GetPOCFiles(repo.Owner.Login, repo.Name, token)
	var fileContents string
	if err != nil {
		fmt.Printf("  ⚠️  获取POC文件失败: %v\n", err)
		fileContents = "(无法获取文件内容)"
	} else if len(pocFiles) == 0 {
		fmt.Println("  ℹ️  未找到POC相关文件")
		fileContents = "(未找到POC相关文件)"
	} else {
		fmt.Printf("  ✅ 成功获取 %d 个文件的内容\n", len(pocFiles))
		// 限制内容长度，避免超过token限制
		totalContent := ""
		for filePath, base64Content := range pocFiles {
			// 解码Base64内容
			decodedContent, err := base64.StdEncoding.DecodeString(base64Content)
			if err != nil {
				fmt.Printf("  ⚠️  解码文件失败 %s: %v\n", filePath, err)
				continue
			}

			// 截断过长的文件内容
			content := string(decodedContent)
			if len(content) > 3000 {
				content = content[:3000] + "\n...[文件内容已截断]"
			}

			totalContent += fmt.Sprintf("\n    === 文件: %s ===\n    %s\n", filePath, content)
		}
		fileContents = totalContent
	}

	// 步骤3: 构建AI提示词
	fmt.Println("\n  📝 步骤3: 构建AI深度分析提示词...")
	prompt := fmt.Sprintf(`你是一个网络安全专家，专门分析漏洞利用代码（POC/Exploit）。请深度分析以下GitHub仓库。

【仓库基本信息】
- 仓库名称: %s
- 仓库描述: %s
- 编程语言: %s
- Stars: %d | Forks: %d
- 仓库地址: %s

【仓库文件列表】
%s

【POC相关文件内容】
%s

【分析任务】
请从以下几个维度进行深度分析：

1. **POC真实性判断**
   - 是否包含真正可执行的漏洞利用代码？
   - 是否有完整的利用步骤和清晰的文档说明？
   - 代码质量和可维护性如何？

2. **漏洞类型识别**（重要！）
   请明确判断这是哪类漏洞：

   a) **Web漏洞**（通过网络请求触发的漏洞）：
      - SQL注入、XSS、CSRF、命令注入、文件上传、反序列化等
      - 特征：包含HTTP请求代码、URL参数、表单数据、Cookie操作等
      - 典型关键词：http://、https://、requests.post、urllib、curl、wget

   b) **终端/本地漏洞**（在本地系统上执行的漏洞）：
      - 本地提权、缓冲区溢出、竞态条件、内核漏洞等
      - 特征：包含系统调用、内存操作、进程操作、文件系统操作等
      - 典型关键词：os.system、subprocess、exec、malloc、memcpy、ptrace

   c) **其他类型**：
      - 信息泄露、配置错误、逻辑缺陷等

3. **可利用性评估**
   - 漏洞利用的难易程度（简单/中等/困难）
   - 是否需要特殊条件或环境
   - 影响范围和危害程度

4. **安全建议**
   - 防御措施
   - 检测方法
   - 修复建议

【输出格式】
请严格按照以下JSON格式返回分析结果（不要输出其他内容）：

{
  "is_poc": true/false,
  "confidence": "high/medium/low",
  "vulnerability_type": "web_vulnerability/terminal_vulnerability/other",
  "vulnerability_category": "具体漏洞类型（如：SQL注入、本地提权、缓冲区溢出等）",
  "target_platform": "攻击目标平台（如：Web应用、Linux系统、Windows系统等）",
  "exploit_difficulty": "easy/medium/hard",
  "risk_level": "critical/high/medium/low",
  "reason": "判断理由（2-3句话，说明为何判断为该类型漏洞）",
  "key_features": [
    "特征1：具体描述",
    "特征2：具体描述",
    "特征3：具体描述"
  ],
  "recommendations": "防御和修复建议"
}

注意：
- 如果文件内容过长，请基于已有信息进行最佳判断
- vulnrability_type必须明确选择web_vulnerability、terminal_vulnerability或other之一
- 如果是Web漏洞，请在vulnerability_category中具体说明（如：SQL注入、XSS、RCE等）
- 如果是终端漏洞，请说明具体类型（如：本地提权、内核漏洞、缓冲区溢出等）`,
		repo.FullName,
		repo.Description,
		repo.Language,
		repo.Stars,
		repo.Forks,
		repo.URL,
		fileList,
		truncateString(fileContents, 8000)) // 限制内容长度

	fmt.Printf("  ✅ 提示词长度: %d 字符\n", len(prompt))
	fmt.Printf("  📋 提示词预览 (前300字符):\n     %s...\n", truncateString(prompt, 300))

	// 步骤4: 调用AI API
	fmt.Println("\n  🤖 步骤4: 调用AI API进行深度分析...")
	fmt.Printf("  🔗 API地址: %s\n", baseURL)
	fmt.Printf("  🎯 模型: %s\n", model)
	fmt.Println("  ⏳ 等待AI响应（这可能需要几秒钟）...")

	startTime := time.Now()
	response, err := sendChatRequest(apiKey, baseURL, model, prompt)
	if err != nil {
		fmt.Printf("  ❌ AI API调用失败: %v\n", err)
		return "", err
	}
	duration := time.Since(startTime)

	fmt.Printf("  ✅ AI响应成功 (耗时: %.2f秒)\n", duration.Seconds())

	// 步骤5: 解析响应
	fmt.Println("\n  📦 步骤5: 解析AI响应...")
	if len(response.Choices) == 0 {
		fmt.Println("  ❌ 错误: AI返回了空响应")
		return "", fmt.Errorf("no response from AI")
	}

	aiResult := response.Choices[0].Message.Content
	fmt.Printf("  ✅ 响应长度: %d 字符\n", len(aiResult))
	fmt.Printf("  📊 Token使用: 提入=%d, 输出=%d, 总计=%d\n",
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
		response.Usage.TotalTokens)

	// 步骤6: 显示分析结果
	fmt.Println("\n  📋 步骤6: AI深度分析结果")
	fmt.Println("  ══════════════════════════════════════════")
	return aiResult, nil
}

// SearchCVE2026Demo CVE-2026搜索演示
func SearchCVE2026Demo(apiKey, baseURL, model string) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           GitHub Search API - CVE-2026搜索                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// GitHub Token（可选）
	// 在 https://github.com/settings/tokens 生成
	// 推荐权限: public_repo
	token := "" // 留空则每小时限制60次搜索

	// ========== 测试1: 基础搜索 + AI分析 ==========
	fmt.Println("【测试1: CVE-2026搜索 + AI智能分析】")
	fmt.Println("搜索: CVE-2026 POC，Python语言，stars>5")
	fmt.Println()

	result, err := SearchCVE2026(token, "CVE-2026 POC", "stars", 1)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个仓库\n\n", result.TotalCount)

		// 显示前3个并进行AI分析
		analyzeCount := 3
		if len(result.Items) < analyzeCount {
			analyzeCount = len(result.Items)
		}

		fmt.Printf("🤖 正在使用AI分析前 %d 个仓库...\n\n", analyzeCount)

		for i := 0; i < analyzeCount; i++ {
			repo := result.Items[i]
			fmt.Printf("═══════════════════════════════════════════════════\n")
			fmt.Printf("【仓库 %d/%d】\n", i+1, analyzeCount)
			fmt.Printf("═══════════════════════════════════════════════════\n")
			fmt.Printf("%s\n\n", FormatGitHubRepository(repo))

			// 调用AI分析
			fmt.Println("🔍 AI分析中...")
			analysis, err := AnalyzeRepoWithAI(apiKey, baseURL, model, token, repo)
			if err != nil {
				fmt.Printf("❌ AI分析失败: %v\n\n", err)
			} else {
				fmt.Printf("📝 AI分析结果:\n%s\n\n", analysis)
			}

			// 避免API限流
			time.Sleep(1 * time.Second)
		}

		// 显示剩余的仓库（不做AI分析）
		if len(result.Items) > analyzeCount {
			displayCount := 10
			if len(result.Items) < displayCount {
				displayCount = len(result.Items)
			}

			fmt.Printf("═══════════════════════════════════════════════════\n")
			fmt.Printf("【其他仓库（未进行AI分析）】\n")
			fmt.Printf("═══════════════════════════════════════════════════\n\n")

			for i := analyzeCount; i < displayCount; i++ {
				repo := result.Items[i]
				fmt.Printf("[%d] %s\n", i+1, FormatGitHubRepository(repo))
			}

			if result.TotalCount > displayCount {
				fmt.Printf("\n...还有 %d 个仓库未显示\n", result.TotalCount-displayCount)
			}
		}
	}

	fmt.Println(strings.Repeat("-", 70))
	time.Sleep(2 * time.Second)

	// ========== 测试2: 搜索exploit ==========
	fmt.Println("\n【测试2: 搜索CVE-2026 exploit】")
	fmt.Println("搜索: CVE-2026 exploit，Python语言")
	fmt.Println()

	result, err = SearchCVE2026(token, "CVE-2026 exploit", "updated", 1)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个仓库（按更新时间排序）\n\n", result.TotalCount)

		displayCount := 5
		if len(result.Items) < displayCount {
			displayCount = len(result.Items)
		}

		for i := 0; i < displayCount; i++ {
			repo := result.Items[i]
			fmt.Printf("[%d] %s\n", i+1, repo.FullName)
			fmt.Printf("    描述: %s\n", truncateString(repo.Description, 80))
			fmt.Printf("    更新: %s\n", repo.UpdatedAt)
			fmt.Printf("    URL: %s\n\n", repo.URL)
		}
	}

	fmt.Println(strings.Repeat("-", 70))
	time.Sleep(2 * time.Second)

	// ========== 测试3: 高级搜索 ==========
	fmt.Println("\n【测试3: 高级筛选】")
	fmt.Println("搜索: CVE-2026，高级筛选条件")
	fmt.Println()

	advancedQuery := "CVE-2026 POC exploit language:python stars:>10 forks:>5"
	fmt.Printf("查询: %s\n\n", advancedQuery)

	result, err = SearchCVE2026(token, advancedQuery, "stars", 1)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n\n", err)
	} else {
		fmt.Printf("✅ 找到 %d 个高质量仓库\n\n", result.TotalCount)

		// 统计信息
		totalStars := 0
		totalForks := 0
		for _, repo := range result.Items {
			totalStars += repo.Stars
			totalForks += repo.Forks
		}

		fmt.Printf("📊 统计信息:\n")
		fmt.Printf("   总Stars: %d\n", totalStars)
		fmt.Printf("   总Forks: %d\n", totalForks)
		if len(result.Items) > 0 {
			fmt.Printf("   平均Stars: %.1f\n", float64(totalStars)/float64(len(result.Items)))
			fmt.Printf("   平均Forks: %.1f\n", float64(totalForks)/float64(len(result.Items)))
		}

		fmt.Println("\n🔝 Top 5 仓库:")
		displayCount := 5
		if len(result.Items) < displayCount {
			displayCount = len(result.Items)
		}

		for i := 0; i < displayCount; i++ {
			repo := result.Items[i]
			fmt.Printf("\n[%d] %s\n", i+1, FormatGitHubRepository(repo))
		}
	}

	fmt.Println("\n========== 搜索完成 ==========")
	fmt.Println()
	fmt.Println("💡 提示:")
	fmt.Println("1. 无Token: 每小时限制60次搜索")
	fmt.Println("2. 有Token: 每小时可搜索5000次")
	fmt.Println("3. Token生成: https://github.com/settings/tokens")
	fmt.Println("4. 推荐权限: public_repo")
	fmt.Println()
	fmt.Println("🔍 高级搜索语法:")
	fmt.Println("   • CVE-2026 POC              - 包含POC关键词")
	fmt.Println("   • language:python           - 限定Python")
	fmt.Println("   • stars:>10                 - Stars大于10")
	fmt.Println("   • forks:>5                  - Forks大于5")
	fmt.Println("   • CVE-2026 pushed:>2026-01-01 - 指定更新时间")
	fmt.Println("   • CVE-2026 -documentation    - 排除文档")
}
