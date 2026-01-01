package database

import (
	"database/sql"
	"fmt"
)

// MigrationV1ToV2 migrates the database from schema version 1 to version 2
// This adds temporal decay columns, entities tables, and updates FTS5
func MigrationV1ToV2(db *sql.DB) error {
	log.Info("running migration v1 to v2")

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Add new columns to memories table
	alterStatements := []string{
		"ALTER TABLE memories ADD COLUMN last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP;",
		"ALTER TABLE memories ADD COLUMN access_count INTEGER DEFAULT 1;",
		"ALTER TABLE memories ADD COLUMN strength REAL DEFAULT 1.0;",
		"ALTER TABLE memories ADD COLUMN decay_score REAL DEFAULT 1.0;",
		"ALTER TABLE memories ADD COLUMN tier_id INTEGER DEFAULT 1;",
	}

	for _, stmt := range alterStatements {
		if _, err := tx.Exec(stmt); err != nil {
			// Column may already exist, log and continue
			log.Debug("alter statement skipped (may already exist)", "stmt", stmt, "error", err)
		}
	}

	// 2. Create new indexes for decay columns
	indexStatements := []string{
		"CREATE INDEX IF NOT EXISTS idx_memories_decay_score ON memories(decay_score);",
		"CREATE INDEX IF NOT EXISTS idx_memories_tier_id ON memories(tier_id);",
		"CREATE INDEX IF NOT EXISTS idx_memories_last_accessed ON memories(last_accessed);",
	}

	for _, stmt := range indexStatements {
		if _, err := tx.Exec(stmt); err != nil {
			log.Warn("failed to create index", "stmt", stmt, "error", err)
		}
	}

	// 3. Create entities table
	entitiesSQL := `
		CREATE TABLE IF NOT EXISTS entities (
			id TEXT PRIMARY KEY,
			canonical_name TEXT NOT NULL UNIQUE,
			entity_type TEXT NOT NULL CHECK (
				entity_type IN ('person', 'place', 'organization', 'concept', 'event', 'thing', 'other')
			),
			embedding BLOB,
			mention_count INTEGER DEFAULT 1,
			first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
		CREATE INDEX IF NOT EXISTS idx_entities_mention_count ON entities(mention_count);
		CREATE INDEX IF NOT EXISTS idx_entities_canonical ON entities(canonical_name);
	`
	if _, err := tx.Exec(entitiesSQL); err != nil {
		log.Warn("failed to create entities table", "error", err)
	}

	// 4. Create memory_entities junction table
	memoryEntitiesSQL := `
		CREATE TABLE IF NOT EXISTS memory_entities (
			memory_id TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			mention_text TEXT,
			confidence REAL DEFAULT 1.0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (memory_id, entity_id),
			FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE,
			FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_memory_entities_memory ON memory_entities(memory_id);
		CREATE INDEX IF NOT EXISTS idx_memory_entities_entity ON memory_entities(entity_id);
	`
	if _, err := tx.Exec(memoryEntitiesSQL); err != nil {
		log.Warn("failed to create memory_entities table", "error", err)
	}

	// 5. Create memory_tiers table
	tiersSQL := `
		CREATE TABLE IF NOT EXISTS memory_tiers (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			min_decay_score REAL DEFAULT 0.0,
			max_decay_score REAL DEFAULT 1.0,
			retention_days INTEGER
		);
		INSERT OR IGNORE INTO memory_tiers (id, name, description, min_decay_score, max_decay_score, retention_days) VALUES
			(1, 'hot', 'Frequently accessed, high relevance', 0.7, 1.0, NULL),
			(2, 'warm', 'Moderate access, good relevance', 0.3, 0.7, NULL),
			(3, 'cold', 'Infrequent access, lower relevance', 0.05, 0.3, 90),
			(4, 'archived', 'Very low relevance, candidate for deletion', 0.0, 0.05, 30);
	`
	if _, err := tx.Exec(tiersSQL); err != nil {
		log.Warn("failed to create memory_tiers table", "error", err)
	}

	// 6. Initialize last_accessed from created_at for existing memories
	if _, err := tx.Exec(`
		UPDATE memories
		SET last_accessed = created_at
		WHERE last_accessed IS NULL
	`); err != nil {
		log.Warn("failed to initialize last_accessed", "error", err)
	}

	// 7. Update schema version
	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO schema_version (version, applied_at)
		VALUES (2, CURRENT_TIMESTAMP)
	`); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	log.Info("migration v1 to v2 completed successfully")
	return nil
}

// RunMigrations checks the current schema version and runs any pending migrations
func (d *Database) RunMigrations() error {
	version, err := d.GetSchemaVersion()
	if err != nil {
		// Schema version table may not exist yet
		version = 0
	}

	log.Info("checking migrations", "current_version", version, "target_version", SchemaVersion)

	if version >= SchemaVersion {
		log.Debug("database is up to date")
		return nil
	}

	// Run migrations sequentially
	if version < 2 {
		if err := MigrationV1ToV2(d.db); err != nil {
			return fmt.Errorf("migration v1 to v2 failed: %w", err)
		}
	}

	// Add future migrations here:
	// if version < 3 { MigrationV2ToV3(d.db) }

	return nil
}
