import { useUserStore } from '../stores/useUserStore';
import service from '../utils/request';

const BASE_URL = import.meta.env.VITE_BASE_API || '/api';

export interface AIChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface AIStreamCallbacks {
  onContent: (text: string) => void;
  onToolCall: (name: string, args: string) => void;
  onToolResult: (name: string, result: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}

export interface AIProvider {
  id: string;
  label: string;
  type: string;
  base_url: string;
  model: string;
  max_tokens: number;
  active: boolean;
}

export interface AIProvidersResponse {
  active: string;
  providers: AIProvider[];
}

export interface AIProviderConfigItem {
  id: string;
  label: string;
  type: string;
  base_url: string;
  api_key: string;
  model: string;
  max_tokens: number;
}

export interface AIConfigResponse {
  active: string;
  providers: AIProviderConfigItem[];
}

export interface AIChatHistoryMessage {
  id: string;
  role: 'user' | 'assistant' | 'tool_status';
  content: string;
  mode?: 'chat' | 'deploy_info';
  apiContent?: string;
  rawContent?: string;
  timestamp: string;
}

export interface AIChatHistoryItem {
  ID?: number;
  chatId: string;
  title: string;
  provider?: string;
  createdAt: string;
  updatedAt: string;
  messages: AIChatHistoryMessage[];
}

export interface AIChatHistoryListResponse {
  list: AIChatHistoryItem[];
  total: number;
}

interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

async function requestAIChatJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const { token } = useUserStore.getState();
  const response = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'x-token': token || '',
      ...(init?.headers || {}),
    },
  });

  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const result = await response.json() as ApiResponse<T>;
  if (result.code !== 0) {
    throw new Error(result.msg || '请求失败');
  }
  return result.data;
}

export async function getAIProviders(): Promise<AIProvidersResponse> {
  const { token } = useUserStore.getState();
  const response = await fetch(`${BASE_URL}/ai/providers`, {
    headers: {
      'x-token': token || '',
    },
  });

  if (!response.ok) {
    throw new Error(`获取 AI 厂商失败 (${response.status})`);
  }

  const result = await response.json();
  if (result.code !== 0) {
    throw new Error(result.msg || '获取 AI 厂商失败');
  }

  return result.data;
}

export const getAIConfig = () => {
  return service.get<any, ApiResponse<AIConfigResponse>>('/ai/config');
};

export const saveAIConfig = (data: AIConfigResponse) => {
  return service.post<any, ApiResponse<AIProvider[]>>('/ai/config', data);
};

export async function getAIChatHistoryList(limit = 20): Promise<AIChatHistoryListResponse> {
  return requestAIChatJSON<AIChatHistoryListResponse>(`/ai/chat/history?limit=${limit}`);
}

export async function saveAIChatHistory(history: AIChatHistoryItem): Promise<AIChatHistoryItem> {
  return requestAIChatJSON<AIChatHistoryItem>('/ai/chat/history', {
    method: 'POST',
    body: JSON.stringify(history),
  });
}

/**
 * 发送 AI 对话请求（SSE 流式）
 *
 * Gin 的 SSEvent 输出格式为:
 *   event:content\ndata:JSON_ENCODED_VALUE\n\n
 * 其中 data 的值会被 Gin 的 sse.Encode JSON 编码（字符串会被加引号）
 */
export async function sendAIChatStream(
  messages: AIChatMessage[],
  provider: string | undefined,
  callbacks: AIStreamCallbacks,
  signal?: AbortSignal
): Promise<void> {
  const { token } = useUserStore.getState();

  let response: Response;
  try {
    response = await fetch(`${BASE_URL}/ai/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-token': token || '',
      },
      body: JSON.stringify({ messages, provider }),
      signal,
    });
  } catch (err: any) {
    if (err.name === 'AbortError') {
      callbacks.onDone();
      return;
    }
    callbacks.onError('网络请求失败: ' + (err.message || ''));
    return;
  }

  if (!response.ok) {
    const text = await response.text();
    callbacks.onError(`请求失败 (${response.status}): ${text}`);
    return;
  }

  const reader = response.body?.getReader();
  if (!reader) {
    callbacks.onError('浏览器不支持流式响应');
    return;
  }

  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // SSE 格式: 以 \n\n 分隔的事件块
      // 每个块包含 event:xxx\ndata:yyy
      const blocks = buffer.split('\n\n');
      // 最后一个可能是不完整的块，保留在 buffer 中
      buffer = blocks.pop() || '';

      for (const block of blocks) {
        if (!block.trim()) continue;

        const lines = block.split(/\r?\n/);
        let eventType = 'message';
        const dataLines: string[] = [];

        for (const line of lines) {
          if (line.startsWith('event:')) {
            eventType = line.slice(6).trim();
          } else if (line.startsWith('data:')) {
            dataLines.push(line.slice(5).replace(/^[ \t]/, ''));
          }
        }

        if (dataLines.length === 0 && !eventType) continue;

        // Gin 的 sse.Encode 会对 string 类型做 JSON 编码（加引号+转义）
        // 尝试 JSON.parse 来解码
        const dataStr = dataLines.join('\n');
        let cleanData = dataStr;
        try {
          cleanData = JSON.parse(dataStr);
        } catch {
          // 如果不是有效 JSON，使用原始值
          cleanData = dataStr;
        }

        switch (eventType) {
          case 'content':
            if (typeof cleanData === 'string') {
              callbacks.onContent(cleanData);
            }
            break;
          case 'tool_call': {
            try {
              const info = typeof cleanData === 'string' ? JSON.parse(cleanData) : cleanData;
              callbacks.onToolCall(info.name, info.arguments);
            } catch {
              callbacks.onToolCall('unknown', String(cleanData));
            }
            break;
          }
          case 'tool_result': {
            try {
              const info = typeof cleanData === 'string' ? JSON.parse(cleanData) : cleanData;
              callbacks.onToolResult(info.name, info.result);
            } catch {
              callbacks.onToolResult('unknown', String(cleanData));
            }
            break;
          }
          case 'error': {
            try {
              const info = typeof cleanData === 'string' ? JSON.parse(cleanData) : cleanData;
              callbacks.onError(info.error || String(cleanData));
            } catch {
              callbacks.onError(String(cleanData));
            }
            break;
          }
          case 'done':
            callbacks.onDone();
            return;
        }
      }
    }
    // Stream 结束但没有 done 事件
    callbacks.onDone();
  } catch (err: any) {
    if (err.name === 'AbortError') {
      callbacks.onDone();
    } else {
      callbacks.onError(err.message || '流式读取失败');
    }
  }
}
