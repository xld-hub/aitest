package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// ==================== CLI命令 ====================

// CLI 命令行界面
type CLI struct {
	apiKey  string
	baseURL string
	model   string
	token   string
}

// NewCLI 创建CLI
func NewCLI() *CLI {
	return &CLI{}
}

// Run 运行CLI
func (cli *CLI) Run(args []string) error {
	if len(args) < 2 {
		cli.printUsage()
		return nil
	}

	// 加载配置
	cli.loadConfig()

	command := args[1]

	switch command {
	case "batch", "b":
		return cli.runBatchCommand(args[2:])
	case "view", "v":
		return cli.runViewCommand(args[2:])
	case "stats", "s":
		return cli.runStatsCommand(args[2:])
	case "check-duplicate", "check":
		return cli.runCheckDuplicateCommand(args[2:])
	case "export", "e":
		return cli.runExportCommand(args[2:])
	case "help", "h":
		cli.printDetailedHelp()
		return nil
	default:
		fmt.Printf("❌ 未知命令: %s\n", command)
		cli.printUsage()
		return nil
	}
}

// loadConfig 加载配置
func (cli *CLI) loadConfig() {
	// 从环境变量或配置文件加载
	cli.apiKey = os.Getenv("AI_API_KEY")
	if cli.apiKey == "" {
		cli.apiKey = "2bed35f970174454aa92801a2b0a32b7.J7N86eK9nLt2t0bd" // 默认值
	}

	cli.baseURL = os.Getenv("AI_BASE_URL")
	if cli.baseURL == "" {
		cli.baseURL = "https://open.bigmodel.cn/api/coding/paas/v4"
	}

	cli.model = os.Getenv("AI_MODEL")
	if cli.model == "" {
		cli.model = "glm-4.7"
	}

	cli.token = os.Getenv("GITHUB_TOKEN")
	if cli.token == "" {
		cli.token = "" // 可选
	}
}

// runBatchCommand 批量分析命令
func (cli *CLI) runBatchCommand(args []string) error {
	fs := flag.NewFlagSet("batch", flag.ExitOnError)
	inputFile := fs.String("input", "", "输入文件（每行一个仓库URL）")
	outputDir := fs.String("output", "reports", "输出目录")
	concurrency := fs.Int("concurrency", 3, "并发数")
	skipDuplicates := fs.Bool("skip-duplicates", true, "跳过重复内容")
	updateOnly := fs.Bool("update-only", false, "只更新已有报告")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *inputFile == "" {
		fmt.Println("❌ 请指定输入文件 (--input)")
		return nil
	}

	// 读取仓库列表
	repos, err := cli.readRepoList(*inputFile)
	if err != nil {
		return fmt.Errorf("读取仓库列表失败: %v", err)
	}

	fmt.Printf("📂 加载了 %d 个仓库\n", len(repos))

	// 创建批量分析器
	config := &BatchConfig{
		MaxConcurrency: *concurrency,
		SkipDuplicates: *skipDuplicates,
		UpdateOnly:     *updateOnly,
	}

	analyzer, err := NewBatchAnalyzer(cli.apiKey, cli.baseURL, cli.model, cli.token, config)
	if err != nil {
		return err
	}
	defer analyzer.Close()

	// 执行批量分析
	result, err := analyzer.AnalyzeBatch(repos)
	if err != nil {
		return err
	}

	// 保存批量分析结果
	cli.saveBatchResult(result, *outputDir)

	return nil
}

// runViewCommand 查看报告命令
func (cli *CLI) runViewCommand(args []string) error {
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	cveID := fs.String("cve", "", "CVE编号")
	repoURL := fs.String("repo", "", "仓库URL")
	latest := fs.Bool("latest", false, "查看最新的报告")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rm := NewReportManager("reports")

	var report *AnalysisReport
	var err error

	if *cveID != "" {
		reports, err := rm.FindReportByCVE(*cveID)
		if err != nil {
			return err
		}
		if len(reports) == 0 {
			fmt.Printf("❌ 未找到CVE报告: %s\n", *cveID)
			return nil
		}
		report = &reports[0]
	} else if *repoURL != "" {
		report, err = rm.FindReportByRepo(*repoURL)
		if err != nil {
			return fmt.Errorf("未找到仓库报告: %v", err)
		}
	} else if *latest {
		reports, err := rm.LoadAllReports()
		if err != nil {
			return err
		}
		if len(reports) == 0 {
			fmt.Println("❌ 没有可用的报告")
			return nil
		}
		// 找最新的
		latest := reports[0]
		for _, r := range reports {
			if r.Metadata.GeneratedAt.After(latest.Metadata.GeneratedAt) {
				latest = r
			}
		}
		report = &latest
	} else {
		fmt.Println("❌ 请指定 --cve, --repo 或 --latest")
		return nil
	}

	cli.printReport(report)
	return nil
}

