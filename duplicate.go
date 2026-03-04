package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite驱动
)

// ==================== 去重数据结构 ====================

// DuplicateRecord 去重记录
type DuplicateRecord struct {
	ID           int       `json:"id"`
	RepoURL      string    `json:"repo_url"`
	RepoHash     string    `json:"repo_hash"`
	ContentHash  string    `json:"content_hash"`
	CombinedHash string    `json:"combined_hash"`
	CVEID        string    `json:"cve_id"`
	VulnType     string    `json:"vulnerability_type"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	ReportPath   string    `json:"report_path"`
}

// DuplicateCheckResult 去重检查结果
type DuplicateCheckResult struct {
	IsDuplicate    bool              `json:"is_duplicate"`
	DuplicateType  string            `json:"duplicate_type"` // "repo", "content", "cve", "none"
	MatchedRecords []DuplicateRecord `json:"matched_records"`
	ShouldAnalyze  bool              `json:"should_analyze"`
	Reason         string            `json:"reason"`
}

// ==================== 去重检查器 ====================

// DuplicateChecker 去重检查器
type DuplicateChecker struct {
	db         *sql.DB
	cacheDir   string
	indexCache map[string]string // hash -> report path
	cacheMutex sync.RWMutex
}

// NewDuplicateChecker 创建去重检查器
func NewDuplicateChecker(dbPath string) (*DuplicateChecker, error) {
	// 确保数据库目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %v", err)
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	// 创建表
	if err := createDuplicateTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("创建表失败: %v", err)
	}

	dc := &DuplicateChecker{
		db:         db,
		cacheDir:   filepath.Join(filepath.Dir(dbPath), "cache"),
		indexCache: make(map[string]string),
	}

	// 加载缓存
	if err := dc.loadCache(); err != nil {
		fmt.Printf("⚠️  加载缓存失败: %v\n", err)
	}

	return dc, nil
}

// createDuplicateTables 创建去重表
func createDuplicateTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS duplicates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_url TEXT NOT NULL,
		repo_hash TEXT NOT NULL,
		content_hash TEXT,
		combined_hash TEXT UNIQUE,
		cve_id TEXT,
		vulnerability_type TEXT,
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		report_path TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_repo_url ON duplicates(repo_url);
	CREATE INDEX IF NOT EXISTS idx_repo_hash ON duplicates(repo_hash);
	CREATE INDEX IF NOT EXISTS idx_content_hash ON duplicates(content_hash);
	CREATE INDEX IF NOT EXISTS idx_cve_id ON duplicates(cve_id);
	CREATE INDEX IF NOT EXISTS idx_combined_hash ON duplicates(combined_hash);
	`

	_, err := db.Exec(query)
	return err
}

// loadCache 加载缓存
func (dc *DuplicateChecker) loadCache() error {
	// 确保缓存目录存在
	if err := os.MkdirAll(dc.cacheDir, 0755); err != nil {
		return err
	}

	// 加载repo hash缓存
	repoHashFile := filepath.Join(dc.cacheDir, "repo_hashes.json")
	if data, err := os.ReadFile(repoHashFile); err == nil {
		var cache map[string]string
		if err := json.Unmarshal(data, &cache); err == nil {
			dc.cacheMutex.Lock()
			for k, v := range cache {
				dc.indexCache[k] = v
			}
			dc.cacheMutex.Unlock()
		}
	}

	return nil
}

