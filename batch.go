package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ==================== 批量分析配置 ====================

// BatchConfig 批量分析配置
type BatchConfig struct {
	MaxConcurrency int           // 最大并发数
	RequestDelay   time.Duration // 请求间隔
	RetryCount     int           // 重试次数
	Timeout        time.Duration // 超时时间
	SkipDuplicates bool          // 是否跳过重复
	UpdateOnly     bool          // 是否只更新已有报告
}

// DefaultBatchConfig 默认配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxConcurrency: 3,                // 最多3个并发
		RequestDelay:   2 * time.Second,  // 每个请求间隔2秒
		RetryCount:     2,                // 失败重试2次
		Timeout:        60 * time.Second, // 单个分析超时60秒
		SkipDuplicates: true,             // 跳过重复内容
		UpdateOnly:     false,            // 不限制只更新
	}
}

// ==================== 批量分析器 ====================

// BatchAnalyzer 批量分析器
type BatchAnalyzer struct {
	config           *BatchConfig
	apiKey           string
	baseURL          string
	model            string
	token            string
	reportManager    *ReportManager
	duplicateChecker *DuplicateChecker
	progress         *ProgressTracker
	rateLimiter      *RateLimiter
}

// NewBatchAnalyzer 创建批量分析器
func NewBatchAnalyzer(apiKey, baseURL, model, token string, config *BatchConfig) (*BatchAnalyzer, error) {
	if config == nil {
		config = DefaultBatchConfig()
	}

	// 创建报告管理器
	rm := NewReportManager("reports")

	// 创建去重检查器
	dc, err := NewDuplicateChecker("data/duplicates.db")
	if err != nil {
		return nil, fmt.Errorf("创建去重检查器失败: %v", err)
	}

	// 创建进度跟踪器
	pt := &ProgressTracker{}

	// 创建限流器
	rl := &RateLimiter{
		delay: config.RequestDelay,
	}

	return &BatchAnalyzer{
		config:           config,
		apiKey:           apiKey,
		baseURL:          baseURL,
		model:            model,
		token:            token,
		reportManager:    rm,
		duplicateChecker: dc,
		progress:         pt,
		rateLimiter:      rl,
	}, nil
}

// BatchAnalysisResult 批量分析结果
type BatchAnalysisResult struct {
	TotalRepos int                    `json:"total_repos"`
	Analyzed   int                    `json:"analyzed"`
	Skipped    int                    `json:"skipped"`
	Failed     int                    `json:"failed"`
	Updated    int                    `json:"updated"`
	Duplicates int                    `json:"duplicates"`
	Duration   time.Duration          `json:"duration"`
	Results    []SingleAnalysisResult `json:"results"`
}

