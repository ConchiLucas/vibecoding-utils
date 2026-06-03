import React, { useState, useRef, useEffect, useCallback } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { X, Trash2, Send, Sparkles, Bot, User, Wrench, Cpu, History, MessageSquare, Clock3 } from 'lucide-react';
import {
  sendAIChatStream,
  getAIProviders,
  getAIChatHistoryList,
  saveAIChatHistory,
  AIChatMessage,
  AIProvider,
  AIChatHistoryItem,
} from '../../api/aiChat';
import {
  buildMinimalAIChatMessages,
  createChatSnapshot,
  getHistoryRequestContext,
} from './AIChatWidgetHistory';
import {
  buildAIChatRequestTextWithContext,
  inferAIChatMode,
  AIChatMode,
} from './AIChatWidgetIntent';
import { useProjectStore } from '../../stores/useProjectStore';
import './AIChatWidget.css';

// ─── Types ──────────────────────────────────────────
interface DisplayMessage {
  id: string;
  role: 'user' | 'assistant' | 'tool_status';
  content: string;
  mode?: AIChatMode;
  apiContent?: string;
  rawContent?: string;
  timestamp: Date;
}

// ─── Quick Actions ──────────────────────────────────
const QUICK_ACTIONS: Array<{ label: string; prompt?: string; mode?: 'deploy_info' }> = [
  { label: '📋 查看项目列表', prompt: '帮我查看现在有哪些部署项目' },
  { label: '🔍 扫描项目目录', prompt: '帮我扫描一下 /Users/conchi/workforce 下的某个项目' },
  { label: '🐳 生成部署配置', mode: 'deploy_info' },
  { label: '💡 部署帮助', prompt: '如何优化 Docker 镜像大小？' },
];

