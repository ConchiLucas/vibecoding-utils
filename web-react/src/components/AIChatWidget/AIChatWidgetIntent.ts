export type AIChatMode = 'chat' | 'deploy_info';

const DEPLOY_INFO_TOOL_PREFIX = '请调用 generate_deploy_info 生成部署信息，用户输入如下：';

const DEPLOY_INFO_INTENT_RE = /(生成|创建)(部署信息|部署配置|部署脚本|部署项目)|generate_deploy_info/i;
const LOCAL_ABSOLUTE_PATH_RE = /(?:^|[\s：:，,。；;])(?:\/(?:Users|Volumes|private|var|tmp|opt|home)\/[^\s，,。；;`"'的为]+|[A-Za-z]:\\[^\s，,。；;`"']+)/;

export function shouldForceGenerateDeployInfo(rawText: string): boolean {
  const text = rawText.trim();
  if (!text) return false;
  return DEPLOY_INFO_INTENT_RE.test(text) && LOCAL_ABSOLUTE_PATH_RE.test(text);
}

export function buildAIChatRequestText(rawText: string, activeMode: AIChatMode): string {
  const text = rawText.trim();
  if (activeMode === 'deploy_info' || shouldForceGenerateDeployInfo(text)) {
    return `${DEPLOY_INFO_TOOL_PREFIX}${text}`;
  }
  return text;
}

export function inferAIChatMode(rawText: string, activeMode: AIChatMode): AIChatMode {
  if (activeMode === 'deploy_info' || shouldForceGenerateDeployInfo(rawText)) {
    return 'deploy_info';
  }
  return 'chat';
}

export function buildAIChatRequestTextWithContext(
  rawText: string,
  activeMode: AIChatMode,
  contextPrompt?: string
): string {
  const text = rawText.trim();
  const context = (contextPrompt || '').trim();
  const mode = inferAIChatMode(context ? `${context}\n${text}` : text, activeMode);

  if (mode !== 'deploy_info') {
    return text;
  }

  if (context) {
    return `${DEPLOY_INFO_TOOL_PREFIX}${context}\n用户继续补充如下：${text}`;
  }
  return `${DEPLOY_INFO_TOOL_PREFIX}${text}`;
}