// SingleAnalysisResult 单个分析结果
type SingleAnalysisResult struct {
	RepoName      string        `json:"repo_name"`
	RepoURL       string        `json:"repo_url"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
	Action        string        `json:"action"` // "analyzed", "skipped", "updated", "failed"
	DuplicateType string        `json:"duplicate_type,omitempty"`
	Duration      time.Duration `json:"duration"`
}

// AnalyzeBatch 批量分析仓库
func (ba *BatchAnalyzer) AnalyzeBatch(repos []GitHubRepository) (*BatchAnalysisResult, error) {
	startTime := time.Now()

	result := &BatchAnalysisResult{
		TotalRepos: len(repos),
		Results:    make([]SingleAnalysisResult, 0, len(repos)),
	}

	// 初始化进度
	ba.progress.Init(len(repos))
	defer ba.progress.Finish()

	// 创建工作池
	jobs := make(chan GitHubRepository, len(repos))
	results := make(chan SingleAnalysisResult, len(repos))

	// 启动worker
	var wg sync.WaitGroup
	for i := 0; i < ba.config.MaxConcurrency; i++ {
		wg.Add(1)
		go ba.worker(jobs, results, &wg)
	}

	// 发送任务
	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	// 收集结果
	for i := 0; i < len(repos); i++ {
		r := <-results
		result.Results = append(result.Results, r)

		// 更新统计
		switch r.Action {
		case "analyzed":
			result.Analyzed++
		case "skipped":
			result.Skipped++
		case "updated":
			result.Updated++
		case "failed":
			result.Failed++
		}

		if r.DuplicateType != "" {
			result.Duplicates++
		}
	}

	// 等待所有worker完成
	wg.Wait()

	result.Duration = time.Since(startTime)

	// 打印统计
	ba.printStatistics(result)

	return result, nil
}

// worker 工作协程
func (ba *BatchAnalyzer) worker(jobs <-chan GitHubRepository, results chan<- SingleAnalysisResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for repo := range jobs {
		// 限流
		ba.rateLimiter.Wait()

		// 分析单个仓库
		result := ba.analyzeSingle(repo)
		results <- result

		// 更新进度
		ba.progress.Increment()
	}
}

// analyzeSingle 分析单个仓库
func (ba *BatchAnalyzer) analyzeSingle(repo GitHubRepository) SingleAnalysisResult {
	startTime := time.Now()

	result := SingleAnalysisResult{
		RepoName: repo.FullName,
		RepoURL:  repo.URL,
		Success:  false,
		Action:   "failed",
	}

	// 1. 获取POC文件
	fmt.Printf("\n📂 [%d/%d] 正在分析: %s\n", ba.progress.Current()+1, ba.progress.Total, repo.FullName)

	pocFiles, err := GetPOCFiles(repo.Owner.Login, repo.Name, ba.token)
	if err != nil {
		result.Error = fmt.Sprintf("获取POC文件失败: %v", err)
		return result
	}

	// 2. 计算hashes用于去重
	hashes := ba.calculateHashes(repo, pocFiles)

	// 3. 检查CVE编号
	cveID := ba.extractCVEID(repo)

	// 4. 去重检查
	if ba.config.SkipDuplicates {
		checkResult, err := ba.duplicateChecker.CheckDuplicate(repo, hashes, cveID)
		if err != nil {
			fmt.Printf("⚠️  去重检查失败: %v\n", err)
		} else if checkResult.IsDuplicate {
			result.DuplicateType = checkResult.DuplicateType

			// 判断是否需要更新
			if !checkResult.ShouldAnalyze && ba.config.UpdateOnly {
				result.Action = "skipped"
				result.Success = true
				fmt.Printf("⏭️  跳过: %s (%s)\n", repo.FullName, checkResult.Reason)
				result.Duration = time.Since(startTime)
				return result
			}
		}
	}

	// 5. 调用AI分析
	fmt.Printf("🤖 AI分析中...\n")
	aiResult, err := ba.analyzeWithRetry(repo)
	if err != nil {
		result.Error = fmt.Sprintf("AI分析失败: %v", err)
		fmt.Printf("❌ %s\n", result.Error)
		result.Duration = time.Since(startTime)
		return result
	}

	// 6. 保存报告
	if err := ba.reportManager.SaveAnalysisReport(repo, aiResult, pocFiles); err != nil {
		result.Error = fmt.Sprintf("保存报告失败: %v", err)
		fmt.Printf("❌ %s\n", result.Error)
		result.Duration = time.Since(startTime)
		return result
	}

	// 7. 添加到去重数据库
	// 注意：这里简化处理，实际应该从保存的报告中获取
	// 报告已经在SaveAnalysisReport中保存了
	// duplicateChecker.AddRecord会在保存报告后自动调用

	result.Success = true
	result.Action = "analyzed"
	result.Duration = time.Since(startTime)

	fmt.Printf("✅ 分析完成: %s (耗时: %.1fs)\n", repo.FullName, result.Duration.Seconds())

	return result
}

// analyzeWithRetry 带重试的分析
func (ba *BatchAnalyzer) analyzeWithRetry(repo GitHubRepository) (string, error) {
	var lastErr error

	for i := 0; i < ba.config.RetryCount; i++ {
		if i > 0 {
			fmt.Printf("🔄 重试 %d/%d...\n", i, ba.config.RetryCount)
			time.Sleep(time.Duration(i) * time.Second)
		}

		result, err := AnalyzeRepoWithAI(ba.apiKey, ba.baseURL, ba.model, ba.token, repo)
		if err == nil {
			return result, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("重试 %d 次后仍然失败: %v", ba.config.RetryCount, lastErr)
}

// calculateHashes 计算hash
func (ba *BatchAnalyzer) calculateHashes(repo GitHubRepository, files map[string]string) ReportHashes {
	return ba.reportManager.CalculateHashes(repo, files)
}

// extractCVEID 提取CVE编号
func (ba *BatchAnalyzer) extractCVEID(repo GitHubRepository) string {
	cveInfo := ba.reportManager.ExtractCVEInfo(repo, AIAnalysisResult{})
	return cveInfo.CVEID
}

// printStatistics 打印统计信息
func (ba *BatchAnalyzer) printStatistics(result *BatchAnalysisResult) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📊 批量分析统计")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("总仓库数:   %d\n", result.TotalRepos)
	fmt.Printf("✅ 成功分析: %d\n", result.Analyzed)
	fmt.Printf("🔄 更新已有: %d\n", result.Updated)
	fmt.Printf("⏭️  跳过重复: %d\n", result.Skipped)
	fmt.Printf("❌ 失败:     %d\n", result.Failed)
	fmt.Printf("⏱️  总耗时:   %.1f秒\n", result.Duration.Seconds())
	if result.Analyzed+result.Updated > 0 {
		fmt.Printf("⚡ 平均速度: %.1f秒/个\n",
			result.Duration.Seconds()/float64(result.Analyzed+result.Updated))
	}
	fmt.Println(strings.Repeat("=", 70))
}

// Close 关闭分析器
func (ba *BatchAnalyzer) Close() error {
	return ba.duplicateChecker.Close()
}

// ==================== 进度跟踪器 ====================

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	total     int
	current   int
	mutex     sync.Mutex
	startTime time.Time
}

// Init 初始化
func (pt *ProgressTracker) Init(total int) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.total = total
	pt.current = 0
	pt.startTime = time.Now()
}

// Increment 增加进度
func (pt *ProgressTracker) Increment() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.current++
	pt.printProgress()
}

// Current 当前进度
func (pt *ProgressTracker) Current() int {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	return pt.current
}

// Total 总数
func (pt *ProgressTracker) Total() int {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	return pt.total
}

// printProgress 打印进度
func (pt *ProgressTracker) printProgress() {
	percentage := float64(pt.current) / float64(pt.total) * 100
	fmt.Printf("\r进度: [%d/%d] %.1f%%", pt.current, pt.total, percentage)
}

// Finish 完成
func (pt *ProgressTracker) Finish() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	fmt.Printf("\n✅ 完成! 总耗时: %.1f秒\n", time.Since(pt.startTime).Seconds())
}

// ==================== 限流器 ====================

// RateLimiter 限流器
type RateLimiter struct {
	delay    time.Duration
	lastTime time.Time
	mutex    sync.Mutex
}

// Wait 等待
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if rl.delay == 0 {
		return
	}

	// 计算需要等待的时间
	elapsed := time.Since(rl.lastTime)
	if elapsed < rl.delay {
		time.Sleep(rl.delay - elapsed)
	}

	rl.lastTime = time.Now()
}
