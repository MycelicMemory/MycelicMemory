/**
 * Extraction Service
 * Extracts memories from Claude Chat Stream and stores them in MycelicMemory
 */

import { ClaudeStreamDB } from './claude-stream-db';
import { MycelicMemoryClient } from './mycelicmemory-client';
import type { ExtractionJob, ExtractionConfig, ClaudeMessage } from '../../shared/types';

interface ExtractionServiceOptions {
  claudeDbPath: string;
  apiUrl: string;
  config: ExtractionConfig;
  onProgress?: (job: ExtractionJob) => void;
}

export class ExtractionService {
  private claudeDb: ClaudeStreamDB;
  private apiClient: MycelicMemoryClient;
  private config: ExtractionConfig;
  private onProgress?: (job: ExtractionJob) => void;
  private pollInterval: NodeJS.Timeout | null = null;
  private jobs: Map<string, ExtractionJob> = new Map();
  private processedChanges: Set<number> = new Set();

  constructor(options: ExtractionServiceOptions) {
    this.claudeDb = new ClaudeStreamDB(options.claudeDbPath);
    this.apiClient = new MycelicMemoryClient(options.apiUrl);
    this.config = options.config;
    this.onProgress = options.onProgress;
  }

  start(): void {
    if (this.pollInterval) return;

    this.pollInterval = setInterval(
      () => this.pollForChanges(),
      this.config.poll_interval_ms
    );

    // Initial poll
    this.pollForChanges();
  }

  stop(): void {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
    this.claudeDb.close();
  }

  getConfig(): ExtractionConfig {
    return { ...this.config };
  }

  updateConfig(config: ExtractionConfig): void {
    const wasRunning = this.pollInterval !== null;
    const intervalChanged = config.poll_interval_ms !== this.config.poll_interval_ms;

    this.config = { ...config };

    // Restart polling if interval changed and we were running
    if (wasRunning && intervalChanged) {
      this.stop();
      if (config.auto_extract) {
        this.start();
      }
    } else if (config.auto_extract && !wasRunning) {
      this.start();
    } else if (!config.auto_extract && wasRunning) {
      this.stop();
    }
  }

  getStatus(): ExtractionJob[] {
    return Array.from(this.jobs.values());
  }

  async extractSession(sessionId: string): Promise<ExtractionJob> {
    const jobId = uuidv4();
    const job: ExtractionJob = {
      id: jobId,
      session_id: sessionId,
      status: 'pending',
      messages_processed: 0,
      memories_created: 0,
      started_at: new Date().toISOString(),
    };

    this.jobs.set(jobId, job);
    this.notifyProgress(job);

    try {
      job.status = 'processing';
      this.notifyProgress(job);

      // Get session and messages
      const session = this.claudeDb.getSession(sessionId);
      if (!session) {
        throw new Error(`Session not found: ${sessionId}`);
      }

      const messages = this.claudeDb.getMessages(sessionId);
      const toolCalls = this.config.extract_tool_calls
        ? this.claudeDb.getToolCalls(sessionId)
        : [];

      // Extract memories from messages
      for (const message of messages) {
        if (message.content.length < this.config.min_message_length) {
          continue;
        }

        // Extract from user messages (prompts, questions, requirements)
        if (message.role === 'user') {
          await this.extractFromUserMessage(message, session.summary || '');
          job.memories_created++;
        }

        // Extract from assistant messages (answers, explanations, code)
        if (message.role === 'assistant') {
          const memoriesCreated = await this.extractFromAssistantMessage(message, session.summary || '');
          job.memories_created += memoriesCreated;
        }

        job.messages_processed++;
        this.notifyProgress(job);
      }

      // Extract from tool calls if enabled
      if (this.config.extract_tool_calls) {
        for (const toolCall of toolCalls) {
          if (this.config.extract_file_operations && this.isFileOperation(toolCall.tool_name)) {
            await this.extractFromFileOperation(toolCall);
            job.memories_created++;
          }
        }
      }

      job.status = 'completed';
      job.completed_at = new Date().toISOString();
    } catch (error) {
      job.status = 'failed';
      job.error = error instanceof Error ? error.message : String(error);
      job.completed_at = new Date().toISOString();
    }

    this.notifyProgress(job);
    return job;
  }

