package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ==================== 报告数据结构 ====================

// CVEInfo CVE信息
type CVEInfo struct {
	CVEID string `json:"cve_id"` // 从仓库名称或描述中提取的CVE编号
}

// AnalysisReport 完整的分析报告
type AnalysisReport struct {
	// 元数据
	Metadata ReportMetadata `json:"metadata"`

	// 仓库信息
	Repository GitHubRepository `json:"repository"`

	// CVE信息
	CVE CVEInfo `json:"cve"`

	// AI分析结果
	Analysis AIAnalysisResult `json:"analysis"`

	// 文件信息
	Files []FileInfo `json:"files"`

	// Hash信息（用于去重）
	Hashes ReportHashes `json:"hashes"`
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	GeneratedAt   time.Time `json:"generated_at"`
	ReportVersion string    `json:"report_version"`
	Analyzer      string    `json:"analyzer"`
	Version       string    `json:"version"`
}

// AIAnalysisResult AI分析结果
type AIAnalysisResult struct {
	IsPOC                 bool     `json:"is_poc"`
	Confidence            string   `json:"confidence"`
	VulnerabilityType     string   `json:"vulnerability_type"`     // web_vulnerability/terminal_vulnerability/other
	VulnerabilityCategory string   `json:"vulnerability_category"` // 具体类型
	TargetPlatform        string   `json:"target_platform"`
	ExploitDifficulty     string   `json:"exploit_difficulty"`
	RiskLevel             string   `json:"risk_level"`
	Reason                string   `json:"reason"`
	KeyFeatures           []string `json:"key_features"`
	Recommendations       string   `json:"recommendations"`
	RawResponse           string   `json:"raw_response"` // AI原始响应
}

// FileInfo 文件信息
type FileInfo struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
	ContentHash string `json:"content_hash"` // SHA256
}

// ReportHashes 用于去重的hash信息
type ReportHashes struct {
	RepoHash     string `json:"repo_hash"`     // 仓库URL + Owner + Name
	ContentHash  string `json:"content_hash"`  // 主要文件内容的hash
	CombinedHash string `json:"combined_hash"` // 组合hash
}

// ==================== 报告管理器 ====================

// ReportManager 报告管理器
type ReportManager struct {
	ReportsDir string // 报告存储目录
	IndexFile  string // 索引文件路径
}

// NewReportManager 创建报告管理器
func NewReportManager(reportsDir string) *ReportManager {
	return &ReportManager{
		ReportsDir: reportsDir,
		IndexFile:  filepath.Join(reportsDir, "index.json"),
	}
}

// SaveAnalysisReport 保存分析报告
func (rm *ReportManager) SaveAnalysisReport(repo GitHubRepository, aiResult string, files map[string]string) error {
	// 1. 创建报告目录（按日期组织）
	dateDir := filepath.Join(rm.ReportsDir, time.Now().Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 2. 解析AI响应
	analysis, err := rm.parseAIResult(aiResult)
	if err != nil {
		return fmt.Errorf("解析AI结果失败: %v", err)
	}

	// 3. 构建文件信息
	fileInfos := rm.buildFileInfos(files)

	// 4. 计算hashes
	hashes := rm.calculateHashes(repo, files)

	// 5. 提取CVE编号
	cveInfo := rm.extractCVEInfo(repo, analysis)

	// 6. 构建完整报告
	report := AnalysisReport{
		Metadata: ReportMetadata{
			GeneratedAt:   time.Now(),
			ReportVersion: "1.0",
			Analyzer:      "AI-POC-Analyzer",
			Version:       "1.0.0",
		},
		Repository: repo,
		CVE:        cveInfo,
		Analysis:   analysis,
		Files:      fileInfos,
		Hashes:     hashes,
	}

	// 7. 生成文件名
	filename := rm.generateFilename(cveInfo.CVEID, repo.Owner.Login)
	reportPath := filepath.Join(dateDir, filename)

	// 8. 保存报告
	if err := rm.saveReportToFile(report, reportPath); err != nil {
		return err
	}

	// 9. 更新索引
	if err := rm.updateIndex(report, reportPath); err != nil {
		fmt.Printf("⚠️  更新索引失败: %v\n", err)
	}

	fmt.Printf("✅ 报告已保存: %s\n", reportPath)
	return nil
}

// parseAIResult 解析AI返回的JSON结果
func (rm *ReportManager) parseAIResult(aiResult string) (AIAnalysisResult, error) {
	// 尝试提取JSON部分（AI可能返回包裹在markdown代码块中的JSON）
	jsonStr := aiResult

	// 移除可能的markdown代码块标记
	if strings.HasPrefix(aiResult, "```json") {
		jsonStr = strings.TrimPrefix(aiResult, "```json")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	} else if strings.HasPrefix(aiResult, "```") {
		jsonStr = strings.TrimPrefix(aiResult, "```")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	}

	var result AIAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// 如果解析失败，返回原始结果
		fmt.Printf("⚠️  解析AI JSON失败，使用原始结果: %v\n", err)
		result.RawResponse = aiResult
		return result, nil
	}

	result.RawResponse = aiResult
	return result, nil
}