// saveCache 保存缓存
func (dc *DuplicateChecker) saveCache() error {
	dc.cacheMutex.RLock()
	defer dc.cacheMutex.RUnlock()

	repoHashFile := filepath.Join(dc.cacheDir, "repo_hashes.json")
	data, err := json.MarshalIndent(dc.indexCache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(repoHashFile, data, 0644)
}

// CheckDuplicate 检查是否重复
func (dc *DuplicateChecker) CheckDuplicate(repo GitHubRepository, hashes ReportHashes, cveID string) (*DuplicateCheckResult, error) {
	result := &DuplicateCheckResult{
		IsDuplicate:    false,
		DuplicateType:  "none",
		MatchedRecords: []DuplicateRecord{},
		ShouldAnalyze:  true,
		Reason:         "新仓库，需要分析",
	}

	// 1. 检查combined hash（最精确的去重）
	if record, err := dc.findByCombinedHash(hashes.CombinedHash); err == nil && record != nil {
		result.IsDuplicate = true
		result.DuplicateType = "content"
		result.MatchedRecords = append(result.MatchedRecords, *record)
		result.ShouldAnalyze = false
		result.Reason = "完全相同的内容（代码完全重复）"
		return result, nil
	}

	// 2. 检查repo hash（同一仓库）
	if record, err := dc.findByRepoHash(hashes.RepoHash); err == nil && record != nil {
		result.IsDuplicate = true
		result.DuplicateType = "repo"
		result.MatchedRecords = append(result.MatchedRecords, *record)

		// 检查仓库是否更新
		repoUpdated, _ := time.Parse(time.RFC3339, repo.UpdatedAt)
		lastSeen := record.LastSeen

		if repoUpdated.After(lastSeen) {
			result.ShouldAnalyze = true
			result.Reason = fmt.Sprintf("仓库已更新（最后分析: %s, 仓库更新: %s）",
				lastSeen.Format("2006-01-02"), repoUpdated.Format("2006-01-02"))
		} else {
			result.ShouldAnalyze = false
			result.Reason = "仓库未更新，无需重新分析"
		}

		return result, nil
	}

	// 3. 检查CVE编号（同一CVE的不同POC）
	if cveID != "" && cveID != "UNKNOWN" {
		if records, err := dc.findByCVE(cveID); err == nil && len(records) > 0 {
			result.IsDuplicate = true
			result.DuplicateType = "cve"
			result.MatchedRecords = records
			result.ShouldAnalyze = true // 同一CVE的不同POC也应该分析
			result.Reason = fmt.Sprintf("已存在该CVE的 %d 个POC", len(records))
			return result, nil
		}
	}

	return result, nil
}

// findByCombinedHash 根据combined hash查找
func (dc *DuplicateChecker) findByCombinedHash(hash string) (*DuplicateRecord, error) {
	query := `SELECT id, repo_url, repo_hash, content_hash, combined_hash, cve_id,
	                 vulnerability_type, first_seen, last_seen, report_path
	          FROM duplicates WHERE combined_hash = ? LIMIT 1`

	row := dc.db.QueryRow(query, hash)

	var record DuplicateRecord
	err := row.Scan(
		&record.ID,
		&record.RepoURL,
		&record.RepoHash,
		&record.ContentHash,
		&record.CombinedHash,
		&record.CVEID,
		&record.VulnType,
		&record.FirstSeen,
		&record.LastSeen,
		&record.ReportPath,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &record, err
}

// findByRepoHash 根据repo hash查找
func (dc *DuplicateChecker) findByRepoHash(hash string) (*DuplicateRecord, error) {
	query := `SELECT id, repo_url, repo_hash, content_hash, combined_hash, cve_id,
	                 vulnerability_type, first_seen, last_seen, report_path
	          FROM duplicates WHERE repo_hash = ? ORDER BY last_seen DESC LIMIT 1`

	row := dc.db.QueryRow(query, hash)

	var record DuplicateRecord
	err := row.Scan(
		&record.ID,
		&record.RepoURL,
		&record.RepoHash,
		&record.ContentHash,
		&record.CombinedHash,
		&record.CVEID,
		&record.VulnType,
		&record.FirstSeen,
		&record.LastSeen,
		&record.ReportPath,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &record, err
}

// findByCVE 根据CVE编号查找
func (dc *DuplicateChecker) findByCVE(cveID string) ([]DuplicateRecord, error) {
	query := `SELECT id, repo_url, repo_hash, content_hash, combined_hash, cve_id,
	                 vulnerability_type, first_seen, last_seen, report_path
	          FROM duplicates WHERE cve_id = ? ORDER BY last_seen DESC`

	rows, err := dc.db.Query(query, cveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DuplicateRecord
	for rows.Next() {
		var record DuplicateRecord
		err := rows.Scan(
			&record.ID,
			&record.RepoURL,
			&record.RepoHash,
			&record.ContentHash,
			&record.CombinedHash,
			&record.CVEID,
			&record.VulnType,
			&record.FirstSeen,
			&record.LastSeen,
			&record.ReportPath,
		)

		if err != nil {
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// AddRecord 添加去重记录
func (dc *DuplicateChecker) AddRecord(report AnalysisReport, reportPath string) error {
	// 检查是否已存在
	existing, err := dc.findByCombinedHash(report.Hashes.CombinedHash)
	if err != nil {
		return err
	}

	// 提取漏洞类型
	vulnType := ""
	if report.Analysis.VulnerabilityType != "" {
		vulnType = report.Analysis.VulnerabilityType
	}

	if existing != nil {
		// 更新last_seen
		query := `UPDATE duplicates SET last_seen = ?, report_path = ? WHERE combined_hash = ?`
		_, err = dc.db.Exec(query, time.Now(), reportPath, report.Hashes.CombinedHash)
		return err
	}

	// 插入新记录
	query := `INSERT INTO duplicates (repo_url, repo_hash, content_hash, combined_hash,
	                                 cve_id, vulnerability_type, report_path)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = dc.db.Exec(
		query,
		report.Repository.URL,
		report.Hashes.RepoHash,
		report.Hashes.ContentHash,
		report.Hashes.CombinedHash,
		report.CVE.CVEID,
		vulnType,
		reportPath,
	)

	// 更新缓存
	dc.cacheMutex.Lock()
	dc.indexCache[report.Hashes.RepoHash] = reportPath
	dc.cacheMutex.Unlock()

	// 保存缓存
	go dc.saveCache()

	return err
}

// GetStatistics 获取去重统计信息
func (dc *DuplicateChecker) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总记录数
	var total int
	err := dc.db.QueryRow("SELECT COUNT(*) FROM duplicates").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_records"] = total

	// 按CVE分组
	rows, err := dc.db.Query(`
		SELECT cve_id, COUNT(*) as count
		FROM duplicates
		WHERE cve_id IS NOT NULL AND cve_id != 'UNKNOWN'
		GROUP BY cve_id
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		var cveStats []map[string]interface{}
		for rows.Next() {
			var cveID string
			var count int
			if rows.Scan(&cveID, &count) == nil {
				cveStats = append(cveStats, map[string]interface{}{
					"cve_id": cveID,
					"count":  count,
				})
			}
		}
		rows.Close()
		stats["top_cves"] = cveStats
	}

	// 按漏洞类型分组
	rows, err = dc.db.Query(`
		SELECT vulnerability_type, COUNT(*) as count
		FROM duplicates
		WHERE vulnerability_type IS NOT NULL AND vulnerability_type != ''
		GROUP BY vulnerability_type
	`)
	if err == nil {
		var typeStats []map[string]interface{}
		for rows.Next() {
			var vulnType string
			var count int
			if rows.Scan(&vulnType, &count) == nil {
				typeStats = append(typeStats, map[string]interface{}{
					"type":  vulnType,
					"count": count,
				})
			}
		}
		rows.Close()
		stats["by_type"] = typeStats
	}

	return stats, nil
}

// Close 关闭数据库连接
func (dc *DuplicateChecker) Close() error {
	// 保存缓存
	dc.saveCache()

	return dc.db.Close()
}