  private async pollForChanges(): Promise<void> {
    if (!this.claudeDb.isAvailable()) {
      return;
    }

    try {
      const changes = this.claudeDb.getUnprocessedChanges('mycelicmemory-desktop', 50);

      for (const change of changes) {
        if (this.processedChanges.has(change.id)) {
          continue;
        }

        if (change.entity_type === 'message' && change.change_type === 'insert') {
          await this.handleNewMessage(change);
        }

        this.processedChanges.add(change.id);
      }
    } catch (error) {
      console.error('Error polling for changes:', error);
    }
  }

  private async handleNewMessage(change: { entity_id: string; payload: string }): Promise<void> {
    try {
      const payload = JSON.parse(change.payload);

      // Only process substantial messages
      if (!payload.content || payload.content.length < this.config.min_message_length) {
        return;
      }

      // Get full message details
      const message = this.claudeDb.getMessage(change.entity_id);
      if (!message) return;

      // Get session context
      const session = this.claudeDb.getSession(message.session_id);
      const context = session?.summary || '';

      if (message.role === 'user') {
        await this.extractFromUserMessage(message, context);
      } else if (message.role === 'assistant') {
        await this.extractFromAssistantMessage(message, context);
      }
    } catch (error) {
      console.error('Error handling new message:', error);
    }
  }

  private async extractFromUserMessage(message: ClaudeMessage, context: string): Promise<void> {
    // User messages often contain:
    // - Requirements and specifications
    // - Questions and problem descriptions
    // - Code examples and snippets

    const content = message.content;

    // Skip very short messages or tool results
    if (content.length < this.config.min_message_length) {
      return;
    }

    // Determine importance based on content characteristics
    const importance = this.calculateImportance(content, 'user');

    // Determine domain from context
    const domain = this.inferDomain(content, context);

    // Store the memory
    await this.apiClient.storeMemory(content, {
      domain,
      source: 'claude-chat-stream',
      importance,
      tags: this.extractTags(content),
    });
  }

  private async extractFromAssistantMessage(message: ClaudeMessage, context: string): Promise<number> {
    const content = message.content;
    let memoriesCreated = 0;

    // Skip very short messages
    if (content.length < this.config.min_message_length) {
      return 0;
    }

    // Check for code blocks - extract as separate memories
    const codeBlocks = this.extractCodeBlocks(content);
    for (const codeBlock of codeBlocks) {
      if (codeBlock.code.length > 50) {
        const domain = this.inferDomain(codeBlock.code, context);
        await this.apiClient.storeMemory(
          `${codeBlock.language ? `[${codeBlock.language}]\n` : ''}${codeBlock.code}`,
          {
            domain: domain || 'code',
            source: 'claude-chat-stream',
            importance: 6,
            tags: ['code', codeBlock.language].filter(Boolean) as string[],
          }
        );
        memoriesCreated++;
      }
    }

    // Extract explanations and key points
    const explanation = this.extractExplanation(content);
    if (explanation && explanation.length >= this.config.min_message_length) {
      const importance = this.calculateImportance(explanation, 'assistant');
      const domain = this.inferDomain(explanation, context);

      await this.apiClient.storeMemory(explanation, {
        domain,
        source: 'claude-chat-stream',
        importance,
        tags: this.extractTags(explanation),
      });
      memoriesCreated++;
    }

    return memoriesCreated;
  }

