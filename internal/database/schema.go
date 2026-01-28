package database

// Schema contains the complete SQLite schema for MyclicMemory
// VERIFIED: Extracted from Local Memory v1.2.0 database via sqlite3 .schema
//
// Tables (16 total):
// - Core: memories, memory_relationships, categories, memory_categorizations, domains, vector_metadata, agent_sessions
// - FTS5: memories_fts (+ 4 internal tables)
// - Metadata: performance_metrics, migration_log, schema_version, sqlite_sequence

// SchemaVersion is the current schema version
const SchemaVersion = 1

// CoreSchema contains the main table definitions
// VERIFIED: Exact schema from ~/.local-memory/unified-memories.db
const CoreSchema = `
-- Enable foreign key support (required for CASCADE)
PRAGMA foreign_keys = ON;

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER PRIMARY KEY,
	applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- MEMORIES TABLE
-- VERIFIED: Primary content storage
-- =============================================================================
CREATE TABLE IF NOT EXISTS memories (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	source TEXT,
	importance INTEGER DEFAULT 5,
	tags TEXT,  -- JSON array: ["tag1", "tag2"]
	session_id TEXT,
	domain TEXT,
	embedding BLOB,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	agent_type TEXT DEFAULT 'unknown',
	agent_context TEXT,
	access_scope TEXT DEFAULT 'session',
	slug TEXT,
	-- Hierarchical chunking fields (Phase 1 benchmark improvement)
	parent_memory_id TEXT REFERENCES memories(id) ON DELETE CASCADE,
	chunk_level INTEGER DEFAULT 0,  -- 0=full, 1=paragraph, 2=atomic
	chunk_index INTEGER DEFAULT 0   -- position within parent
);

-- VERIFIED: 9 indexes on memories table (added chunk indexes for Phase 1)
CREATE INDEX IF NOT EXISTS idx_memories_session_id ON memories(session_id);
CREATE INDEX IF NOT EXISTS idx_memories_domain ON memories(domain);
CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance);
CREATE INDEX IF NOT EXISTS idx_memories_access_scope ON memories(access_scope);
CREATE INDEX IF NOT EXISTS idx_memories_slug ON memories(slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_slug_unique ON memories(slug) WHERE slug IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_parent ON memories(parent_memory_id);
CREATE INDEX IF NOT EXISTS idx_memories_chunk_level ON memories(chunk_level);

-- =============================================================================
-- MEMORY RELATIONSHIPS TABLE
-- VERIFIED: Graph edges with 7 relationship types
-- =============================================================================
CREATE TABLE IF NOT EXISTS memory_relationships (
	id TEXT PRIMARY KEY,
	source_memory_id TEXT NOT NULL,
	target_memory_id TEXT NOT NULL,
	relationship_type TEXT NOT NULL CHECK (
		relationship_type IN ('references', 'contradicts', 'expands', 'similar', 'sequential', 'causes', 'enables')
	),
	strength REAL NOT NULL CHECK (strength >= 0.0 AND strength <= 1.0),
	context TEXT,
	auto_generated BOOLEAN NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (source_memory_id) REFERENCES memories(id) ON DELETE CASCADE,
	FOREIGN KEY (target_memory_id) REFERENCES memories(id) ON DELETE CASCADE
);

-- VERIFIED: 4 indexes on relationships + compound indexes for optimized graph queries
CREATE INDEX IF NOT EXISTS idx_relationships_source ON memory_relationships(source_memory_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON memory_relationships(target_memory_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON memory_relationships(relationship_type);
CREATE INDEX IF NOT EXISTS idx_relationships_strength ON memory_relationships(strength);

-- Compound indexes for optimized graph traversal (N+1 query fix)
CREATE INDEX IF NOT EXISTS idx_relationships_source_target ON memory_relationships(source_memory_id, target_memory_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target_source ON memory_relationships(target_memory_id, source_memory_id);
CREATE INDEX IF NOT EXISTS idx_relationships_source_strength ON memory_relationships(source_memory_id, strength);
CREATE INDEX IF NOT EXISTS idx_relationships_target_strength ON memory_relationships(target_memory_id, strength);

-- =============================================================================
-- CATEGORIES TABLE
-- VERIFIED: Hierarchical organization with parent support
-- =============================================================================
CREATE TABLE IF NOT EXISTS categories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL,
	parent_category_id TEXT,
	confidence_threshold REAL NOT NULL DEFAULT 0.7 CHECK (confidence_threshold >= 0.0 AND confidence_threshold <= 1.0),
	auto_generated BOOLEAN NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (parent_category_id) REFERENCES categories(id) ON DELETE SET NULL
);

-- =============================================================================
-- MEMORY CATEGORIZATIONS TABLE
-- VERIFIED: M2M junction with confidence scoring
-- =============================================================================
CREATE TABLE IF NOT EXISTS memory_categorizations (
	memory_id TEXT NOT NULL,
	category_id TEXT NOT NULL,
	confidence REAL NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
	reasoning TEXT,
	created_at DATETIME NOT NULL,
	PRIMARY KEY (memory_id, category_id),
	FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE,
	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- =============================================================================
-- DOMAINS TABLE
-- VERIFIED: Knowledge partitions
-- =============================================================================
CREATE TABLE IF NOT EXISTS domains (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

-- =============================================================================
-- VECTOR METADATA TABLE
-- VERIFIED: 768-dimensional embedding tracking (nomic-embed-text)
-- =============================================================================
CREATE TABLE IF NOT EXISTS vector_metadata (
	memory_id TEXT PRIMARY KEY,
	vector_index INTEGER NOT NULL,
	embedding_model TEXT NOT NULL,
	embedding_dimension INTEGER NOT NULL,
	last_updated DATETIME NOT NULL,
	FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE
);

-- =============================================================================
-- AGENT SESSIONS TABLE
-- VERIFIED: Session management with 4 agent types
-- =============================================================================
CREATE TABLE IF NOT EXISTS agent_sessions (
	session_id TEXT PRIMARY KEY,
	agent_type TEXT NOT NULL CHECK (agent_type IN ('claude-desktop', 'claude-code', 'api', 'unknown')),
	agent_context TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_accessed DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_active BOOLEAN DEFAULT 1,
	metadata TEXT DEFAULT '{}'
);

-- =============================================================================
-- PERFORMANCE METRICS TABLE
-- VERIFIED: Operation timing tracking
-- =============================================================================
CREATE TABLE IF NOT EXISTS performance_metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	operation_type TEXT NOT NULL,
	execution_time_ms INTEGER NOT NULL,
	memory_count INTEGER,
	timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- MIGRATION LOG TABLE
-- VERIFIED: Database migration tracking
-- =============================================================================
CREATE TABLE IF NOT EXISTS migration_log (
	id TEXT PRIMARY KEY,
	migration_type TEXT NOT NULL,
	source_db_path TEXT,
	original_session_id TEXT,
	new_session_id TEXT,
	memories_migrated INTEGER DEFAULT 0,
	relationships_migrated INTEGER DEFAULT 0,
	categories_migrated INTEGER DEFAULT 0,
	migration_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	checksum TEXT,
	success BOOLEAN DEFAULT 0,
	error_message TEXT
);

-- =============================================================================
-- BENCHMARK RUNS TABLE
-- Captures each benchmark execution with git context and results
-- =============================================================================
CREATE TABLE IF NOT EXISTS benchmark_runs (
	id TEXT PRIMARY KEY,
	started_at DATETIME NOT NULL,
	completed_at DATETIME,
	status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),

	-- Git context for reproducibility
	git_commit_hash TEXT NOT NULL,
	git_branch TEXT,
	git_dirty BOOLEAN DEFAULT 0,

	-- Configuration snapshot (JSON)
	config_snapshot TEXT NOT NULL,
	benchmark_type TEXT NOT NULL DEFAULT 'locomo',

	-- Results
	overall_score REAL,
	overall_f1 REAL,
	overall_bleu1 REAL,
	total_questions INTEGER,
	total_correct INTEGER,

	-- Timing
	duration_seconds REAL,

	-- Error tracking
	error_message TEXT,

	-- Comparison
	baseline_run_id TEXT REFERENCES benchmark_runs(id),
	improvement_from_baseline REAL,
	is_best_run BOOLEAN DEFAULT 0,

	-- Autonomous loop context
	autonomous_loop_id TEXT,
	iteration_number INTEGER DEFAULT 0,
	change_description TEXT,

	-- Metadata
	created_by TEXT DEFAULT 'manual',
	notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_benchmark_runs_started ON benchmark_runs(started_at);
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_status ON benchmark_runs(status);
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_commit ON benchmark_runs(git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_loop ON benchmark_runs(autonomous_loop_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_runs_best ON benchmark_runs(is_best_run);

-- =============================================================================
-- BENCHMARK RESULTS BY CATEGORY TABLE
-- Per-category breakdown of scores
-- =============================================================================
CREATE TABLE IF NOT EXISTS benchmark_results_by_category (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL REFERENCES benchmark_runs(id) ON DELETE CASCADE,
	category TEXT NOT NULL,

	-- Scores
	llm_judge_accuracy REAL,
	f1_score REAL,
	bleu1_score REAL,

	-- Counts
	total_questions INTEGER,
	correct_count INTEGER,

	-- Comparison to previous best
	previous_best_accuracy REAL,
	improvement REAL,

	UNIQUE(run_id, category)
);

CREATE INDEX IF NOT EXISTS idx_category_results_run ON benchmark_results_by_category(run_id);
CREATE INDEX IF NOT EXISTS idx_category_results_category ON benchmark_results_by_category(category);

-- =============================================================================
-- BENCHMARK QUESTION RESULTS TABLE
-- Individual question-level results for drill-down analysis
-- =============================================================================
CREATE TABLE IF NOT EXISTS benchmark_question_results (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL REFERENCES benchmark_runs(id) ON DELETE CASCADE,

	-- Question identification
	question_id TEXT NOT NULL,
	category TEXT NOT NULL,
	question_text TEXT NOT NULL,

	-- Answers
	gold_answer TEXT NOT NULL,
	generated_answer TEXT,

	-- Scores
	llm_judge_label INTEGER,
	f1_score REAL,
	bleu1_score REAL,

	-- Context used
	context_length INTEGER,
	memories_used INTEGER,
	retrieval_time_ms INTEGER,
	generation_time_ms INTEGER,

	-- Comparison
	changed_from_previous BOOLEAN DEFAULT 0,
	previous_was_correct BOOLEAN,

	UNIQUE(run_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_question_results_run ON benchmark_question_results(run_id);
CREATE INDEX IF NOT EXISTS idx_question_results_question ON benchmark_question_results(question_id);
CREATE INDEX IF NOT EXISTS idx_question_results_category ON benchmark_question_results(category);

-- =============================================================================
-- AUTONOMOUS LOOPS TABLE
-- Tracks autonomous improvement sessions
-- =============================================================================
CREATE TABLE IF NOT EXISTS autonomous_loops (
	id TEXT PRIMARY KEY,
	started_at DATETIME NOT NULL,
	completed_at DATETIME,
	status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'stopped', 'failed')),

	-- Configuration
	max_iterations INTEGER DEFAULT 10,
	min_improvement_threshold REAL DEFAULT 0.01,
	convergence_threshold REAL DEFAULT 0.005,

	-- Results
	total_iterations INTEGER DEFAULT 0,
	baseline_score REAL,
	final_score REAL,
	best_score REAL,
	best_run_id TEXT REFERENCES benchmark_runs(id),

	-- Stop reason
	stop_reason TEXT,

	-- Change tracking (JSON arrays)
	changes_attempted TEXT,
	changes_accepted TEXT,
	changes_rejected TEXT
);

CREATE INDEX IF NOT EXISTS idx_loops_status ON autonomous_loops(status);
CREATE INDEX IF NOT EXISTS idx_loops_started ON autonomous_loops(started_at);
`

