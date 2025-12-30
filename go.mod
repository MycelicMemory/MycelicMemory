module github.com/MycelicMemory/ultrathink

go 1.21

require (
	// Database
	github.com/mattn/go-sqlite3 v1.14.19

	// REST API
	github.com/gin-gonic/gin v1.9.1
	github.com/gin-contrib/cors v1.5.0

	// CLI
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2

	// Utilities
	github.com/google/uuid v1.5.0
	github.com/tidwall/gjson v1.17.0
)