  private async extractFromFileOperation(toolCall: {
    tool_name: string;
    parameters?: string;
    result?: string;
  }): Promise<void> {
    try {
      const params = toolCall.parameters ? JSON.parse(toolCall.parameters) : {};
      const filePath = params.file_path || params.path;

      if (!filePath) return;

      // Create a memory about the file operation
      const content = `File operation: ${toolCall.tool_name} on ${filePath}`;
      await this.apiClient.storeMemory(content, {
        domain: 'file-operations',
        source: 'claude-chat-stream',
        importance: 4,
        tags: ['file', toolCall.tool_name],
      });
    } catch {
      // Skip malformed tool calls
    }
  }

  private isFileOperation(toolName: string): boolean {
    const fileOps = ['Read', 'Write', 'Edit', 'MultiEdit', 'Glob', 'Grep'];
    return fileOps.includes(toolName);
  }

  private calculateImportance(content: string, role: 'user' | 'assistant'): number {
    let importance = 5;

    // Length-based adjustments
    if (content.length > 1000) importance += 1;
    if (content.length > 3000) importance += 1;

    // Content-based adjustments
    if (role === 'user') {
      // Questions tend to be important
      if (content.includes('?')) importance += 1;
      // Requirements/specifications
      if (/\b(must|should|need|require)\b/i.test(content)) importance += 1;
    } else {
      // Explanations with examples
      if (/\bfor example\b/i.test(content)) importance += 1;
      // Step-by-step instructions
      if (/\b(step \d|first|second|third)\b/i.test(content)) importance += 1;
    }

    // Code presence
    if (/```/.test(content)) importance += 1;

    return Math.min(importance, 10);
  }

  private inferDomain(content: string, context: string): string {
    const combined = `${content} ${context}`.toLowerCase();

    // Technology domains
    if (/\b(react|vue|angular|svelte)\b/.test(combined)) return 'frontend';
    if (/\b(node|express|fastify|nestjs)\b/.test(combined)) return 'backend';
    if (/\b(sql|postgres|mysql|mongodb|database)\b/.test(combined)) return 'database';
    if (/\b(docker|kubernetes|k8s|aws|gcp|azure)\b/.test(combined)) return 'devops';
    if (/\b(test|jest|mocha|cypress|playwright)\b/.test(combined)) return 'testing';
    if (/\b(typescript|javascript|python|go|rust)\b/.test(combined)) return 'programming';

    return 'general';
  }

  private extractTags(content: string): string[] {
    const tags: string[] = [];

    // Programming languages
    const languages = ['typescript', 'javascript', 'python', 'go', 'rust', 'java', 'c++', 'ruby'];
    for (const lang of languages) {
      if (new RegExp(`\\b${lang}\\b`, 'i').test(content)) {
        tags.push(lang);
      }
    }

    // Frameworks
    const frameworks = ['react', 'vue', 'angular', 'express', 'fastify', 'django', 'flask'];
    for (const fw of frameworks) {
      if (new RegExp(`\\b${fw}\\b`, 'i').test(content)) {
        tags.push(fw);
      }
    }

    return tags.slice(0, 5); // Limit to 5 tags
  }

  private extractCodeBlocks(content: string): Array<{ language?: string; code: string }> {
    const codeBlockRegex = /```(\w*)\n?([\s\S]*?)```/g;
    const blocks: Array<{ language?: string; code: string }> = [];

    let match;
    while ((match = codeBlockRegex.exec(content)) !== null) {
      blocks.push({
        language: match[1] || undefined,
        code: match[2].trim(),
      });
    }

    return blocks;
  }

  private extractExplanation(content: string): string {
    // Remove code blocks for explanation extraction
    let explanation = content.replace(/```[\s\S]*?```/g, '');

    // Remove tool call formatting
    explanation = explanation.replace(/<[\s\S]*?<\/antml:[^>]+>/g, '');

    // Trim and clean
    explanation = explanation.trim();

    return explanation;
  }

  private notifyProgress(job: ExtractionJob): void {
    if (this.onProgress) {
      this.onProgress(job);
    }
  }
}

// Crypto UUID polyfill for Node
function uuidv4(): string {
  return require('crypto').randomUUID();
}
