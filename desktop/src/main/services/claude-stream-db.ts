/**
 * Claude Chat Stream Database Reader
 * Reads data from the claude-chat-stream SQLite database using better-sqlite3
 */

import Database from 'better-sqlite3';
import * as fs from 'fs';
import type {
  ClaudeProject,
  ClaudeSession,
  ClaudeMessage,
  ClaudeToolCall,
} from '../../shared/types';

// Local types for claude-stream DB tables that differ from the shared API types
interface ClaudeFileReference {
  id: string;
  message_id: string;
  session_id: string;
  tool_call_id?: string;
  file_path: string;
  operation?: string;
  old_content_preview?: string;
  new_content_preview?: string;
  lines_changed?: number;
  timestamp?: string;
}

interface ChangeStreamEntry {
  id: number;
  entity_type: string;
  entity_id: string;
  change_type: string;
  payload: string;
  created_at: string;
  processed_by?: string;
}

export class ClaudeStreamDB {
  private db: Database.Database | null = null;
  private dbPath: string;

  constructor(dbPath: string) {
    this.dbPath = dbPath;
  }

  private getConnection(): Database.Database {
    if (!this.db) {
      if (!fs.existsSync(this.dbPath)) {
        throw new Error(`Claude Chat Stream database not found at: ${this.dbPath}`);
      }
      this.db = new Database(this.dbPath, { readonly: true });
    }
    return this.db;
  }

  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
  }

  // Check if database exists and is accessible
  isAvailable(): boolean {
    try {
      return fs.existsSync(this.dbPath);
    } catch {
      return false;
    }
  }

  // Projects
  getProjects(): ClaudeProject[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        original_path,
        display_name,
        discovered_at,
        last_activity,
        session_count,
        message_count
      FROM projects
      ORDER BY last_activity DESC
    `);
    return stmt.all() as ClaudeProject[];
  }

  getProject(id: string): ClaudeProject | null {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        original_path,
        display_name,
        discovered_at,
        last_activity,
        session_count,
        message_count
      FROM projects
      WHERE id = ?
    `);
    return stmt.get(id) as ClaudeProject | null;
  }

  // Sessions
  getSessions(projectId?: string): ClaudeSession[] {
    const db = this.getConnection();

    let sql = `
      SELECT
        id,
        project_id,
        file_path,
        first_prompt,
        summary,
        git_branch,
        is_sidechain,
        is_subagent,
        parent_session_id,
        message_count,
        user_message_count,
        assistant_message_count,
        tool_call_count,
        started_at,
        ended_at
      FROM sessions
    `;

    if (projectId) {
      sql += ` WHERE project_id = ?`;
    }

    sql += ` ORDER BY started_at DESC`;

    const stmt = db.prepare(sql);
    const rows = projectId ? stmt.all(projectId) : stmt.all();

    return rows.map((row: Record<string, unknown>) => ({
      ...row,
      is_sidechain: Boolean(row.is_sidechain),
      is_subagent: Boolean(row.is_subagent),
    })) as any as ClaudeSession[];
  }

  getSession(id: string): ClaudeSession | null {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        project_id,
        file_path,
        first_prompt,
        summary,
        git_branch,
        is_sidechain,
        is_subagent,
        parent_session_id,
        message_count,
        user_message_count,
        assistant_message_count,
        tool_call_count,
        started_at,
        ended_at
      FROM sessions
      WHERE id = ?
    `);
    const row = stmt.get(id) as Record<string, unknown> | undefined;

    if (!row) return null;

    return {
      ...row,
      is_sidechain: Boolean(row.is_sidechain),
      is_subagent: Boolean(row.is_subagent),
    } as any as ClaudeSession;
  }

  // Messages
  getMessages(sessionId: string): ClaudeMessage[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        session_id,
        parent_uuid,
        role,
        content,
        message_type,
        timestamp,
        cwd,
        git_branch,
        thinking_tokens,
        has_tool_calls
      FROM messages
      WHERE session_id = ?
      ORDER BY timestamp ASC, line_number ASC
    `);
    const rows = stmt.all(sessionId);

    return rows.map((row: Record<string, unknown>) => ({
      ...row,
      has_tool_calls: Boolean(row.has_tool_calls),
    })) as any as ClaudeMessage[];
  }

  getMessage(id: string): ClaudeMessage | null {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        session_id,
        parent_uuid,
        role,
        content,
        message_type,
        timestamp,
        cwd,
        git_branch,
        thinking_tokens,
        has_tool_calls
      FROM messages
      WHERE id = ?
    `);
    const row = stmt.get(id) as Record<string, unknown> | undefined;

    if (!row) return null;

    return {
      ...row,
      has_tool_calls: Boolean(row.has_tool_calls),
    } as any as ClaudeMessage;
  }

  // Tool calls
  getToolCalls(sessionId: string): ClaudeToolCall[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        message_id,
        session_id,
        tool_name,
        tool_id,
        parameters,
        result,
        result_truncated,
        success,
        error_message,
        duration_ms,
        sequence_number,
        timestamp
      FROM tool_calls
      WHERE session_id = ?
      ORDER BY timestamp ASC, sequence_number ASC
    `);
    const rows = stmt.all(sessionId);

    return rows.map((row: Record<string, unknown>) => ({
      ...row,
      result_truncated: Boolean(row.result_truncated),
      success: row.success === null ? undefined : Boolean(row.success),
    })) as any as ClaudeToolCall[];
  }

  getToolCallsForMessage(messageId: string): ClaudeToolCall[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        message_id,
        session_id,
        tool_name,
        tool_id,
        parameters,
        result,
        result_truncated,
        success,
        error_message,
        duration_ms,
        sequence_number,
        timestamp
      FROM tool_calls
      WHERE message_id = ?
      ORDER BY sequence_number ASC
    `);
    const rows = stmt.all(messageId);

    return rows.map((row: Record<string, unknown>) => ({
      ...row,
      result_truncated: Boolean(row.result_truncated),
      success: row.success === null ? undefined : Boolean(row.success),
    })) as any as ClaudeToolCall[];
  }

  // File references
  getFileReferences(sessionId: string): ClaudeFileReference[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        message_id,
        session_id,
        tool_call_id,
        file_path,
        operation,
        old_content_preview,
        new_content_preview,
        lines_changed,
        timestamp
      FROM file_references
      WHERE session_id = ?
      ORDER BY timestamp ASC
    `);
    return stmt.all(sessionId) as ClaudeFileReference[];
  }

  // Change stream - for extraction service
  getUnprocessedChanges(processor: string, limit = 100): ChangeStreamEntry[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        entity_type,
        entity_id,
        change_type,
        payload,
        created_at,
        processed_by
      FROM change_stream
      WHERE processed_by IS NULL OR processed_by != ?
      ORDER BY created_at ASC
      LIMIT ?
    `);
    return stmt.all(processor, limit) as ChangeStreamEntry[];
  }

  getRecentChanges(limit = 50): ChangeStreamEntry[] {
    const db = this.getConnection();
    const stmt = db.prepare(`
      SELECT
        id,
        entity_type,
        entity_id,
        change_type,
        payload,
        created_at,
        processed_by
      FROM change_stream
      ORDER BY created_at DESC
      LIMIT ?
    `);
    return stmt.all(limit) as ChangeStreamEntry[];
  }

  // Statistics
  getStats(): {
    projectCount: number;
    sessionCount: number;
    messageCount: number;
    toolCallCount: number;
  } {
    const db = this.getConnection();

    const projectCount = (db.prepare('SELECT COUNT(*) as count FROM projects').get() as { count: number }).count;
    const sessionCount = (db.prepare('SELECT COUNT(*) as count FROM sessions').get() as { count: number }).count;
    const messageCount = (db.prepare('SELECT COUNT(*) as count FROM messages').get() as { count: number }).count;
    const toolCallCount = (db.prepare('SELECT COUNT(*) as count FROM tool_calls').get() as { count: number }).count;

    return {
      projectCount,
      sessionCount,
      messageCount,
      toolCallCount,
    };
  }

  // Search messages (using FTS if available)
  searchMessages(query: string, limit = 50): ClaudeMessage[] {
    const db = this.getConnection();

    // Try FTS first
    try {
      const stmt = db.prepare(`
        SELECT m.*
        FROM messages m
        JOIN messages_fts fts ON m.rowid = fts.rowid
        WHERE messages_fts MATCH ?
        ORDER BY rank
        LIMIT ?
      `);
      const rows = stmt.all(query, limit);
      return rows.map((row: Record<string, unknown>) => ({
        ...row,
        has_tool_calls: Boolean(row.has_tool_calls),
      })) as any as ClaudeMessage[];
    } catch {
      // Fallback to LIKE search
      const stmt = db.prepare(`
        SELECT *
        FROM messages
        WHERE content LIKE ?
        ORDER BY timestamp DESC
        LIMIT ?
      `);
      const rows = stmt.all(`%${query}%`, limit);
      return rows.map((row: Record<string, unknown>) => ({
        ...row,
        has_tool_calls: Boolean(row.has_tool_calls),
      })) as any as ClaudeMessage[];
    }
  }
}
