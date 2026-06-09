import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { ArrowLeft, Braces, Database, Edit2, Plus, RefreshCw, Save, Trash2, X } from 'lucide-react';
import toast from 'react-hot-toast';
import clsx from 'clsx';
import {
  createDbTemplateScript,
  createDbTemplateType,
  deleteDbTemplateScript,
  deleteDbTemplateType,
  getDbTemplateScripts,
  getDbTemplateTypes,
  updateDbTemplateScript,
  updateDbTemplateType,
} from '@/api/db_template';
import { getProjectList } from '@/api/code_generate_project';
import { parseDbTemplatePlaceholders, stringifyDbTemplatePlaceholders, type DbTemplatePlaceholder } from './dbTemplateCopy';

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

const emptyType = (projectId: number) => ({
  projectId,
  typeName: '',
  prompt: '',
  dynamicPlaceholders: '',
  sort: 0,
});

const emptyScript = (projectId: number, typeId: number) => ({
  projectId,
  typeId,
  scriptName: '',
  scriptKind: 'sql',
  content: '',
  sort: 0,
});

const scriptDraftForType = (projectId: number, typeObj: any, script?: any) => ({
  ...emptyScript(projectId, Number(typeObj?.ID || 0)),
  scriptName: typeObj?.typeName || '',
  ...(script || {}),
});

type DbTemplateLibraryProps = {
  projectIdOverride?: number | string;
  onClose?: () => void;
  fullscreenDialog?: boolean;
};

