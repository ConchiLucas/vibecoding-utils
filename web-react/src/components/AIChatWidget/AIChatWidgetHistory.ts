import { AIChatHistoryItem, AIChatHistoryMessage, AIChatMessage } from '../../api/aiChat';
import { AIChatMode } from './AIChatWidgetIntent';

function generateHistoryId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function normalizeTitle(content: string) {
  const title = content.replace(/\s+/g, ' ').trim();
  if (!title) return '新的对话';
  return title.length > 32 ? `${title.slice(0, 32)}...` : title;
}

function toIsoTimestamp(timestamp: string | Date) {
  return timestamp instanceof Date ? timestamp.toISOString() : timestamp;
}

export function createChatSnapshot(
  messages: Array<{
    id: string;
    role: 'user' | 'assistant' | 'tool_status';
    content: string;
    timestamp: string | Date;
  }>,
  existingChatId?: string,
  provider?: string
): AIChatHistoryItem {
  const serializableMessages = messages.map((message) => ({
    ...message,
    timestamp: toIsoTimestamp(message.timestamp),
  }));
  const firstUserMessage = serializableMessages.find(message => message.role === 'user');
  const firstTimestamp = serializableMessages[0]?.timestamp || new Date().toISOString();
  const lastTimestamp = serializableMessages[serializableMessages.length - 1]?.timestamp || firstTimestamp;

  return {
    chatId: existingChatId || generateHistoryId(),
    title: normalizeTitle(firstUserMessage?.content || serializableMessages[0]?.content || ''),
    provider,
    createdAt: firstTimestamp,
    updatedAt: lastTimestamp,
    messages: serializableMessages,
  };
}

export function toAIChatMessages(messages: AIChatHistoryMessage[]): AIChatMessage[] {
  const lastUser = [...messages].reverse().find(message => message.role === 'user');
  const lastAssistant = [...messages].reverse().find(message => message.role === 'assistant');
  return [lastUser, lastAssistant]
    .filter((message): message is AIChatHistoryMessage & { role: 'user' | 'assistant' } =>
      !!message && (message.role === 'user' || message.role === 'assistant')
    )
    .map(message => ({
      role: message.role,
      content: message.apiContent || message.content,
    }));
}

export function getHistoryRequestContext(messages: AIChatHistoryMessage[]): {
  mode: AIChatMode;
  prompt?: string;
} {
  const deployUserMessage = [...messages]
    .reverse()
    .find(message => message.role === 'user' && message.mode === 'deploy_info');

  if (deployUserMessage) {
    return {
      mode: 'deploy_info',
      prompt: deployUserMessage.rawContent || deployUserMessage.content,
    };
  }

  const latestUserMessage = [...messages].reverse().find(message => message.role === 'user');
  return {
    mode: latestUserMessage?.mode === 'deploy_info' ? 'deploy_info' : 'chat',
    prompt: latestUserMessage?.apiContent || latestUserMessage?.content,
  };
}

export function buildMinimalAIChatMessages(currentRequestText: string): AIChatMessage[] {
  return [{
    role: 'user',
    content: currentRequestText,
  }];
}

export function toLegacyAIChatMessages(messages: AIChatHistoryMessage[]): AIChatMessage[] {
  return messages
    .filter((message): message is AIChatHistoryMessage & { role: 'user' | 'assistant' } =>
      message.role === 'user' || message.role === 'assistant'
    )
    .map(message => ({
      role: message.role,
      content: message.content,
    }));
}
