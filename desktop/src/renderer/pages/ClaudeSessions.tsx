import { useState, useEffect } from 'react';
import {
  MessageSquare,
  FolderOpen,
  ChevronRight,
  Clock,
  GitBranch,
  Wrench,
  User,
  Bot,
  Download,
  Search,
  RefreshCw,
} from 'lucide-react';
import type { ClaudeProject, ClaudeSession, ClaudeMessage, ClaudeToolCall } from '../../shared/types';

interface ProjectCardProps {
  project: ClaudeProject;
  isSelected: boolean;
  onClick: () => void;
}

function ProjectCard({ project, isSelected, onClick }: ProjectCardProps) {
  return (
    <div
      onClick={onClick}
      className={`p-3 rounded-lg cursor-pointer transition-all ${
        isSelected
          ? 'bg-primary-500/20 border border-primary-500'
          : 'bg-slate-800 border border-slate-700 hover:border-slate-600'
      }`}
    >
      <div className="flex items-center gap-3">
        <FolderOpen className="w-5 h-5 text-amber-400" />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate">{project.display_name || project.original_path.split('/').pop()}</p>
          <p className="text-xs text-slate-400">{project.session_count} sessions</p>
        </div>
      </div>
    </div>
  );
}

interface SessionCardProps {
  session: ClaudeSession;
  isSelected: boolean;
  onClick: () => void;
  onExtract: () => void;
}

function SessionCard({ session, isSelected, onClick, onExtract }: SessionCardProps) {
  return (
    <div
      className={`p-4 rounded-lg transition-all ${
        isSelected
          ? 'bg-primary-500/20 border border-primary-500'
          : 'bg-slate-800 border border-slate-700 hover:border-slate-600'
      }`}
    >
      <div className="cursor-pointer" onClick={onClick}>
        <p className="text-sm line-clamp-2 mb-2">
          {session.summary || session.first_prompt || 'No summary'}
        </p>
        <div className="flex items-center gap-3 text-xs text-slate-400">
          <span className="flex items-center gap-1">
            <MessageSquare className="w-3 h-3" />
            {session.message_count}
          </span>
          <span className="flex items-center gap-1">
            <Wrench className="w-3 h-3" />
            {session.tool_call_count}
          </span>
          {session.git_branch && (
            <span className="flex items-center gap-1">
              <GitBranch className="w-3 h-3" />
              {session.git_branch}
            </span>
          )}
        </div>
        <div className="flex items-center justify-between mt-2">
          <span className="text-xs text-slate-500">
            {new Date(session.started_at).toLocaleDateString()}
          </span>
          {session.is_subagent && (
            <span className="text-xs bg-purple-500/20 text-purple-400 px-2 py-0.5 rounded">
              Subagent
            </span>
          )}
        </div>
      </div>
      <div className="mt-3 pt-3 border-t border-slate-700">
        <button
          onClick={(e) => {
            e.stopPropagation();
            onExtract();
          }}
          className="w-full py-2 px-3 bg-primary-500/20 text-primary-400 rounded-lg hover:bg-primary-500/30 transition-colors flex items-center justify-center gap-2 text-sm"
        >
          <Download className="w-4 h-4" />
          Extract Memories
        </button>
      </div>
    </div>
  );
}

interface MessageItemProps {
  message: ClaudeMessage;
  toolCalls: ClaudeToolCall[];
}