// buildFileInfos 构建文件信息
func (rm *ReportManager) buildFileInfos(files map[string]string) []FileInfo {
	var infos []FileInfo
	for path, content := range files {
		hash := sha256.Sum256([]byte(content))
		infos = append(infos, FileInfo{
			Path:        path,
			Name:        filepath.Base(path),
			Size:        len(content),
			ContentHash: hex.EncodeToString(hash[:]),
		})
	}
	return infos
}

// calculateHashes 计算各种hash用于去重（导出版本）
func (rm *ReportManager) CalculateHashes(repo GitHubRepository, files map[string]string) ReportHashes {
	return rm.calculateHashes(repo, files)
}

// calculateHashes 计算各种hash用于去重
func (rm *ReportManager) calculateHashes(repo GitHubRepository, files map[string]string) ReportHashes {
	// 1. 仓库hash：基于URL和owner
	repoHash := sha256.Sum256([]byte(repo.URL + repo.Owner.Login + repo.Name))

	// 2. 内容hash：基于主要文件内容
	var contentData string
	for path, content := range files {
		// 只包含代码文件，忽略README等
		if !strings.Contains(strings.ToLower(path), "readme") &&
			!strings.Contains(strings.ToLower(path), "license") {
			contentData += content
		}
	}
	contentHash := sha256.Sum256([]byte(contentData))

	// 3. 组合hash
	combinedData := string(repoHash[:]) + string(contentHash[:])
	combinedHash := sha256.Sum256([]byte(combinedData))

	return ReportHashes{
		RepoHash:     hex.EncodeToString(repoHash[:]),
		ContentHash:  hex.EncodeToString(contentHash[:]),
		CombinedHash: hex.EncodeToString(combinedHash[:]),
	}
}

// extractCVEInfo 提取CVE编号（导出版本）
func (rm *ReportManager) ExtractCVEInfo(repo GitHubRepository, analysis AIAnalysisResult) CVEInfo {
	return rm.extractCVEInfo(repo, analysis)
}

// extractCVEInfo 提取CVE编号
func (rm *ReportManager) extractCVEInfo(repo GitHubRepository, analysis AIAnalysisResult) CVEInfo {
	// 尝试从仓库名称中提取
	cveID := rm.extractCVEFromString(repo.Name)

	// 如果仓库名中没有，尝试从描述中提取
	if cveID == "" {
		cveID = rm.extractCVEFromString(repo.Description)
	}

	// 如果描述中也没有，尝试从AI分析结果中提取
	if cveID == "" {
		cveID = rm.extractCVEFromString(analysis.VulnerabilityCategory)
	}

	// 如果仍然没有，使用unknown
	if cveID == "" {
		cveID = "UNKNOWN"
	}

	return CVEInfo{
		CVEID: cveID,
	}
}

