package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ==================== 统计数据结构 ====================

// StatisticsReport 统计报告
type StatisticsReport struct {
	GeneratedAt  time.Time       `json:"generated_at"`
	Period       string          `json:"period"`
	TotalReports int             `json:"total_reports"`
	Summary      SummaryStats    `json:"summary"`
	VulnTypes    VulnTypeStats   `json:"vulnerability_types"`
	RiskLevels   RiskLevelStats  `json:"risk_levels"`
	TimeTrends   TimeTrendStats  `json:"time_trends"`
	TopCVEs      []TopCVEStat    `json:"top_cves"`
	Languages    LanguageStats   `json:"programming_languages"`
	Difficulties DifficultyStats `json:"exploit_difficulties"`
}

// SummaryStats 摘要统计
type SummaryStats struct {
	TotalAnalyzed int     `json:"total_analyzed"`
	TruePOCs      int     `json:"true_pocs"`
	POCRate       float64 `json:"poc_rate"`
	WebVulns      int     `json:"web_vulnerabilities"`
	TerminalVulns int     `json:"terminal_vulnerabilities"`
	OtherVulns    int     `json:"other_vulnerabilities"`
	CriticalRisk  int     `json:"critical_risk"`
	HighRisk      int     `json:"high_risk"`
}

// VulnTypeStats 漏洞类型统计
type VulnTypeStats struct {
	Web      int     `json:"web"`
	Terminal int     `json:"terminal"`
	Other    int     `json:"other"`
	WebRate  float64 `json:"web_rate"`
}

// RiskLevelStats 风险等级统计
type RiskLevelStats struct {
	Critical int            `json:"critical"`
	High     int            `json:"high"`
	Medium   int            `json:"medium"`
	Low      int            `json:"low"`
	Counts   map[string]int `json:"counts"`
}

// TimeTrendStats 时间趋势统计
type TimeTrendStats struct {
	Daily   map[string]int `json:"daily"`
	Weekly  map[string]int `json:"weekly"`
	Monthly map[string]int `json:"monthly"`
}

