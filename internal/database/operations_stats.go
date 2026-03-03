package database

import (
	"encoding/json"
)

// DetailedStats represents comprehensive system statistics computed via SQL
type DetailedStats struct {
	MemoriesThisWeek       int                   `json:"memories_this_week"`
	MemoriesByDay          []DayCount            `json:"memories_by_day"`
	ImportanceDistribution []ImportanceBucket    `json:"importance_distribution"`
	SourceBreakdown        []SourceCount         `json:"source_breakdown"`
	RelationshipCount      int                   `json:"relationship_count"`
	MostCommonTags         []TagCountResult      `json:"most_common_tags"`
	TotalMemories          int                   `json:"total_memories"`
}

// DayCount represents memory count for a specific day
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// ImportanceBucket represents a grouped importance range
type ImportanceBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// SourceCount represents memory count by source
type SourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

// TagCountResult represents a tag and its frequency
type TagCountResult struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// GetDetailedStats computes comprehensive statistics using SQL aggregations
func (d *Database) GetDetailedStats() (*DetailedStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &DetailedStats{}

	// Total memories
	err := d.db.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&stats.TotalMemories)
	if err != nil {
		return nil, err
	}

	// Memories this week
	err = d.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE created_at >= datetime('now', '-7 days')`).Scan(&stats.MemoriesThisWeek)
	if err != nil {
		return nil, err
	}

	// Memories by day (last 30 days)
	rows, err := d.db.Query(`
		SELECT date(created_at) as day, COUNT(*) as count
		FROM memories
		WHERE created_at >= datetime('now', '-30 days')
		GROUP BY day
		ORDER BY day
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.MemoriesByDay = []DayCount{}
	for rows.Next() {
		var dc DayCount
		if err := rows.Scan(&dc.Day, &dc.Count); err != nil {
			continue
		}
		stats.MemoriesByDay = append(stats.MemoriesByDay, dc)
	}

	// Importance distribution
	importanceRows, err := d.db.Query(`
		SELECT
			CASE
				WHEN importance <= 3 THEN '1-3'
				WHEN importance <= 6 THEN '4-6'
				WHEN importance <= 8 THEN '7-8'
				ELSE '9-10'
			END as range_bucket,
			COUNT(*) as count
		FROM memories
		GROUP BY range_bucket
		ORDER BY range_bucket
	`)
	if err != nil {
		return nil, err
	}
	defer importanceRows.Close()

	stats.ImportanceDistribution = []ImportanceBucket{}
	for importanceRows.Next() {
		var ib ImportanceBucket
		if err := importanceRows.Scan(&ib.Range, &ib.Count); err != nil {
			continue
		}
		stats.ImportanceDistribution = append(stats.ImportanceDistribution, ib)
	}

	// Source breakdown
	sourceRows, err := d.db.Query(`
		SELECT COALESCE(source, 'unknown') as src, COUNT(*) as count
		FROM memories
		GROUP BY src
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer sourceRows.Close()

	stats.SourceBreakdown = []SourceCount{}
	for sourceRows.Next() {
		var sc SourceCount
		if err := sourceRows.Scan(&sc.Source, &sc.Count); err != nil {
			continue
		}
		stats.SourceBreakdown = append(stats.SourceBreakdown, sc)
	}

	// Relationship count
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM relationships`).Scan(&stats.RelationshipCount)

	// Most common tags (top 10) — tags stored as JSON array in text column
	tagRows, err := d.db.Query(`SELECT tags FROM memories WHERE tags IS NOT NULL AND tags != '[]' AND tags != ''`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	tagCounts := make(map[string]int)
	for tagRows.Next() {
		var tagsJSON string
		if err := tagRows.Scan(&tagsJSON); err != nil {
			continue
		}
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			continue
		}
		for _, tag := range tags {
			tagCounts[tag]++
		}
	}

	// Sort and take top 10
	stats.MostCommonTags = []TagCountResult{}
	for i := 0; i < 10; i++ {
		maxTag := ""
		maxCount := 0
		for tag, count := range tagCounts {
			if count > maxCount {
				maxTag = tag
				maxCount = count
			}
		}
		if maxTag == "" {
			break
		}
		stats.MostCommonTags = append(stats.MostCommonTags, TagCountResult{Tag: maxTag, Count: maxCount})
		delete(tagCounts, maxTag)
	}

	return stats, nil
}