export default function DbTemplateLibrary({ projectIdOverride, onClose, fullscreenDialog = false }: DbTemplateLibraryProps = {}) {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const numericProjectId = Number(projectIdOverride || projectId || 0);

  const [projectName, setProjectName] = useState('解析中...');
  const [types, setTypes] = useState<any[]>([]);
  const [activeType, setActiveType] = useState<any>(null);
  const [typeDraft, setTypeDraft] = useState<any>(emptyType(numericProjectId));
  const [showTypeModal, setShowTypeModal] = useState(false);
  const [placeholderType, setPlaceholderType] = useState<any>(null);
  const [placeholderDraft, setPlaceholderDraft] = useState<DbTemplatePlaceholder[]>([]);
  const [showPlaceholderModal, setShowPlaceholderModal] = useState(false);
  const [loadingTypes, setLoadingTypes] = useState(false);

  const [scriptDraft, setScriptDraft] = useState<any>(null);
  const [loadingScripts, setLoadingScripts] = useState(false);
  const [savingScript, setSavingScript] = useState(false);

  const loadScripts = async (typeObj: any, preferredScriptId?: number) => {
    if (!typeObj?.ID || !numericProjectId) {
      setScriptDraft(null);
      return;
    }

    setLoadingScripts(true);
    try {
      const res: any = await getDbTemplateScripts(numericProjectId, Number(typeObj.ID));
      const list = Array.isArray(unwrapResponseData(res)) ? unwrapResponseData(res) : [];
      const nextScript = list.find((item: any) => item.ID === preferredScriptId) || list[0];
      setScriptDraft(scriptDraftForType(numericProjectId, typeObj, nextScript));
    } catch (e) {
      toast.error('加载数据库脚本失败');
    } finally {
      setLoadingScripts(false);
    }
  };

  const loadTypes = async (preferredTypeId?: number) => {
    if (!numericProjectId) return;
    setLoadingTypes(true);
    try {
      const res: any = await getDbTemplateTypes(numericProjectId);
      const list = Array.isArray(unwrapResponseData(res)) ? unwrapResponseData(res) : [];
      setTypes(list);
      const nextType = list.find((item: any) => item.ID === preferredTypeId) ||
        list.find((item: any) => item.ID === activeType?.ID) ||
        list[0] ||
        null;
      setActiveType(nextType);
      await loadScripts(nextType);
    } catch (e) {
      toast.error('加载业务类型失败');
    } finally {
      setLoadingTypes(false);
    }
  };

  const loadProjectName = async () => {
    if (!numericProjectId) return;
    try {
      const res: any = await getProjectList();
      const list = Array.isArray(unwrapResponseData(res)) ? unwrapResponseData(res) : [];
      const found = list.find((item: any) => Number(item.ID) === numericProjectId);
      if (found?.projectName) setProjectName(found.projectName);
    } catch (e) {
      setProjectName(`Project ${numericProjectId}`);
    }
  };

  useEffect(() => {
    if (numericProjectId) {
      loadProjectName();
      loadTypes();
    }
  }, [numericProjectId]);

  const openCreateType = () => {
    setTypeDraft(emptyType(numericProjectId));
    setShowTypeModal(true);
  };

  const openEditType = (typeObj: any) => {
    setTypeDraft(typeObj);
    setShowTypeModal(true);
  };

  const openPlaceholderEditor = (typeObj: any) => {
    const list = parseDbTemplatePlaceholders(typeObj?.dynamicPlaceholders);
    setPlaceholderType(typeObj);
    setPlaceholderDraft(list.length > 0 ? list : [{ key: '', description: '', value: '' }]);
    setShowPlaceholderModal(true);
  };

  const updatePlaceholderRow = (index: number, patch: Partial<DbTemplatePlaceholder>) => {
    setPlaceholderDraft((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, ...patch } : row));
  };

  const addPlaceholderRow = () => {
    setPlaceholderDraft((rows) => [...rows, { key: '', description: '', value: '' }]);
  };

  const removePlaceholderRow = (index: number) => {
    setPlaceholderDraft((rows) => rows.filter((_, rowIndex) => rowIndex !== index));
  };

  const handleSavePlaceholders = async () => {
    if (!placeholderType?.ID) return;
    const keys = new Set<string>();
    const normalized = placeholderDraft
      .map((row) => ({
        key: String(row.key || '').trim(),
        description: String(row.description || '').trim(),
        value: String(row.value || '').trim(),
      }))
      .filter((row) => row.key);

    for (const row of normalized) {
      if (keys.has(row.key)) {
        toast.error(`占位符 key 重复：${row.key}`);
        return;
      }
      keys.add(row.key);
    }

    try {
      const payload = {
        ...placeholderType,
        dynamicPlaceholders: stringifyDbTemplatePlaceholders(normalized),
      };
      const res: any = await updateDbTemplateType(payload);
      const saved = unwrapResponseData(res);
      toast.success('占位符已保存');
      setShowPlaceholderModal(false);
      setPlaceholderType(null);
      await loadTypes(saved?.ID || placeholderType.ID);
    } catch (e) {
      toast.error('保存占位符失败');
    }
  };

  const handleSaveType = async () => {
    if (!typeDraft.typeName?.trim()) {
      toast.error('业务类型名称不能为空');
      return;
    }
    try {
      const payload = {
        ...typeDraft,
        projectId: numericProjectId,
        typeName: typeDraft.typeName.trim(),
        prompt: String(typeDraft.prompt || '').trim(),
        dynamicPlaceholders: stringifyDbTemplatePlaceholders(typeDraft.dynamicPlaceholders),
      };
      const res: any = payload.ID ? await updateDbTemplateType(payload) : await createDbTemplateType(payload);
      const saved = unwrapResponseData(res);
      toast.success(payload.ID ? '业务类型已更新' : '业务类型已创建');
      setShowTypeModal(false);
      await loadTypes(saved?.ID || payload.ID);
    } catch (e) {
      toast.error('业务类型保存失败');
    }
  };

  const handleDeleteType = async (typeObj: any) => {
    if (!confirm(`确定删除业务类型「${typeObj.typeName}」及其全部脚本吗？`)) return;
    try {
      await deleteDbTemplateType(typeObj);
      toast.success('业务类型已删除');
      setActiveType(null);
      await loadTypes();
    } catch (e) {
      toast.error('删除业务类型失败');
    }
  };

  const handleSelectType = async (typeObj: any) => {
    setActiveType(typeObj);
    await loadScripts(typeObj);
  };

  const handleSaveScript = async () => {
    if (!activeType?.ID || !scriptDraft) return;
    const scriptName = scriptDraft.scriptName?.trim() || activeType.typeName?.trim() || '数据库脚本';
    setSavingScript(true);
    try {
      const payload = {
        ...scriptDraft,
        projectId: numericProjectId,
        typeId: Number(activeType.ID),
        scriptName,
        scriptKind: scriptDraft.scriptKind || 'sql',
      };
      const res: any = payload.ID ? await updateDbTemplateScript(payload) : await createDbTemplateScript(payload);
      const saved = unwrapResponseData(res);
      toast.success(payload.ID ? '脚本已更新' : '脚本已创建');
      await loadScripts(activeType, saved?.ID || payload.ID);
    } catch (e) {
      toast.error('脚本保存失败');
    } finally {
      setSavingScript(false);
    }
  };

  const handleDeleteScript = async () => {
    if (!scriptDraft?.ID) return;
    if (!confirm(`确定删除脚本「${scriptDraft.scriptName}」吗？`)) return;
    try {
      await deleteDbTemplateScript(scriptDraft);
      toast.success('脚本已删除');
      await loadScripts(activeType);
    } catch (e) {
      toast.error('删除脚本失败');
    }
  };

  return (
    <div className={clsx('flex flex-col overflow-hidden bg-white', fullscreenDialog ? 'h-full' : 'h-screen')}>
      <div className="z-20 flex items-center justify-between bg-gradient-to-r from-slate-800 to-slate-900 px-6 py-4 text-white shadow-md">
        <div className="flex min-w-0 items-center gap-4">
          {!fullscreenDialog && (
            <>
              <button
                onClick={() => navigate('/code-generate')}
                className="flex items-center gap-1 rounded-xl border border-slate-700 p-2 text-sm font-medium transition-colors hover:bg-white/10"
              >
                <ArrowLeft size={16} /> 返回主版
              </button>
              <div className="h-6 w-px bg-slate-700" />
            </>
          )}
          <div className="min-w-0">
            <h1 className="truncate text-lg font-bold tracking-tight text-cyan-300">
              数据库模板示例库 <span className="font-normal text-slate-300">/ {projectName}</span>
            </h1>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            onClick={handleSaveScript}
            disabled={!activeType || !scriptDraft || loadingScripts || savingScript}
            className="inline-flex items-center gap-2 rounded-xl bg-cyan-400 px-4 py-2 text-sm font-bold text-slate-950 shadow-[0_0_16px_rgba(34,211,238,0.25)] transition-all hover:bg-cyan-300 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {savingScript ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
            保存脚本
          </button>
          {fullscreenDialog && (
            <button
              type="button"
              onClick={onClose}
              className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-slate-700 text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
              title="关闭"
            >
              <X size={18} />
            </button>
          )}
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        <aside className="flex w-80 flex-shrink-0 flex-col border-r border-slate-200 bg-slate-50">
          <div className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3">
            <div className="flex items-center gap-2 text-sm font-bold text-slate-700">
              <Database size={16} className="text-cyan-600" />
              业务类型
            </div>
            <button
              onClick={openCreateType}
              className="inline-flex items-center gap-1 rounded-lg bg-slate-900 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-slate-700"
            >
              <Plus size={13} /> 新增
            </button>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto p-3">
            {loadingTypes ? (
              <div className="mt-10 text-center text-sm text-slate-400">加载中...</div>
            ) : types.length === 0 ? (
              <div className="mt-10 text-center text-sm text-slate-400">暂无业务类型</div>
            ) : (
              <div className="flex flex-col gap-1">
                {types.map((typeObj) => {
                  const active = activeType?.ID === typeObj.ID;
                  return (
                    <div
                      key={typeObj.ID}
                      className={clsx(
                        'group flex cursor-pointer items-center gap-2 rounded-xl border px-3 py-2 transition-colors',
                        active ? 'border-cyan-200 bg-cyan-50 text-cyan-900' : 'border-transparent text-slate-600 hover:bg-white hover:text-slate-900'
                      )}
                      onClick={() => handleSelectType(typeObj)}
                    >
                      <Database size={15} className={active ? 'text-cyan-600' : 'text-slate-400'} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-semibold" title={typeObj.typeName}>{typeObj.typeName}</div>
                      </div>
                      <button
                        onClick={(event) => {
                          event.stopPropagation();
                          openEditType(typeObj);
                        }}
                        className="rounded-md p-1 text-slate-300 opacity-0 transition hover:bg-cyan-100 hover:text-cyan-700 group-hover:opacity-100"
                        title="编辑"
                      >
                        <Edit2 size={14} />
                      </button>
                      <button
                        onClick={(event) => {
                          event.stopPropagation();
                          openPlaceholderEditor(typeObj);
                        }}
                        className="rounded-md p-1 text-slate-300 opacity-0 transition hover:bg-emerald-100 hover:text-emerald-700 group-hover:opacity-100"
                        title="动态占位符"
                      >
                        <Braces size={14} />
                      </button>
                      <button
                        onClick={(event) => {
                          event.stopPropagation();
                          handleDeleteType(typeObj);
                        }}
                        className="rounded-md p-1 text-slate-300 opacity-0 transition hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col bg-slate-950">
          {activeType ? (
            <>
              <div className="flex flex-wrap items-end gap-4 border-b border-slate-800 bg-slate-900 px-5 py-3">
                <div className="min-w-[240px] flex-1">
                  <div className="mb-1 flex items-center gap-2 text-xs font-semibold text-cyan-300">
                    <Database size={14} />
                    数据库脚本
                  </div>
                  <h2 className="truncate text-lg font-bold text-slate-100" title={activeType.typeName}>
                    {activeType.typeName}
                  </h2>
                </div>
                <button
                  onClick={handleDeleteScript}
                  disabled={loadingScripts || !scriptDraft?.ID}
                  className="inline-flex items-center gap-2 rounded-xl border border-red-500/30 px-3 py-2 text-sm font-semibold text-red-300 transition hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <Trash2 size={15} /> 删除脚本
                </button>
              </div>

              <div className="min-h-0 flex-1">
                {loadingScripts || !scriptDraft ? (
                  <div className="flex h-full items-center justify-center text-sm text-slate-500">加载脚本...</div>
                ) : (
                  <Editor
                    height="100%"
                    theme="vs-dark"
                    language="sql"
                    value={scriptDraft.content || ''}
                    onChange={(value) => setScriptDraft({ ...scriptDraft, content: value || '' })}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 15,
                      fontFamily: '"Fira Code", Monaco, "Courier New", monospace',
                      wordWrap: 'on',
                      padding: { top: 24, bottom: 24 },
                      smoothScrolling: true,
                      lineHeight: 1.6,
                    }}
                  />
                )}
              </div>
            </>
          ) : (
            <div className="flex flex-1 flex-col items-center justify-center bg-slate-950 text-slate-500">
              <div className="mb-6 rounded-full bg-slate-900 p-8">
                <Database size={64} className="text-cyan-500 opacity-20" />
              </div>
              <h2 className="mb-2 text-2xl font-bold text-slate-400">数据库模板工作区</h2>
              <p className="text-sm text-slate-500">选择业务类型后维护 SQL 脚本</p>
            </div>
          )}
        </main>
      </div>

      {showTypeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
          <div className="max-h-[92vh] w-full max-w-5xl rounded-3xl bg-white p-9 shadow-2xl">
            <h2 className="mb-5 text-xl font-bold text-slate-800">{typeDraft.ID ? '编辑业务类型' : '新增业务类型'}</h2>
            <div className="space-y-5">
              <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_220px]">
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-slate-700">业务类型名称</label>
                  <input
                    value={typeDraft.typeName || ''}
                    onChange={(event) => setTypeDraft({ ...typeDraft, typeName: event.target.value })}
                    className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-base font-semibold outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                    placeholder="建表 SQL / 字典 SQL / 菜单按钮 SQL"
                    autoFocus
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-slate-700">排序</label>
                  <input
                    type="number"
                    value={typeDraft.sort ?? 0}
                    onChange={(event) => setTypeDraft({ ...typeDraft, sort: Number(event.target.value) })}
                    className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-base outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                  />
                </div>
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-slate-700">提示词</label>
                <textarea
                  value={typeDraft.prompt || ''}
                  onChange={(event) => setTypeDraft({ ...typeDraft, prompt: event.target.value })}
                  className="min-h-[380px] w-full resize-y rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm leading-6 outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                  placeholder="输入该业务类型的生成说明、约束或示例要求..."
                />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button
                onClick={() => setShowTypeModal(false)}
                className="rounded-xl px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
              >
                取消
              </button>
              <button
                onClick={handleSaveType}
                className="rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700"
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {showPlaceholderModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm">
          <div className="flex max-h-[92vh] w-full max-w-5xl flex-col rounded-3xl bg-white shadow-2xl">
            <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-8 py-6">
              <div className="min-w-0">
                <h2 className="text-xl font-bold text-slate-800">动态占位符</h2>
                <p className="mt-1 truncate text-sm font-medium text-slate-500">
                  {placeholderType?.typeName || '业务类型'}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setShowPlaceholderModal(false)}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
              <div className="overflow-hidden rounded-2xl border border-slate-200">
                <div className="grid grid-cols-[minmax(140px,0.9fr)_minmax(180px,1.2fr)_minmax(180px,1.2fr)_48px] border-b border-slate-200 bg-slate-50 text-sm font-bold text-slate-600">
                  <div className="px-4 py-3">占位符 key</div>
                  <div className="px-4 py-3">描述</div>
                  <div className="px-4 py-3">默认 value</div>
                  <div />
                </div>
                {placeholderDraft.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm font-medium text-slate-400">暂无占位符</div>
                ) : (
                  placeholderDraft.map((row, index) => (
                    <div
                      key={index}
                      className="grid grid-cols-[minmax(140px,0.9fr)_minmax(180px,1.2fr)_minmax(180px,1.2fr)_48px] items-center border-b border-slate-100 last:border-b-0"
                    >
                      <div className="p-2">
                        <input
                          value={row.key || ''}
                          onChange={(event) => updatePlaceholderRow(index, { key: event.target.value })}
                          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 font-mono text-sm outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                          placeholder="companyId"
                        />
                      </div>
                      <div className="p-2">
                        <input
                          value={row.description || ''}
                          onChange={(event) => updatePlaceholderRow(index, { description: event.target.value })}
                          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                          placeholder="公司 ID"
                        />
                      </div>
                      <div className="p-2">
                        <input
                          value={row.value || ''}
                          onChange={(event) => updatePlaceholderRow(index, { value: event.target.value })}
                          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 font-mono text-sm outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
                          placeholder="-1"
                        />
                      </div>
                      <div className="flex justify-center p-2">
                        <button
                          type="button"
                          onClick={() => removePlaceholderRow(index)}
                          className="flex h-9 w-9 items-center justify-center rounded-xl text-slate-400 transition hover:bg-red-50 hover:text-red-600"
                          title="删除"
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
              <button
                type="button"
                onClick={addPlaceholderRow}
                className="mt-4 inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-2 text-sm font-bold text-slate-700 transition hover:bg-slate-50"
              >
                <Plus size={15} />
                新增占位符
              </button>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 px-8 py-5">
              <button
                onClick={() => setShowPlaceholderModal(false)}
                className="rounded-xl px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
              >
                取消
              </button>
              <button
                onClick={handleSavePlaceholders}
                className="rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700"
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