// TopCVEStat Top CVE统计
type TopCVEStat struct {
	CVEID      string  `json:"cve_id"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// LanguageStats 编程语言统计
type LanguageStats struct {
	TopLanguages []LanguageItem `json:"top_languages"`
	Counts       map[string]int `json:"counts"`
}

// LanguageItem 语言项
type LanguageItem struct {
	Language   string  `json:"language"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// DifficultyStats 利用难度统计
type DifficultyStats struct {
	Easy   int            `json:"easy"`
	Medium int            `json:"medium"`
	Hard   int            `json:"hard"`
	Counts map[string]int `json:"counts"`
}

// ==================== 统计收集器 ====================

// StatisticsCollector 统计收集器
type StatisticsCollector struct {
	reportManager *ReportManager
}

// NewStatisticsCollector 创建统计收集器
func NewStatisticsCollector(rm *ReportManager) *StatisticsCollector {
	return &StatisticsCollector{
		reportManager: rm,
	}
}

// GenerateStatistics 生成统计报告
func (sc *StatisticsCollector) GenerateStatistics(period string) (*StatisticsReport, error) {
	// 加载所有报告
	reports, err := sc.reportManager.LoadAllReports()
	if err != nil {
		return nil, fmt.Errorf("加载报告失败: %v", err)
	}

	// 过滤时间范围
	reports = sc.filterByPeriod(reports, period)

	// 生成报告
	report := &StatisticsReport{
		GeneratedAt:  time.Now(),
		Period:       period,
		TotalReports: len(reports),
		Summary:      sc.calculateSummary(reports),
		VulnTypes:    sc.calculateVulnTypes(reports),
		RiskLevels:   sc.calculateRiskLevels(reports),
		TimeTrends:   sc.calculateTimeTrends(reports),
		TopCVEs:      sc.calculateTopCVEs(reports),
		Languages:    sc.calculateLanguages(reports),
		Difficulties: sc.calculateDifficulties(reports),
	}

	return report, nil
}

// filterByPeriod 按时间范围过滤
func (sc *StatisticsCollector) filterByPeriod(reports []AnalysisReport, period string) []AnalysisReport {
	now := time.Now()
	var cutoff time.Time

	switch period {
	case "today":
		cutoff = now.Truncate(24 * time.Hour)
	case "week":
		cutoff = now.AddDate(0, 0, -7)
	case "month":
		cutoff = now.AddDate(0, -1, 0)
	case "year":
		cutoff = now.AddDate(-1, 0, 0)
	default:
		return reports
	}

	var filtered []AnalysisReport
	for _, report := range reports {
		if report.Metadata.GeneratedAt.After(cutoff) {
			filtered = append(filtered, report)
		}
	}

	return filtered
}

// calculateSummary 计算摘要统计
func (sc *StatisticsCollector) calculateSummary(reports []AnalysisReport) SummaryStats {
	summary := SummaryStats{}

	for _, report := range reports {
		summary.TotalAnalyzed++

		// 统计POC
		if report.Analysis.IsPOC {
			summary.TruePOCs++
		}

		// 统计漏洞类型
		switch report.Analysis.VulnerabilityType {
		case "web_vulnerability":
			summary.WebVulns++
		case "terminal_vulnerability":
			summary.TerminalVulns++
		default:
			summary.OtherVulns++
		}

		// 统计风险等级
		if report.Analysis.RiskLevel == "critical" {
			summary.CriticalRisk++
		} else if report.Analysis.RiskLevel == "high" {
			summary.HighRisk++
		}
	}

	if summary.TotalAnalyzed > 0 {
		summary.POCRate = float64(summary.TruePOCs) / float64(summary.TotalAnalyzed) * 100
	}

	return summary
}

// calculateVulnTypes 计算漏洞类型统计
func (sc *StatisticsCollector) calculateVulnTypes(reports []AnalysisReport) VulnTypeStats {
	stats := VulnTypeStats{}

	total := len(reports)
	for _, report := range reports {
		switch report.Analysis.VulnerabilityType {
		case "web_vulnerability":
			stats.Web++
		case "terminal_vulnerability":
			stats.Terminal++
		default:
			stats.Other++
		}
	}

	if total > 0 {
		stats.WebRate = float64(stats.Web) / float64(total) * 100
	}

	return stats
}

// calculateRiskLevels 计算风险等级统计
func (sc *StatisticsCollector) calculateRiskLevels(reports []AnalysisReport) RiskLevelStats {
	stats := RiskLevelStats{
		Counts: make(map[string]int),
	}

	for _, report := range reports {
		level := report.Analysis.RiskLevel
		stats.Counts[level]++

		switch level {
		case "critical":
			stats.Critical++
		case "high":
			stats.High++
		case "medium":
			stats.Medium++
		case "low":
			stats.Low++
		}
	}

	return stats
}

// calculateTimeTrends 计算时间趋势
func (sc *StatisticsCollector) calculateTimeTrends(reports []AnalysisReport) TimeTrendStats {
	trends := TimeTrendStats{
		Daily:   make(map[string]int),
		Weekly:  make(map[string]int),
		Monthly: make(map[string]int),
	}

	for _, report := range reports {
		date := report.Metadata.GeneratedAt

		// 按天统计
		dailyKey := date.Format("2006-01-02")
		trends.Daily[dailyKey]++

		// 按周统计
		year, week := date.ISOWeek()
		weeklyKey := fmt.Sprintf("%d-W%02d", year, week)
		trends.Weekly[weeklyKey]++

		// 按月统计
		monthlyKey := date.Format("2006-01")
		trends.Monthly[monthlyKey]++
	}

	return trends
}

// calculateTopCVEs 计算Top CVE
func (sc *StatisticsCollector) calculateTopCVEs(reports []AnalysisReport) []TopCVEStat {
	cveCounts := make(map[string]int)

	for _, report := range reports {
		cveID := report.CVE.CVEID
		if cveID != "" && cveID != "UNKNOWN" {
			cveCounts[cveID]++
		}
	}

	// 转换为数组并排序
	var stats []TopCVEStat
	total := len(reports)
	for cveID, count := range cveCounts {
		stats = append(stats, TopCVEStat{
			CVEID:      cveID,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}

	// 按数量降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// 只返回前10个
	if len(stats) > 10 {
		stats = stats[:10]
	}

	return stats
}

// calculateLanguages 计算编程语言统计
func (sc *StatisticsCollector) calculateLanguages(reports []AnalysisReport) LanguageStats {
	langCounts := make(map[string]int)

	for _, report := range reports {
		lang := report.Repository.Language
		if lang != "" {
			langCounts[lang]++
		}
	}

	// 转换为数组并排序
	var items []LanguageItem
	total := len(reports)
	for lang, count := range langCounts {
		items = append(items, LanguageItem{
			Language:   lang,
			Count:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}

	// 按数量降序排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	// 只返回前10个
	if len(items) > 10 {
		items = items[:10]
	}

	return LanguageStats{
		TopLanguages: items,
		Counts:       langCounts,
	}
}

// calculateDifficulties 计算利用难度统计
func (sc *StatisticsCollector) calculateDifficulties(reports []AnalysisReport) DifficultyStats {
	stats := DifficultyStats{
		Counts: make(map[string]int),
	}

	for _, report := range reports {
		difficulty := report.Analysis.ExploitDifficulty
		stats.Counts[difficulty]++

		switch difficulty {
		case "easy":
			stats.Easy++
		case "medium":
			stats.Medium++
		case "hard":
			stats.Hard++
		}
	}

	return stats
}

// ==================== 报告输出 ====================

// PrintReport 打印报告（人类可读格式）
func (sc *StatisticsCollector) PrintReport(report *StatisticsReport) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📊 POC分析统计报告")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("生成时间: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("统计周期: %s\n", report.Period)
	fmt.Printf("报告总数: %d\n\n", report.TotalReports)

	// 摘要
	fmt.Println("📈 摘要统计")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("总分析数:   %d\n", report.Summary.TotalAnalyzed)
	fmt.Printf("真实POC:    %d (%.1f%%)\n", report.Summary.TruePOCs, report.Summary.POCRate)
	fmt.Printf("Web漏洞:    %d\n", report.Summary.WebVulns)
	fmt.Printf("终端漏洞:   %d\n", report.Summary.TerminalVulns)
	fmt.Printf("其他:       %d\n", report.Summary.OtherVulns)
	fmt.Printf("Critical:   %d\n", report.Summary.CriticalRisk)
	fmt.Printf("High:       %d\n", report.Summary.HighRisk)

	// 漏洞类型分布
	fmt.Println("\n🎯 漏洞类型分布")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Web漏洞:    %d (%.1f%%)\n", report.VulnTypes.Web, report.VulnTypes.WebRate)
	fmt.Printf("终端漏洞:   %d (%.1f%%)\n", report.VulnTypes.Terminal,
		float64(report.VulnTypes.Terminal)/float64(report.TotalReports)*100)
	fmt.Printf("其他:       %d (%.1f%%)\n", report.VulnTypes.Other,
		float64(report.VulnTypes.Other)/float64(report.TotalReports)*100)

	// 风险等级
	fmt.Println("\n🚨 风险等级分布")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Critical:   %d\n", report.RiskLevels.Critical)
	fmt.Printf("High:       %d\n", report.RiskLevels.High)
	fmt.Printf("Medium:     %d\n", report.RiskLevels.Medium)
	fmt.Printf("Low:        %d\n", report.RiskLevels.Low)

	// Top CVEs
	if len(report.TopCVEs) > 0 {
		fmt.Println("\n🔝 Top CVE (出现次数最多)")
		fmt.Println(strings.Repeat("-", 70))
		for i, cve := range report.TopCVEs {
			fmt.Printf("%d. %s: %d次 (%.1f%%)\n", i+1, cve.CVEID, cve.Count, cve.Percentage)
		}
	}

	// 编程语言
	if len(report.Languages.TopLanguages) > 0 {
		fmt.Println("\n💻 编程语言分布 (Top 10)")
		fmt.Println(strings.Repeat("-", 70))
		for i, lang := range report.Languages.TopLanguages {
			fmt.Printf("%d. %s: %d个 (%.1f%%)\n", i+1, lang.Language, lang.Count, lang.Percentage)
		}
	}

	// 利用难度
	fmt.Println("\n⚡ 利用难度分布")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Easy:       %d\n", report.Difficulties.Easy)
	fmt.Printf("Medium:     %d\n", report.Difficulties.Medium)
	fmt.Printf("Hard:       %d\n", report.Difficulties.Hard)

	fmt.Println(strings.Repeat("=", 70))
}

// SaveReportJSON 保存为JSON
func (sc *StatisticsCollector) SaveReportJSON(report *StatisticsReport, filepath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// ExportToCSV 导出为CSV
func (sc *StatisticsCollector) ExportToCSV(reports []AnalysisReport, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 1. 漏洞类型CSV
	if err := sc.exportVulnTypesToCSV(reports, filepath.Join(outputDir, "vulnerability_types.csv")); err != nil {
		return err
	}

	// 2. 风险等级CSV
	if err := sc.exportRiskLevelsToCSV(reports, filepath.Join(outputDir, "risk_levels.csv")); err != nil {
		return err
	}

	// 3. 时间趋势CSV
	if err := sc.exportTimeTrendsToCSV(reports, filepath.Join(outputDir, "time_trends.csv")); err != nil {
		return err
	}

	return nil
}

// exportVulnTypesToCSV 导出漏洞类型到CSV
func (sc *StatisticsCollector) exportVulnTypesToCSV(reports []AnalysisReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"CVE ID", "Repository", "Vulnerability Type", "Category", "Risk Level"})

	// 写入数据
	for _, report := range reports {
		writer.Write([]string{
			report.CVE.CVEID,
			report.Repository.FullName,
			report.Analysis.VulnerabilityType,
			report.Analysis.VulnerabilityCategory,
			report.Analysis.RiskLevel,
		})
	}

	return nil
}

// exportRiskLevelsToCSV 导出风险等级到CSV
func (sc *StatisticsCollector) exportRiskLevelsToCSV(reports []AnalysisReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Risk Level", "Count"})

	// 统计
	counts := make(map[string]int)
	for _, report := range reports {
		level := report.Analysis.RiskLevel
		counts[level]++
	}

	// 写入
	for level, count := range counts {
		writer.Write([]string{level, fmt.Sprintf("%d", count)})
	}

	return nil
}

// exportTimeTrendsToCSV 导出时间趋势到CSV
func (sc *StatisticsCollector) exportTimeTrendsToCSV(reports []AnalysisReport, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Date", "Count"})

	// 按日期统计
	dailyCounts := make(map[string]int)
	for _, report := range reports {
		date := report.Metadata.GeneratedAt.Format("2006-01-02")
		dailyCounts[date]++
	}

	// 写入并排序
	var dates []string
	for date := range dailyCounts {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	for _, date := range dates {
		writer.Write([]string{date, fmt.Sprintf("%d", dailyCounts[date])})
	}

	return nil
}