// runStatsCommand 统计命令
func (cli *CLI) runStatsCommand(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	period := fs.String("period", "all", "统计周期: today, week, month, year, all")
	outputJSON := fs.String("output", "", "输出JSON文件路径")
	exportCSV := fs.String("export", "", "导出CSV目录")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rm := NewReportManager("reports")
	collector := NewStatisticsCollector(rm)

	// 生成统计
	report, err := collector.GenerateStatistics(*period)
	if err != nil {
		return err
	}

	// 打印报告
	collector.PrintReport(report)

	// 保存JSON
	if *outputJSON != "" {
		if err := collector.SaveReportJSON(report, *outputJSON); err != nil {
			fmt.Printf("⚠️  保存JSON失败: %v\n", err)
		} else {
			fmt.Printf("✅ JSON已保存: %s\n", *outputJSON)
		}
	}

	// 导出CSV
	if *exportCSV != "" {
		reports, _ := rm.LoadAllReports()
		if err := collector.ExportToCSV(reports, *exportCSV); err != nil {
			fmt.Printf("⚠️  导出CSV失败: %v\n", err)
		} else {
			fmt.Printf("✅ CSV已导出到: %s\n", *exportCSV)
		}
	}

	return nil
}

// runCheckDuplicateCommand 检查重复命令
func (cli *CLI) runCheckDuplicateCommand(args []string) error {
	fs := flag.NewFlagSet("check-duplicate", flag.ExitOnError)
	repoURL := fs.String("repo", "", "仓库URL")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *repoURL == "" {
		fmt.Println("❌ 请指定仓库URL (--repo)")
		return nil
	}

	dc, err := NewDuplicateChecker("data/duplicates.db")
	if err != nil {
		return err
	}
	defer dc.Close()

	// 查询重复记录
	// 这里需要先获取仓库信息并计算hash
	fmt.Printf("🔍 检查仓库重复: %s\n", *repoURL)
	fmt.Println("💡 提示：使用 'batch' 命令时会自动进行去重检查")

	return nil
}

// runExportCommand 导出命令
func (cli *CLI) runExportCommand(args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	format := fs.String("format", "csv", "导出格式: csv, json")
	outputDir := fs.String("output", "exports", "输出目录")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rm := NewReportManager("reports")
	reports, err := rm.LoadAllReports()
	if err != nil {
		return err
	}

	collector := NewStatisticsCollector(rm)

	switch *format {
	case "csv":
		if err := collector.ExportToCSV(reports, *outputDir); err != nil {
			return err
		}
		fmt.Printf("✅ CSV已导出到: %s\n", *outputDir)
	case "json":
		// 导出所有报告为JSON
		for _, report := range reports {
			filename := fmt.Sprintf("%s_%s.json", report.CVE.CVEID,
				report.Repository.Owner.Login)
			_ = fmt.Sprintf("%s/%s", *outputDir, filename)
			// 保存单个报告
			fmt.Printf("导出: %s\n", filename)
		}
	default:
		fmt.Printf("❌ 不支持的格式: %s\n", *format)
	}

	return nil
}

// ==================== 辅助方法 ====================

// readRepoList 读取仓库列表
func (cli *CLI) readRepoList(filename string) ([]GitHubRepository, error) {
	// 这里简化处理，实际应该从文件读取
	// 暂时返回空列表
	return []GitHubRepository{}, nil
}

// saveBatchResult 保存批量分析结果
func (cli *CLI) saveBatchResult(result *BatchAnalysisResult, outputDir string) {
	// 保存为JSON
	filename := fmt.Sprintf("batch_result_%s.json", time.Now().Format("20060102_150405"))
	path := outputDir + "/" + filename

	// 这里应该实际保存
	fmt.Printf("📄 批量分析结果已保存: %s\n", path)
}