// extractCVEFromString 从字符串中提取CVE编号
func (rm *ReportManager) extractCVEFromString(s string) string {
	// CVE格式: CVE-YYYY-NNNNN
	if strings.Contains(s, "CVE-") {
		parts := strings.Split(s, "CVE-")
		for i := 1; i < len(parts); i++ {
			// 提取CVE编号（例如：CVE-2026-21858）
			cvePart := strings.TrimSpace(parts[i])
			// CVE编号通常是 CVE-YYYY-NNNNN 格式
			if len(cvePart) >= 9 {
				// 找到第一个空格或标点符号
				for j := 0; j < len(cvePart); j++ {
					if cvePart[j] < '0' || cvePart[j] > '9' {
						if j > 8 {
							return "CVE-" + cvePart[:j]
						}
						break
					}
				}
				// 如果没有找到分隔符，返回整个部分（但限制长度）
				if len(cvePart) > 20 {
					return "CVE-" + cvePart[:20]
				}
				return "CVE-" + cvePart
			}
		}
	}
	return ""
}

// generateFilename 生成报告文件名
func (rm *ReportManager) GenerateFilename(cveID, owner string) string {
	// 格式: CVE-YYYY-NNNNN_Owner.json
	timestamp := time.Now().Format("20060102_150405")
	if cveID == "UNKNOWN" {
		return fmt.Sprintf("UNKNOWN_%s_%s.json", owner, timestamp)
	}
	return fmt.Sprintf("%s_%s.json", cveID, owner)
}

// generateFilename 内部使用的生成文件名函数
func (rm *ReportManager) generateFilename(cveID, owner string) string {
	return rm.GenerateFilename(cveID, owner)
}

// saveReportToFile 保存报告到文件
func (rm *ReportManager) saveReportToFile(report AnalysisReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化报告失败: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// updateIndex 更新索引文件
func (rm *ReportManager) updateIndex(report AnalysisReport, reportPath string) error {
	// 加载现有索引
	index := make(map[string]string)
	if data, err := os.ReadFile(rm.IndexFile); err == nil {
		json.Unmarshal(data, &index)
	}

	// 添加新条目
	key := report.Hashes.CombinedHash
	index[key] = reportPath

	// 保存索引
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rm.IndexFile, data, 0644)
}

// ==================== 报告加载功能 ====================

// LoadAnalysisReport 加载分析报告
func (rm *ReportManager) LoadAnalysisReport(reportPath string) (*AnalysisReport, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("读取报告失败: %v", err)
	}

	var report AnalysisReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("解析报告失败: %v", err)
	}

	return &report, nil
}

// LoadAllReports 加载所有报告
func (rm *ReportManager) LoadAllReports() ([]AnalysisReport, error) {
	var reports []AnalysisReport

	// 遍历reports目录
	err := filepath.Walk(rm.ReportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和非json文件
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// 跳过索引文件
		if strings.HasSuffix(path, "index.json") {
			return nil
		}

		// 加载报告
		report, err := rm.LoadAnalysisReport(path)
		if err != nil {
			fmt.Printf("⚠️  加载报告失败 %s: %v\n", path, err)
			return nil
		}

		reports = append(reports, *report)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return reports, nil
}

// FindReportByCVE 根据CVE编号查找报告
func (rm *ReportManager) FindReportByCVE(cveID string) ([]AnalysisReport, error) {
	allReports, err := rm.LoadAllReports()
	if err != nil {
		return nil, err
	}

	var matches []AnalysisReport
	for _, report := range allReports {
		if report.CVE.CVEID == cveID {
			matches = append(matches, report)
		}
	}

	return matches, nil
}

// FindReportByRepo 根据仓库URL查找报告
func (rm *ReportManager) FindReportByRepo(repoURL string) (*AnalysisReport, error) {
	allReports, err := rm.LoadAllReports()
	if err != nil {
		return nil, err
	}

	for _, report := range allReports {
		if report.Repository.URL == repoURL {
			return &report, nil
		}
	}

	return nil, fmt.Errorf("未找到仓库报告: %s", repoURL)
}
