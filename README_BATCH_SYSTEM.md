# POC批量分析系统使用文档

## 🎯 功能概述

这是一个完整的POC（Proof of Concept）批量分析系统，具备以下核心功能：

### ✅ 已实现功能

1. **报告保存系统**
   - JSON格式保存分析结果
   - 按日期组织报告文件
   - 自动索引和查询

2. **智能去重系统**
   - 基于CVE编号去重
   - 基于仓库URL去重
   - 基于内容hash去重（SHA256）
   - SQLite数据库存储去重记录

3. **智能更新机制**
   - 自动检测仓库更新
   - 只有仓库更新时才重新分析
   - 保留历史版本

4. **批量分析框架**
   - 并发分析（可配置并发数）
   - 请求限流（避免API限流）
   - 失败重试机制
   - 进度跟踪

5. **统计分析功能**
   - 漏洞类型分布统计
   - 风险等级统计
   - 时间趋势分析
   - Top CVE统计
   - 编程语言分布
   - 利用难度统计

6. **命令行界面**
   - 完整的CLI工具
   - 支持多种命令
   - 详细的帮助信息

---

## 📦 安装

### 依赖项

```bash
# 安装SQLite驱动
go get github.com/mattn/go-sqlite3

# 编译程序
go build -o test.exe *.go
```

---

## 🚀 使用方法

### 1. 批量分析POC

```bash
# 基本用法
./test.exe batch --input repos.txt

# 自定义并发数
./test.exe batch --input repos.txt --concurrency 5

# 只更新已有报告
./test.exe batch --input repos.txt --update-only
```

**输入文件格式 (repos.txt)**:
```
CVE-2026 POC
CVE-2026 exploit
https://github.com/user/repo
```

### 2. 查看分析报告

```bash
# 查看指定CVE的报告
./test.exe view --cve CVE-2026-21858

# 查看指定仓库的报告
./test.exe view --repo https://github.com/Chocapikk/CVE-2026-21858

# 查看最新的报告
./test.exe view --latest
```

### 3. 生成统计报告

```bash
# 查看所有统计
./test.exe stats

# 查看本周统计
./test.exe stats --period week

# 导出JSON
./test.exe stats --output statistics.json

# 导出CSV
./test.exe stats --export csv/
```

### 4. 导出数据

```bash
# 导出为CSV
./test.exe export --format csv --output data/

# 导出为JSON
./test.exe export --format json --output data/
```

---

## 📂 文件结构

```
test/
├── main.go           # 主程序和演示代码
├── report.go         # 报告保存和加载
├── duplicate.go      # 去重系统
├── batch.go          # 批量分析框架
├── statistics.go     # 统计分析功能
├── cli.go            # 命令行界面
├── types.go          # 通用类型定义
│
├── reports/          # 分析报告存储
│   ├── 2026-03-04/
│   │   ├── CVE-2026-21858_Chocapikk.json
│   │   └── ...
│   └── index.json    # 报告索引
│
├── data/             # 数据存储
│   ├── duplicates.db # SQLite去重数据库
│   └── cache/
│       ├── repo_hashes.json
│       └── content_hashes.json
│
└── statistics/       # 统计输出
    ├── summary.json
    ├── vulnerability_types.csv
    ├── risk_levels.csv
    └── time_trends.csv
```

---

## 📊 报告格式

### JSON报告结构

```json
{
  "metadata": {
    "generated_at": "2026-03-04T12:00:00Z",
    "report_version": "1.0",
    "analyzer": "AI-POC-Analyzer",
    "version": "1.0.0"
  },
  "repository": {
    "full_name": "Chocapikk/CVE-2026-21858",
    "url": "https://github.com/Chocapikk/CVE-2026-21858",
    "language": "Python",
    "stars": 250,
    "forks": 48
  },
  "cve": {
    "cve_id": "CVE-2026-21858"
  },
  "analysis": {
    "is_poc": true,
    "confidence": "high",
    "vulnerability_type": "web_vulnerability",
    "vulnerability_category": "RCE",
    "target_platform": "Web应用",
    "exploit_difficulty": "medium",
    "risk_level": "critical",
    "reason": "这是一个完整的n8n RCE利用链，包含文件读取到RCE的完整步骤",
    "key_features": [
      "未认证文件读取漏洞",
      "反序列化漏洞",
      "远程代码执行"
    ],
    "recommendations": "立即升级到最新版本，限制网络访问"
  },
  "files": [
    {
      "path": "exploit.py",
      "name": "exploit.py",
      "size": 10406,
      "content_hash": "abc123..."
    }
  ],
  "hashes": {
    "repo_hash": "def456...",
    "content_hash": "ghi789...",
    "combined_hash": "jkl012..."
  }
}
```

