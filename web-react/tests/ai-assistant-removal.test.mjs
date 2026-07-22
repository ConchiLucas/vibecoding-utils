import assert from 'node:assert/strict';
import { access, readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import test from 'node:test';

const projectRoot = fileURLToPath(new URL('../', import.meta.url));
const sourcePath = (...parts) => path.join(projectRoot, 'src', ...parts);
const sourceText = (...parts) => readFile(sourcePath(...parts), 'utf8');

async function assertMissing(...parts) {
  await assert.rejects(access(sourcePath(...parts)), (error) => error?.code === 'ENOENT');
}

test('AI assistant UI and mixed chat API are removed', async () => {
  const header = await sourceText('views', 'layout', 'Header.tsx');
  const layout = await sourceText('views', 'layout', 'Layout.tsx');

  assert.doesNotMatch(header, /toggle-ai-chat/);
  assert.doesNotMatch(layout, /AIChatWidget/);
  await assertMissing('components', 'AIChatWidget');
  await assertMissing('api', 'aiChat.ts');
});

test('AI configuration and table-sample generation remain available', async () => {
  const aiConfig = await sourceText('api', 'aiConfig.ts');
  const configManager = await sourceText('views', 'config-manager', 'AIConfigManager.tsx');
  const tableDataPreview = await sourceText('components', 'TableDataPreview.tsx');

  assert.match(aiConfig, /export const getAIConfig/);
  assert.match(aiConfig, /export const saveAIConfig/);
  assert.match(aiConfig, /export const saveAIActiveProvider/);
  assert.match(configManager, /\.\.\/\.\.\/api\/aiConfig/);
  assert.match(tableDataPreview, /generateRemoteTableData/);
  assert.match(tableDataPreview, /AI 造数/);
});
