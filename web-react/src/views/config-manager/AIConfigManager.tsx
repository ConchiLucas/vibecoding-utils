import React, { useEffect, useMemo, useState } from 'react';
import { Bot, CheckCircle2, Hash, KeyRound, Link, Plus, Save, Trash2 } from 'lucide-react';
import toast from 'react-hot-toast';
import { getAIConfig, saveAIConfig, AIProviderConfigItem } from '../../api/aiChat';
import { useConfirm } from '../../hooks/useConfirm';
import ConfirmDialog from '../../components/ConfirmDialog';

const createLocalId = () => `ai-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const createEmptyProvider = (index: number): AIProviderConfigItem => ({
  id: `provider-${index}`,
  label: '',
  type: 'openai-compatible',
  base_url: '',
  api_key: '',
  model: '',
  max_tokens: 4096,
});

export default function AIConfigManager() {
  const [active, setActive] = useState('');
  const [providers, setProviders] = useState<AIProviderConfigItem[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { confirm, dialogProps } = useConfirm();

  const selectedProvider = useMemo(
    () => providers.find(provider => provider.id === selectedId) || providers[0] || null,
    [providers, selectedId]
  );

  const loadConfig = async () => {
    setLoading(true);
    try {
      const res: any = await getAIConfig();
      if (res.code === 0) {
        const nextProviders = res.data?.providers || [];
        setActive(res.data?.active || nextProviders[0]?.id || '');
        setProviders(nextProviders);
        setSelectedId(res.data?.active || nextProviders[0]?.id || '');
      } else {
        toast.error(res.msg || 'AI 配置加载失败');
      }
    } catch {
      toast.error('AI 配置加载异常');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadConfig();
  }, []);

  const updateProvider = (id: string, patch: Partial<AIProviderConfigItem>) => {
    setProviders(prev => prev.map(provider => provider.id === id ? { ...provider, ...patch } : provider));
    if (patch.id && selectedId === id) {
      setSelectedId(patch.id);
    }
    if (patch.id && active === id) {
      setActive(patch.id);
    }
  };

  const handleAddProvider = () => {
    const next = createEmptyProvider(providers.length + 1);
    while (providers.some(provider => provider.id === next.id)) {
      next.id = createLocalId();
    }
    setProviders(prev => [...prev, next]);
    setSelectedId(next.id);
    if (!active) setActive(next.id);
  };

  const handleDeleteProvider = async (id: string) => {
    if (providers.length <= 1) {
      toast.error('至少保留一个 AI 配置');
      return;
    }
    const ok = await confirm('确定删除该 AI 配置吗？');
    if (!ok) return;
    setProviders(prev => {
      const next = prev.filter(provider => provider.id !== id);
      if (selectedId === id) setSelectedId(next[0]?.id || '');
      if (active === id) setActive(next[0]?.id || '');
      return next;
    });
  };

  const handleSetActive = (id: string) => {
    setActive(id);
    setSelectedId(id);
  };

  const validateConfig = () => {
    if (providers.length === 0) return '请至少添加一个 AI 配置';
    const ids = new Set<string>();
    for (const provider of providers) {
      const id = provider.id.trim();
      if (!id) return '请填写 AI 配置 ID';
      if (ids.has(id)) return `AI 配置 ID「${id}」重复`;
      ids.add(id);
      if (!provider.base_url.trim()) return `请填写「${id}」的 Base URL`;
      if (!provider.model.trim()) return `请填写「${id}」的模型名称`;
    }
    if (!ids.has(active)) return '请选择默认 AI 配置';
    return '';
  };

  const handleSave = async () => {
    const error = validateConfig();
    if (error) {
      toast.error(error);
      return;
    }
    setSaving(true);
    try {
      const res: any = await saveAIConfig({
        active,
        providers: providers.map(provider => ({
          ...provider,
          id: provider.id.trim(),
          label: provider.label.trim(),
          type: provider.type || 'openai-compatible',
          base_url: provider.base_url.trim(),
          api_key: provider.api_key.trim(),
          model: provider.model.trim(),
          max_tokens: Number(provider.max_tokens) || 4096,
        })),
      });
      if (res.code === 0) {
        toast.success('AI 配置已保存');
        await loadConfig();
      } else {
        toast.error(res.msg || '保存失败');
      }
    } catch {
      toast.error('保存 AI 配置异常');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="w-full h-full flex flex-col gap-6">
      <ConfirmDialog {...dialogProps} />
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-gray-900">AI 配置</h1>
          <p className="text-gray-500 mt-1">管理右上角 AI 下拉框可切换的模型厂商与默认选项。</p>
        </div>
        <div className="flex items-center gap-3 w-full sm:w-auto">
          <button
            type="button"
            onClick={handleAddProvider}
            className="flex-1 sm:flex-none border border-gray-200 bg-white hover:bg-gray-50 text-gray-800 font-medium py-2 px-4 rounded-lg shadow-sm transition-colors flex items-center justify-center gap-2"
          >
            <Plus size={16} /> 添加 AI
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving || loading}
            className="flex-1 sm:flex-none bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium py-2 px-4 rounded-lg shadow-sm transition-colors flex items-center justify-center gap-2"
          >
            <Save size={16} /> {saving ? '保存中...' : '保存配置'}
          </button>
        </div>
      </div>

      {loading ? (
        <div className="h-40 flex items-center justify-center text-gray-400">AI 配置加载中...</div>
      ) : providers.length === 0 ? (
        <div className="border border-dashed border-gray-300 rounded-xl p-12 text-center bg-gray-50">
          <h3 className="text-lg font-medium text-gray-900 mb-2">暂无 AI 配置</h3>
          <p className="text-gray-500 mb-6">添加一个模型配置后，右上角 AI 下拉框就可以选择它。</p>
          <button onClick={handleAddProvider} className="bg-black hover:bg-gray-800 text-white font-medium py-2 px-6 rounded-lg transition-colors">
            添加 AI
          </button>
        </div>
      ) : (
        <div className="grid min-h-0 flex-1 gap-5 lg:grid-cols-[280px_minmax(0,1fr)]">
          <div className="min-h-0 space-y-3 overflow-y-auto rounded-xl border border-gray-200 bg-gray-50 p-3">
            {providers.map(provider => (
              <button
                key={provider.id}
                type="button"
                onClick={() => setSelectedId(provider.id)}
                className={`w-full rounded-lg border p-4 text-left transition ${
                  selectedProvider?.id === provider.id
                    ? 'border-indigo-400 bg-white shadow-sm'
                    : 'border-gray-200 bg-white hover:border-gray-300'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <Bot size={16} className={active === provider.id ? 'text-indigo-600' : 'text-gray-400'} />
                      <span className="truncate text-sm font-semibold text-gray-900">{provider.label || provider.id || '未命名 AI'}</span>
                    </div>
                    <p className="mt-1 truncate text-xs text-gray-500">{provider.model || '未设置模型'}</p>
                  </div>
                  {active === provider.id && <CheckCircle2 size={17} className="shrink-0 text-indigo-600" />}
                </div>
              </button>
            ))}
          </div>

          <div className="min-h-0 overflow-y-auto rounded-xl border border-gray-200 bg-white p-5">
            {selectedProvider && (
              <div className="space-y-5">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-lg font-bold text-gray-900">{selectedProvider.label || selectedProvider.id || '未命名 AI'}</h2>
                    <p className="text-sm text-gray-500 mt-1">保存后会同步写入后端配置文件。</p>
                  </div>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => handleSetActive(selectedProvider.id)}
                      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-indigo-100 bg-indigo-50 px-3 text-xs font-medium text-indigo-700 hover:bg-indigo-100"
                    >
                      <CheckCircle2 size={14} /> 设为默认
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDeleteProvider(selectedProvider.id)}
                      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-100 text-red-500 hover:bg-red-50"
                      title="删除 AI 配置"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                  <label className="block">
                    <span className="mb-1 flex items-center gap-1 text-sm font-medium text-gray-700"><Hash size={14} /> 配置 ID</span>
                    <input
                      value={selectedProvider.id}
                      onChange={(e) => updateProvider(selectedProvider.id, { id: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="omlx"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-1 block text-sm font-medium text-gray-700">显示名称</span>
                    <input
                      value={selectedProvider.label}
                      onChange={(e) => updateProvider(selectedProvider.id, { label: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="oMLX 本地"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-1 block text-sm font-medium text-gray-700">接口类型</span>
                    <select
                      value={selectedProvider.type || 'openai-compatible'}
                      onChange={(e) => updateProvider(selectedProvider.id, { type: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-indigo-100"
                    >
                      <option value="openai-compatible">openai-compatible</option>
                      <option value="anthropic-compatible">anthropic-compatible</option>
                    </select>
                  </label>
                  <label className="block">
                    <span className="mb-1 flex items-center gap-1 text-sm font-medium text-gray-700"><Bot size={14} /> 模型名称</span>
                    <input
                      value={selectedProvider.model}
                      onChange={(e) => updateProvider(selectedProvider.id, { model: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="qwen3-coder-plus"
                    />
                  </label>
                  <label className="block lg:col-span-2">
                    <span className="mb-1 flex items-center gap-1 text-sm font-medium text-gray-700"><Link size={14} /> Base URL</span>
                    <input
                      value={selectedProvider.base_url}
                      onChange={(e) => updateProvider(selectedProvider.id, { base_url: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="https://api.example.com"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-1 flex items-center gap-1 text-sm font-medium text-gray-700"><KeyRound size={14} /> API Key</span>
                    <input
                      type="password"
                      value={selectedProvider.api_key}
                      onChange={(e) => updateProvider(selectedProvider.id, { api_key: e.target.value })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="sk-..."
                      autoComplete="off"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-1 block text-sm font-medium text-gray-700">最大 Tokens</span>
                    <input
                      type="number"
                      min={1}
                      value={selectedProvider.max_tokens || 4096}
                      onChange={(e) => updateProvider(selectedProvider.id, { max_tokens: Number(e.target.value) })}
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono outline-none focus:ring-2 focus:ring-indigo-100"
                      placeholder="4096"
                    />
                  </label>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