// FTS5Schema contains the full-text search configuration
// VERIFIED: FTS5 virtual table with automatic sync triggers
// NOTE: Using standalone FTS5 table (not external content) for reliable trigger behavior
const FTS5Schema = `
-- =============================================================================
-- FTS5 VIRTUAL TABLE
-- VERIFIED: Full-text search with content sync
-- Using standalone FTS5 (stores own content) for reliable sync
-- =============================================================================
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
	id UNINDEXED,
	slug UNINDEXED,
	content,
	source,
	tags,
	session_id UNINDEXED,
	domain UNINDEXED
);

-- =============================================================================
-- FTS5 SYNCHRONIZATION TRIGGERS
-- VERIFIED: Automatic index maintenance
-- =============================================================================

-- Insert trigger: Add new content to FTS index
CREATE TRIGGER IF NOT EXISTS memories_fts_insert AFTER INSERT ON memories BEGIN
	INSERT INTO memories_fts(id, slug, content, source, tags, session_id, domain)
	VALUES (new.id, new.slug, new.content, new.source, new.tags, new.session_id, new.domain);
END;

-- Delete trigger: Remove content from FTS index
CREATE TRIGGER IF NOT EXISTS memories_fts_delete AFTER DELETE ON memories BEGIN
	DELETE FROM memories_fts WHERE id = old.id;
END;

-- Update trigger: Sync changes to FTS index
CREATE TRIGGER IF NOT EXISTS memories_fts_update AFTER UPDATE ON memories BEGIN
	UPDATE memories_fts SET
		slug = new.slug,
		content = new.content,
		source = new.source,
		tags = new.tags,
		session_id = new.session_id,
		domain = new.domain
	WHERE id = old.id;
END;
`

// RelationshipTypes contains the 7 verified relationship types
var RelationshipTypes = []string{
	"references",  // Memory references another
	"contradicts", // Memory contradicts another
	"expands",     // Memory expands on another
	"similar",     // Memory is similar to another
	"sequential",  // Memory follows another in sequence
	"causes",      // Memory causes another
	"enables",     // Memory enables another
}

// AgentTypes contains the 4 verified agent types
var AgentTypes = []string{
	"claude-desktop", // Claude Desktop app
	"claude-code",    // Claude Code CLI
	"api",            // Direct API access
	"unknown",        // Unknown/default agent
}

// IsValidRelationshipType checks if a relationship type is valid
func IsValidRelationshipType(t string) bool {
	for _, rt := range RelationshipTypes {
		if rt == t {
			return true
		}
	}
	return false
}

// IsValidAgentType checks if an agent type is valid
func IsValidAgentType(t string) bool {
	for _, at := range AgentTypes {
		if at == t {
			return true
		}
	}
	return false
}