---

## 🔧 配置

### 环境变量

```bash
# AI配置
export AI_API_KEY="your-api-key"
export AI_BASE_URL="https://open.bigmodel.cn/api/coding/paas/v4"
export AI_MODEL="glm-4.7"

# GitHub配置（可选，用于提高API限制）
export GITHUB_TOKEN="your-github-token"
```

### 批量分析配置

```go
BatchConfig{
    MaxConcurrency:  3,               // 最大并发数
    RequestDelay:    2 * time.Second, // 请求间隔
    RetryCount:      2,               // 失败重试次数
    Timeout:         60 * time.Second,// 单个分析超时
    SkipDuplicates:  true,            // 跳过重复
    UpdateOnly:      false,           // 只更新已有
}
```

---

## 📈 统计报告示例

```
======================================================================
📊 POC分析统计报告
======================================================================
生成时间: 2026-03-04 12:00:00
统计周期: all
报告总数: 100

📈 摘要统计
----------------------------------------------------------------------
总分析数:   100
真实POC:    75 (75.0%)
Web漏洞:    60
终端漏洞:   30
其他:       10
Critical:   15
High:       45

🎯 漏洞类型分布
----------------------------------------------------------------------
Web漏洞:    60 (60.0%)
终端漏洞:   30 (30.0%)
其他:       10 (10.0%)

🚨 风险等级分布
----------------------------------------------------------------------
Critical:   15
High:       45
Medium:     30
Low:        10

🔝 Top CVE (出现次数最多)
----------------------------------------------------------------------
1. CVE-2026-21858: 5次 (5.0%)
2. CVE-2026-12345: 3次 (3.0%)
3. CVE-2026-67890: 2次 (2.0%)

💻 编程语言分布 (Top 10)
----------------------------------------------------------------------
1. Python: 50个 (50.0%)
2. JavaScript: 20个 (20.0%)
3. Go: 15个 (15.0%)

⚡ 利用难度分布
----------------------------------------------------------------------
Easy:       20
Medium:     50
Hard:       30
======================================================================
```

---

## 🎯 使用场景

### 1. 每日CVE监控

```bash
# 每天搜索最新的CVE POC并分析
./test.exe batch --input daily_cves.txt
./test.exe stats --period today
```

### 2. 大规模POC收集

```bash
# 批量分析GitHub上的POC
./test.exe batch --input all_pocs.txt --concurrency 10
```

### 3. 重复POC检测

```bash
# 系统会自动检测重复内容
# 相同的POC只会分析一次
# 节省API调用和时间
```

### 4. 趋势分析

```bash
# 生成月度统计报告
./test.exe stats --period month --export csv/
```

---

## 💡 最佳实践

1. **使用GitHub Token**
   - 注册GitHub Token可以提高API限制
   - 从60次/小时提升到5000次/小时

2. **合理设置并发数**
   - 默认3个并发是安全的
   - 有Token可以提高到5-10个
   - 注意API限流

3. **定期备份**
   - 备份reports/目录
   - 备份data/duplicates.db

4. **利用去重功能**
   - 开启SkipDuplicates选项
   - 避免重复分析相同内容

5. **统计分析**
   - 定期生成统计报告
   - 了解漏洞分布趋势

---

## 🐛 故障排除

### 问题1: SQLite编译错误

```bash
# Windows: 安装GCC工具链
# 或使用纯Go的SQLite实现
go get github.com/glebarez/sqlite
```

### 问题2: API限流

```bash
# 减少并发数
./test.exe batch --input repos.txt --concurrency 1

# 增加请求延迟
# 修改batch.go中的RequestDelay
```

### 问题3: 内存不足

```bash
# 分批处理
# 不要一次分析太多仓库
# 每100个仓库一批
```

---

## 📝 开发计划

- [ ] Web界面
- [ ] 实时监控Dashboard
- [ ] 自动定时任务
- [ ] 邮件/Slack通知
- [ ] 更多数据库支持
- [ ] Docker容器化

---

## 📄 许可证

MIT License

---

## 👥 贡献

欢迎提交Issue和Pull Request！

---

**Happy Hacking! 🎉**