function MessageItem({ message, toolCalls }: MessageItemProps) {
  const [expanded, setExpanded] = useState(false);
  const isUser = message.role === 'user';
  const relevantToolCalls = toolCalls.filter((tc) => tc.message_id === message.id);

  return (
    <div className={`p-4 rounded-lg ${isUser ? 'bg-slate-800' : 'bg-slate-800/50'}`}>
      <div className="flex items-start gap-3">
        <div
          className={`w-8 h-8 rounded-full flex items-center justify-center ${
            isUser ? 'bg-blue-500/20 text-blue-400' : 'bg-primary-500/20 text-primary-400'
          }`}
        >
          {isUser ? <User className="w-4 h-4" /> : <Bot className="w-4 h-4" />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-medium text-sm">{isUser ? 'User' : 'Claude'}</span>
            <span className="text-xs text-slate-500">
              {new Date(message.timestamp).toLocaleTimeString()}
            </span>
          </div>
          <div
            className={`text-sm text-slate-300 whitespace-pre-wrap ${
              !expanded && message.content.length > 500 ? 'line-clamp-3' : ''
            }`}
          >
            {message.content}
          </div>
          {message.content.length > 500 && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-xs text-primary-400 mt-2 hover:underline"
            >
              {expanded ? 'Show less' : 'Show more'}
            </button>
          )}

          {/* Tool Calls */}
          {relevantToolCalls.length > 0 && (
            <div className="mt-3 space-y-2">
              {relevantToolCalls.map((tc) => (
                <div key={tc.id} className="p-2 bg-slate-700/50 rounded text-xs">
                  <div className="flex items-center gap-2">
                    <Wrench className="w-3 h-3 text-amber-400" />
                    <span className="font-mono">{tc.tool_name}</span>
                    {tc.success !== undefined && (
                      <span
                        className={`px-1.5 py-0.5 rounded ${
                          tc.success ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                        }`}
                      >
                        {tc.success ? 'Success' : 'Failed'}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function ClaudeSessions() {
  const [projects, setProjects] = useState<ClaudeProject[]>([]);
  const [sessions, setSessions] = useState<ClaudeSession[]>([]);
  const [messages, setMessages] = useState<ClaudeMessage[]>([]);
  const [toolCalls, setToolCalls] = useState<ClaudeToolCall[]>([]);
  const [selectedProject, setSelectedProject] = useState<ClaudeProject | null>(null);
  const [selectedSession, setSelectedSession] = useState<ClaudeSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [extracting, setExtracting] = useState<string | null>(null);

  useEffect(() => {
    fetchProjects();
  }, []);

  useEffect(() => {
    if (selectedProject) {
      fetchSessions(selectedProject.id);
    }
  }, [selectedProject]);

  useEffect(() => {
    if (selectedSession) {
      fetchMessages(selectedSession.id);
    }
  }, [selectedSession]);

  async function fetchProjects() {
    try {
      setLoading(true);
      const response = await window.mycelicMemory.claude.projects();
      setProjects(response || []);
    } catch (err) {
      console.error('Failed to fetch projects:', err);
    } finally {
      setLoading(false);
    }
  }

  async function fetchSessions(projectId: string) {
    try {
      const response = await window.mycelicMemory.claude.sessions(projectId);
      setSessions(response || []);
    } catch (err) {
      console.error('Failed to fetch sessions:', err);
    }
  }

  async function fetchMessages(sessionId: string) {
    try {
      const [messagesRes, toolCallsRes] = await Promise.all([
        window.mycelicMemory.claude.messages(sessionId),
        window.mycelicMemory.claude.toolCalls(sessionId),
      ]);
      setMessages(messagesRes || []);
      setToolCalls(toolCallsRes || []);
    } catch (err) {
      console.error('Failed to fetch messages:', err);
    }
  }

  async function handleExtract(sessionId: string) {
    try {
      setExtracting(sessionId);
      await window.mycelicMemory.extraction.start(sessionId);
    } catch (err) {
      console.error('Extraction failed:', err);
    } finally {
      setExtracting(null);
    }
  }

  const filteredSessions = sessions.filter(
    (s) =>
      !searchQuery ||
      s.summary?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.first_prompt?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="h-screen flex">
      {/* Projects Panel */}
      <div className="w-64 border-r border-slate-700 flex flex-col bg-slate-900">
        <div className="p-4 border-b border-slate-700">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">Projects</h2>
            <button
              onClick={fetchProjects}
              className="p-1 hover:bg-slate-700 rounded transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-auto p-4 space-y-2">
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin w-6 h-6 border-2 border-primary-500 border-t-transparent rounded-full" />
            </div>
          ) : projects.length > 0 ? (
            projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                isSelected={selectedProject?.id === project.id}
                onClick={() => setSelectedProject(project)}
              />
            ))
          ) : (
            <p className="text-center text-slate-500 py-8 text-sm">
              No projects found. Make sure claude-chat-stream is running.
            </p>
          )}
        </div>
      </div>

      {/* Sessions Panel */}
      <div className="w-80 border-r border-slate-700 flex flex-col bg-slate-900">
        <div className="p-4 border-b border-slate-700">
          <h2 className="font-semibold mb-3">Sessions</h2>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search sessions..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>
        <div className="flex-1 overflow-auto p-4 space-y-3">
          {selectedProject ? (
            filteredSessions.length > 0 ? (
              filteredSessions.map((session) => (
                <SessionCard
                  key={session.id}
                  session={session}
                  isSelected={selectedSession?.id === session.id}
                  onClick={() => setSelectedSession(session)}
                  onExtract={() => handleExtract(session.id)}
                />
              ))
            ) : (
              <p className="text-center text-slate-500 py-8 text-sm">No sessions found</p>
            )
          ) : (
            <p className="text-center text-slate-500 py-8 text-sm">Select a project to view sessions</p>
          )}
        </div>
        <div className="p-4 border-t border-slate-700 text-sm text-slate-400">
          {filteredSessions.length} sessions
        </div>
      </div>

      {/* Messages Panel */}
      <div className="flex-1 flex flex-col bg-slate-900">
        {selectedSession ? (
          <>
            <div className="p-4 border-b border-slate-700">
              <h2 className="font-semibold">Conversation</h2>
              <p className="text-sm text-slate-400 mt-1">
                {selectedSession.message_count} messages • {selectedSession.tool_call_count} tool calls
              </p>
            </div>
            <div className="flex-1 overflow-auto p-4 space-y-4">
              {messages.map((message) => (
                <MessageItem key={message.id} message={message} toolCalls={toolCalls} />
              ))}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-slate-500">
            <div className="text-center">
              <MessageSquare className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Select a session to view messages</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