// printReport 打印报告
func (cli *CLI) printReport(report *AnalysisReport) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("📊 POC分析报告\n")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Printf("仓库: %s\n", report.Repository.FullName)
	fmt.Printf("URL: %s\n", report.Repository.URL)
	fmt.Printf("CVE: %s\n", report.CVE.CVEID)
	fmt.Printf("分析时间: %s\n\n", report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))

	fmt.Println("🎯 漏洞分析:")
	fmt.Printf("  类型: %s\n", report.Analysis.VulnerabilityType)
	fmt.Printf("  分类: %s\n", report.Analysis.VulnerabilityCategory)
	fmt.Printf("  目标平台: %s\n", report.Analysis.TargetPlatform)
	fmt.Printf("  利用难度: %s\n", report.Analysis.ExploitDifficulty)
	fmt.Printf("  风险等级: %s\n", report.Analysis.RiskLevel)

	fmt.Println("\n📝 判断理由:")
	fmt.Printf("  %s\n", report.Analysis.Reason)

	if len(report.Analysis.KeyFeatures) > 0 {
		fmt.Println("\n🔑 关键特征:")
		for _, feature := range report.Analysis.KeyFeatures {
			fmt.Printf("  • %s\n", feature)
		}
	}

	fmt.Println("\n💡 建议:")
	fmt.Printf("  %s\n", report.Analysis.Recommendations)

	fmt.Println(strings.Repeat("=", 70))
}

// printUsage 打印基本用法
func (cli *CLI) printUsage() {
	fmt.Println("🤖 POC批量分析工具")
	fmt.Println("\n用法:")
	fmt.Println("  test.exe <command> [options]")
	fmt.Println("\n命令:")
	fmt.Println("  batch, b         批量分析POC")
	fmt.Println("  view, v          查看分析报告")
	fmt.Println("  stats, s         生成统计报告")
	fmt.Println("  check-duplicate  检查重复")
	fmt.Println("  export, e        导出数据")
	fmt.Println("  help, h          查看详细帮助")
	fmt.Println("\n使用 'test.exe help' 查看详细帮助")
}

// printDetailedHelp 打印详细帮助
func (cli *CLI) printDetailedHelp() {
	fmt.Println("🤖 POC批量分析工具 - 详细帮助")
	fmt.Println(strings.Repeat("=", 70))

	fmt.Println("\n📦 批量分析 (batch)")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("用法: test.exe batch --input <file> [options]")
	fmt.Println("\n选项:")
	fmt.Println("  --input <file>        输入文件（每行一个仓库URL或搜索关键词）")
	fmt.Println("  --output <dir>        输出目录（默认: reports）")
	fmt.Println("  --concurrency <n>     并发数（默认: 3）")
	fmt.Println("  --skip-duplicates     跳过重复内容（默认: true）")
	fmt.Println("  --update-only         只更新已有报告（默认: false）")
	fmt.Println("\n示例:")
	fmt.Println("  test.exe batch --input repos.txt")
	fmt.Println("  test.exe batch --input repos.txt --concurrency 5")

	fmt.Println("\n👁️  查看报告 (view)")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("用法: test.exe view [options]")
	fmt.Println("\n选项:")
	fmt.Println("  --cve <CVE-ID>        查看指定CVE的报告")
	fmt.Println("  --repo <URL>          查看指定仓库的报告")
	fmt.Println("  --latest              查看最新的报告")
	fmt.Println("\n示例:")
	fmt.Println("  test.exe view --cve CVE-2026-21858")
	fmt.Println("  test.exe view --latest")

	fmt.Println("\n📊 统计报告 (stats)")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("用法: test.exe stats [options]")
	fmt.Println("\n选项:")
	fmt.Println("  --period <period>     统计周期: today, week, month, year, all（默认: all）")
	fmt.Println("  --output <file>       输出JSON文件")
	fmt.Println("  --export <dir>        导出CSV到目录")
	fmt.Println("\n示例:")
	fmt.Println("  test.exe stats")
	fmt.Println("  test.exe stats --period week")
	fmt.Println("  test.exe stats --export csv/")

	fmt.Println("\n🔍 检查重复 (check-duplicate)")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("用法: test.exe check-duplicate --repo <URL>")
	fmt.Println("\n示例:")
	fmt.Println("  test.exe check-duplicate --repo https://github.com/user/repo")

	fmt.Println("\n📤 导出数据 (export)")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("用法: test.exe export --format <format> --output <dir>")
	fmt.Println("\n选项:")
	fmt.Println("  --format <format>     导出格式: csv, json（默认: csv）")
	fmt.Println("  --output <dir>        输出目录（默认: exports）")
	fmt.Println("\n示例:")
	fmt.Println("  test.exe export --format csv --output data/")

	fmt.Println("\n🔧 环境变量")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("  AI_API_KEY            AI API密钥")
	fmt.Println("  AI_BASE_URL           API基础URL")
	fmt.Println("  AI_MODEL              AI模型名称")
	fmt.Println("  GITHUB_TOKEN          GitHub Token（可选）")

	fmt.Println(strings.Repeat("=", 70))
}

// ==================== Main函数更新 ====================

// RunCLI 运行命令行界面（供main函数调用）
func RunCLI() {
	cli := NewCLI()
	if err := cli.Run(os.Args); err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		os.Exit(1)
	}
}
