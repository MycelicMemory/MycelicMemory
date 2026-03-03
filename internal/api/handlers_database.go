package api

import (
	"github.com/gin-gonic/gin"
)

// listDatabases handles GET /api/v1/databases
func (s *Server) listDatabases(c *gin.Context) {
	dbs, err := s.dbManager.ListDatabases()
	if err != nil {
		InternalError(c, "Failed to list databases: "+err.Error())
		return
	}

	SuccessResponse(c, "Found databases", dbs)
}

// createDatabase handles POST /api/v1/databases
func (s *Server) createDatabase(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	info, err := s.dbManager.CreateDatabase(req.Name, req.Description)
	if err != nil {
		BadRequestError(c, err.Error())
		return
	}

	CreatedResponse(c, "Database created", info)
}

// getDatabaseInfo handles GET /api/v1/databases/:name
func (s *Server) getDatabaseInfo(c *gin.Context) {
	name := c.Param("name")

	info, err := s.dbManager.GetDatabase(name)
	if err != nil {
		NotFoundError(c, err.Error())
		return
	}

	SuccessResponse(c, "Database info", info)
}

// deleteDatabase handles DELETE /api/v1/databases/:name
func (s *Server) deleteDatabase(c *gin.Context) {
	name := c.Param("name")

	if err := s.dbManager.DeleteDatabase(name); err != nil {
		BadRequestError(c, err.Error())
		return
	}

	SuccessResponse(c, "Database deleted", nil)
}

// switchDatabase handles POST /api/v1/databases/:name/switch
func (s *Server) switchDatabase(c *gin.Context) {
	name := c.Param("name")

	// Checkpoint WAL on the current database before switching away
	if err := s.db.Checkpoint(); err != nil {
		s.log.Warn("failed to checkpoint before switch", "error", err)
	}

	if err := s.dbManager.SwitchDatabase(name); err != nil {
		BadRequestError(c, err.Error())
		return
	}

	// Hot-reload: swap the database connection and all service layers
	if err := s.ReloadDatabase(); err != nil {
		InternalError(c, "Config updated but failed to reload database: "+err.Error())
		return
	}

	// Return the new active database info for frontend verification
	info, err := s.dbManager.GetDatabase(name)
	if err != nil {
		// Switch succeeded but couldn't fetch info — still report success
		SuccessResponse(c, "Switched to database: "+name, map[string]interface{}{
			"name":      name,
			"is_active": true,
		})
		return
	}

	SuccessResponse(c, "Switched to database: "+name, info)
}

// archiveDatabase handles POST /api/v1/databases/:name/archive
func (s *Server) archiveDatabase(c *gin.Context) {
	name := c.Param("name")

	// Checkpoint WAL to flush all pending writes into the main .db file.
	// Without this, the archive copy may be missing recent data (WAL mode
	// keeps uncommitted changes in a separate -wal file).
	activePath := s.config.GetActiveDBPath()
	archivePath := ""
	if name == "" || name == "default" {
		archivePath = s.config.Database.Path
	} else {
		for _, p := range s.config.Databases {
			if p.Name == name {
				archivePath = p.Path
				break
			}
		}
	}

	// Only checkpoint if we're archiving the currently active database
	if archivePath == activePath {
		if err := s.db.Checkpoint(); err != nil {
			s.log.Warn("failed to checkpoint WAL before archive", "error", err)
			// Continue anyway — the copy will still work, just might miss WAL data
		}
	}

	path, err := s.dbManager.ArchiveDatabase(name)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessResponse(c, "Database archived", map[string]string{
		"backup_path": path,
	})
}

// importDatabase handles POST /api/v1/databases/import
func (s *Server) importDatabase(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	info, err := s.dbManager.ImportDatabase(req.Path, req.Name)
	if err != nil {
		BadRequestError(c, err.Error())
		return
	}

	CreatedResponse(c, "Database imported", info)
}

// exportDatabase handles POST /api/v1/databases/:name/export
func (s *Server) exportDatabase(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	if err := s.dbManager.ExportDatabase(name, req.Path); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessResponse(c, "Database exported to "+req.Path, nil)
}