function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function formatTime(date: Date) {
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

function addProjectContext(text: string, activeProjectId: number | null, activeProject: string, activeConnectionId: number | null) {
  if (!activeProjectId) return text;
  const connectionText = activeConnectionId ? `activeConnectionId=${activeConnectionId}` : 'activeConnectionId=未选择';
  return `当前项目上下文：projectConfigId=${activeProjectId}，projectName=${activeProject || '未命名'}，${connectionText}。\n${text}`;
}

const TOOL_NAME_MAP: Record<string, string> = {
  scan_project: '📂 扫描项目目录',
  create_deploy_project: '🚀 创建部署项目',
  generate_deploy_info: '🐳 生成部署信息',
  create_project_group: '📁 创建项目组',
  auto_create_deploy_project: '🚀 创建部署项目',
  list_projects: '📋 获取项目列表',
  import_table_relations: '🔗 导入表血缘关系',
};

// ═════════════════════════════════════════════════════
// Main Widget Component — Real AI Integration
// ═════════════════════════════════════════════════════
export default function AIChatWidget() {
  const [isOpen, setIsOpen] = useState(false);
  const [displayMessages, setDisplayMessages] = useState<DisplayMessage[]>([]);
  const [requestContext, setRequestContext] = useState<{ mode: AIChatMode; prompt?: string }>({ mode: 'chat' });
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [selectedProvider, setSelectedProvider] = useState('');
  const [providerError, setProviderError] = useState('');
  const [activeMode, setActiveMode] = useState<AIChatMode>('chat');
  const [isHistoryView, setIsHistoryView] = useState(false);
  const [historyItems, setHistoryItems] = useState<AIChatHistoryItem[]>([]);
  const [historyError, setHistoryError] = useState('');
  const { activeProject, activeProjectId, activeConnectionId } = useProjectStore();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const currentChatIdRef = useRef<string | null>(null);

  const refreshHistoryItems = useCallback(async () => {
    try {
      const data = await getAIChatHistoryList(20);
      setHistoryItems(data.list || []);
      setHistoryError('');
    } catch (err: any) {
      setHistoryError(err.message || '历史记录加载失败');
    }
  }, []);

  // Listen for toggle event from Header button
  useEffect(() => {
    const handler = () => setIsOpen(prev => !prev);
    window.addEventListener('toggle-ai-chat', handler);
    return () => window.removeEventListener('toggle-ai-chat', handler);
  }, []);

  // Auto-scroll to bottom
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [displayMessages]);

  // Focus textarea when modal opens
  useEffect(() => {
    if (isOpen && textareaRef.current) {
      setTimeout(() => textareaRef.current?.focus(), 300);
    }
  }, [isOpen]);

  useEffect(() => {
    if (isOpen) {
      refreshHistoryItems();
    }
  }, [isOpen, refreshHistoryItems]);

  useEffect(() => {
    if (!isOpen) return;
    let cancelled = false;

    getAIProviders()
      .then((data) => {
        if (cancelled) return;
        const nextProviders = data.providers || [];
        const savedProvider = localStorage.getItem('ai-chat-provider') || '';
        const activeProvider = nextProviders.find(provider => provider.id === savedProvider)
          ? savedProvider
          : nextProviders.find(provider => provider.active)?.id || data.active || nextProviders[0]?.id || '';

        setProviders(nextProviders);
        setSelectedProvider(activeProvider);
        setProviderError('');
      })
      .catch((err: Error) => {
        if (cancelled) return;
        setProviderError(err.message || 'AI 厂商加载失败');
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen]);

  // ESC to close
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) setIsOpen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [isOpen]);

  // Auto-resize textarea
  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInputValue(e.target.value);
    const ta = e.target;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 120) + 'px';
  };

  // Send message
  const handleSend = useCallback(async (overrideText?: string, displayText?: string) => {
    const rawText = (overrideText || inputValue).trim();
    if (!rawText || isLoading) return;

    const contextPrompt = requestContext.mode === 'deploy_info' ? requestContext.prompt : undefined;
    const baseText = overrideText ? rawText : buildAIChatRequestTextWithContext(rawText, activeMode, contextPrompt);
    const messageMode = inferAIChatMode(contextPrompt ? `${contextPrompt}\n${rawText}` : rawText, activeMode);
    const text = messageMode === 'deploy_info'
      ? baseText
      : addProjectContext(baseText, activeProjectId, activeProject, activeConnectionId);
    const userContent = displayText || rawText;

    // Add user message to display
    const userMsg: DisplayMessage = {
      id: generateId(),
      role: 'user',
      content: userContent,
      mode: messageMode,
      apiContent: text,
      rawContent: rawText,
      timestamp: new Date(),
    };
    setDisplayMessages(prev => [...prev, userMsg]);
    setInputValue('');
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }

    const requestMessages: AIChatMessage[] = buildMinimalAIChatMessages(text);

    // Start loading
    setIsLoading(true);

    // Create assistant message placeholder
    const assistantId = generateId();
    const assistantMsg: DisplayMessage = {
      id: assistantId,
      role: 'assistant',
      content: '',
      timestamp: new Date(),
    };
    setDisplayMessages(prev => [...prev, assistantMsg]);

    // Abort controller for cancellation
    const abortController = new AbortController();
    abortRef.current = abortController;

    let fullContent = '';

    await sendAIChatStream(
      requestMessages,
      selectedProvider || undefined,
      {
        onContent: (chunk) => {
          fullContent += chunk;
          setDisplayMessages(prev =>
            prev.map(m => m.id === assistantId ? { ...m, content: fullContent } : m)
          );
        },
        onToolCall: (name, _args) => {
          const toolMsg: DisplayMessage = {
            id: generateId(),
            role: 'tool_status',
            content: `正在调用 ${TOOL_NAME_MAP[name] || name} ...`,
            timestamp: new Date(),
          };
          setDisplayMessages(prev => {
            // Insert before the assistant placeholder
            const idx = prev.findIndex(m => m.id === assistantId);
            if (idx >= 0) {
              const copy = [...prev];
              copy.splice(idx, 0, toolMsg);
              return copy;
            }
            return [...prev, toolMsg];
          });
        },
        onToolResult: (name, result) => {
          const toolLabel = TOOL_NAME_MAP[name] || name;
          const failed = result.startsWith('工具执行失败:');
          const content = failed
            ? `❌ ${toolLabel} 失败：${result.replace(/^工具执行失败:\s*/, '')}`
            : `✅ ${toolLabel} 执行完成`;
          // Update the tool_status message
          setDisplayMessages(prev =>
            prev.map(m =>
              m.role === 'tool_status' && m.content.includes(toolLabel)
                ? { ...m, content }
                : m
            )
          );
        },
        onError: (error) => {
          setDisplayMessages(prev =>
            prev.map(m => m.id === assistantId
              ? { ...m, content: `❌ 出错了: ${error}` }
              : m)
          );
        },
        onDone: () => {
          // Save assistant response to conversation history
          if (fullContent) {
            setRequestContext(prev => {
              if (messageMode === 'deploy_info') {
                return { mode: 'deploy_info', prompt: contextPrompt || rawText };
              }
              return prev.mode === 'deploy_info' ? prev : { mode: 'chat' };
            });
            const snapshot = createChatSnapshot(
              [
                ...displayMessages,
                userMsg,
                { ...assistantMsg, content: fullContent, timestamp: new Date() },
              ],
              currentChatIdRef.current || undefined,
              selectedProvider || undefined
            );
            saveAIChatHistory(snapshot)
              .then((saved) => {
                currentChatIdRef.current = saved.chatId;
                return refreshHistoryItems();
              })
              .catch((err: Error) => {
                setHistoryError(err.message || '历史记录保存失败');
              });
          }
        },
      },
      abortController.signal
    );

    setIsLoading(false);
    abortRef.current = null;
  }, [inputValue, isLoading, requestContext, selectedProvider, activeMode, displayMessages, refreshHistoryItems, activeProjectId, activeProject, activeConnectionId]);

  const handleProviderChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const providerID = e.target.value;
    setSelectedProvider(providerID);
    localStorage.setItem('ai-chat-provider', providerID);
  };

  const currentProvider = providers.find(provider => provider.id === selectedProvider);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleClear = () => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    setDisplayMessages([]);
    setRequestContext({ mode: 'chat' });
    setIsLoading(false);
    setActiveMode('chat');
    setIsHistoryView(false);
    currentChatIdRef.current = null;
  };

  const activateDeployInfoMode = () => {
    setActiveMode('deploy_info');
    setInputValue('');
    setTimeout(() => textareaRef.current?.focus(), 0);
  };

  const openHistoryView = () => {
    refreshHistoryItems();
    setIsHistoryView(true);
  };

  const backToChat = () => {
    setIsHistoryView(false);
    setTimeout(() => textareaRef.current?.focus(), 0);
  };

  const restoreHistoryItem = (item: AIChatHistoryItem) => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    currentChatIdRef.current = item.chatId;
    setDisplayMessages(item.messages.map(message => ({
      ...message,
      timestamp: new Date(message.timestamp),
    })));
    const restoredContext = getHistoryRequestContext(item.messages);
    setRequestContext(restoredContext);
    setActiveMode(restoredContext.mode);
    setIsLoading(false);
    setIsHistoryView(false);
    setTimeout(() => textareaRef.current?.focus(), 0);
  };

  const inputPlaceholder = activeMode === 'deploy_info'
    ? '输入项目目录，也可以补充：组名、项目名、再生成一份'
    : '输入你的问题，例如：帮我查看有哪些部署项目';

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="ai-chat-overlay open">
          {/* Backdrop */}
          <motion.div
            className="ai-chat-backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            onClick={() => setIsOpen(false)}
          />

          {/* Center Modal */}
          <motion.div
            className="ai-chat-modal"
            initial={{ opacity: 0, scale: 0.92, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.92, y: 20 }}
            transition={{ type: 'spring', stiffness: 350, damping: 28 }}
          >
            {/* Header */}
            <div className="ai-chat-header">
              <div className="ai-chat-header-left">
                <div className="ai-chat-header-avatar">
                  <Bot size={20} />
                </div>
                <div className="ai-chat-header-info">
                  <h3>AI 部署助手</h3>
                  <p><span className="status-dot" /> 在线 · Function Calling 已启用</p>
                </div>
              </div>
              <div className="ai-chat-header-actions">
                <button
                  className={`ai-chat-header-action-text ${isHistoryView ? 'active' : ''}`}
                  onClick={isHistoryView ? backToChat : openHistoryView}
                  title={isHistoryView ? '返回当前对话' : '查询历史记录'}
                  disabled={isLoading && !isHistoryView}
                >
                  {isHistoryView ? <MessageSquare size={15} /> : <History size={15} />}
                  <span>{isHistoryView ? '当前对话' : '历史记录'}</span>
                </button>
                <label className="ai-provider-select" title={currentProvider ? `${currentProvider.model} · ${currentProvider.type}` : providerError || '选择 AI 厂商'}>
                  <Cpu size={14} />
                  <select
                    value={selectedProvider}
                    onChange={handleProviderChange}
                    disabled={isLoading || providers.length === 0}
                    aria-label="选择 AI 厂商"
                  >
                    {providers.length === 0 ? (
                      <option value="">{providerError || '加载中'}</option>
                    ) : (
                      providers.map((provider) => (
                        <option key={provider.id} value={provider.id}>
                          {provider.label || provider.id}
                        </option>
                      ))
                    )}
                  </select>
                </label>
                <button className="ai-chat-header-btn" onClick={handleClear} title="清空对话">
                  <Trash2 size={16} />
                </button>
                <button className="ai-chat-header-btn" onClick={() => setIsOpen(false)} title="关闭">
                  <X size={18} />
                </button>
              </div>
            </div>

            {/* Messages */}
            <div className={`ai-chat-messages ${isHistoryView ? 'history-view' : ''}`}>
              {isHistoryView ? (
                <div className="ai-chat-history-panel">
                  <div className="ai-chat-history-title">
                    <div>
                      <h4>历史记录</h4>
                      <p>最近 20 条对话保存在后端数据库，点击即可恢复继续聊。</p>
                    </div>
                    <span>{historyItems.length} 条</span>
                  </div>
                  {historyError ? (
                    <div className="ai-chat-history-empty">
                      <History size={32} />
                      <strong>历史记录加载失败</strong>
                      <span>{historyError}</span>
                    </div>
                  ) : historyItems.length === 0 ? (
                    <div className="ai-chat-history-empty">
                      <History size={32} />
                      <strong>还没有历史记录</strong>
                      <span>完成一次对话后，这里会自动保存。</span>
                    </div>
                  ) : (
                    <div className="ai-chat-history-list">
                      {historyItems.map((item) => (
                        <button
                          key={item.chatId}
                          className="ai-chat-history-item"
                          onClick={() => restoreHistoryItem(item)}
                        >
                          <div className="ai-chat-history-item-main">
                            <strong>{item.title}</strong>
                            <span>{item.messages.find(message => message.role === 'assistant')?.content || '暂无回复内容'}</span>
                          </div>
                          <div className="ai-chat-history-item-meta">
                            <Clock3 size={13} />
                            <span>{formatTime(new Date(item.updatedAt))}</span>
                            <small>{item.messages.filter(message => message.role !== 'tool_status').length} 条</small>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ) : displayMessages.length === 0 && !isLoading ? (
                <div className="ai-chat-welcome">
                  <div className="ai-chat-welcome-icon">
                    <Sparkles size={36} />
                  </div>
                  <h4>嗨，我是你的 AI 部署助手</h4>
                  <p>我可以帮你扫描项目、创建部署配置、查看项目列表，还能回答运维问题。试试下面的快捷操作吧 👇</p>
                  <div className="ai-chat-quick-actions">
                    {QUICK_ACTIONS.map((action) => (
                      <button
                        key={action.label}
                        className="ai-chat-quick-chip"
                        onClick={() => action.mode === 'deploy_info' ? activateDeployInfoMode() : handleSend(action.prompt || '')}
                      >
                        {action.label}
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <>
                  {displayMessages.map((msg) => {
                    // Tool status messages
                    if (msg.role === 'tool_status') {
                      return (
                        <div key={msg.id} className="ai-tool-status">
                          <Wrench size={13} className="ai-tool-status-icon" />
                          <span>{msg.content}</span>
                        </div>
                      );
                    }

                    return (
                      <div key={msg.id} className={`ai-msg ${msg.role}`}>
                        <div className="ai-msg-avatar">
                          {msg.role === 'assistant' ? <Bot size={14} /> : <User size={14} />}
                        </div>
                        <div>
                          <div className="ai-msg-bubble">
                            {msg.role === 'assistant' ? (
                              <SimpleMarkdown
                                content={msg.content}
                                isStreaming={isLoading && msg === displayMessages[displayMessages.length - 1]}
                              />
                            ) : (
                              msg.content
                            )}
                          </div>
                          <div className="ai-msg-time">{formatTime(msg.timestamp)}</div>
                        </div>
                      </div>
                    );
                  })}

                  {isLoading && displayMessages[displayMessages.length - 1]?.content === '' && (
                    <div className="ai-typing">
                      <div className="ai-msg-avatar" style={{
                        width: 28, height: 28, borderRadius: 10,
                        background: 'linear-gradient(135deg, #6366f1, #a855f7)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: 'white', flexShrink: 0,
                      }}>
                        <Bot size={14} />
                      </div>
                      <div className="ai-typing-dots">
                        <span /><span /><span />
                      </div>
                    </div>
                  )}
                </>
              )}
              <div ref={messagesEndRef} />
            </div>

            {/* Input Area */}
            {!isHistoryView && (
              <div className="ai-chat-input-area">
              {activeMode === 'deploy_info' && (
                <div className="ai-chat-mode-panel">
                  <div>
                    <strong>生成部署信息</strong>
                    <span>输入项目目录即可生成，也可以补充组名、项目名或“再生成一份”。</span>
                  </div>
                  <button type="button" onClick={() => setActiveMode('chat')}>退出</button>
                </div>
              )}
              <div className="ai-chat-input-wrapper">
                <textarea
                  ref={textareaRef}
                  value={inputValue}
                  onChange={handleTextareaChange}
                  onKeyDown={handleKeyDown}
                  placeholder={inputPlaceholder}
                  rows={1}
                  disabled={isLoading}
                />
                <button
                  className="ai-chat-send-btn"
                  onClick={() => handleSend()}
                  disabled={!inputValue.trim() || isLoading}
                  title="发送"
                >
                  <Send size={16} />
                </button>
              </div>
              <div className="ai-chat-input-hint">
                <kbd>Enter</kbd> 发送 · <kbd>Shift + Enter</kbd> 换行 · <kbd>Esc</kbd> 关闭
              </div>
              </div>
            )}
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}

// ─── Simple Markdown Renderer ───────────────────────
function SimpleMarkdown({ content, isStreaming }: { content: string; isStreaming: boolean }) {
  if (!content) {
    return isStreaming ? <span style={{
      display: 'inline-block', width: 2, height: '1em',
      background: '#6366f1', animation: 'cursorBlink 1s step-end infinite',
    }} /> : null;
  }

  const parts = content.split(/(```[\s\S]*?```)/g);

  return (
    <>
      {parts.map((part, i) => {
        if (part.startsWith('```') && part.endsWith('```')) {
          const lines = part.slice(3, -3).split('\n');
          const lang = lines[0]?.trim() || '';
          const code = lang ? lines.slice(1).join('\n') : lines.join('\n');
          return (
            <pre key={i}>
              <code>{code}</code>
            </pre>
          );
        }
        return (
          <span key={i}>
            {part.split('\n').map((line, j, arr) => (
              <React.Fragment key={j}>
                <InlineMarkdown text={line} />
                {j < arr.length - 1 && <br />}
              </React.Fragment>
            ))}
          </span>
        );
      })}
      {isStreaming && <span style={{
        display: 'inline-block', width: 2, height: '1em',
        background: '#6366f1', marginLeft: 2,
        animation: 'cursorBlink 1s step-end infinite',
        verticalAlign: 'text-bottom',
      }} />}
      <style>{`@keyframes cursorBlink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }`}</style>
    </>
  );
}

function InlineMarkdown({ text }: { text: string }) {
  const tokens: React.ReactNode[] = [];
  let remaining = text;
  let key = 0;

  while (remaining.length > 0) {
    const codeMatch = remaining.match(/^`([^`]+)`/);
    if (codeMatch) {
      tokens.push(<code key={key++}>{codeMatch[1]}</code>);
      remaining = remaining.slice(codeMatch[0].length);
      continue;
    }
    const boldMatch = remaining.match(/^\*\*([^*]+)\*\*/);
    if (boldMatch) {
      tokens.push(<strong key={key++}>{boldMatch[1]}</strong>);
      remaining = remaining.slice(boldMatch[0].length);
      continue;
    }
    const nextSpecial = remaining.search(/[`*]/);
    if (nextSpecial === -1) {
      tokens.push(remaining);
      break;
    } else if (nextSpecial === 0) {
      tokens.push(remaining[0]);
      remaining = remaining.slice(1);
    } else {
      tokens.push(remaining.slice(0, nextSpecial));
      remaining = remaining.slice(nextSpecial);
    }
  }

  return <>{tokens}</>;
}
