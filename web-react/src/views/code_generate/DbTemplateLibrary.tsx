import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { ArrowLeft, Database, Edit2, Plus, RefreshCw, Save, Trash2 } from 'lucide-react';
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

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

const emptyType = (projectId: number) => ({
  projectId,
  typeName: '',
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

export default function DbTemplateLibrary() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const numericProjectId = Number(projectId || 0);

  const [projectName, setProjectName] = useState('解析中...');
  const [types, setTypes] = useState<any[]>([]);
  const [activeType, setActiveType] = useState<any>(null);
  const [typeDraft, setTypeDraft] = useState<any>(emptyType(numericProjectId));
  const [showTypeModal, setShowTypeModal] = useState(false);
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

  const handleSaveType = async () => {
    if (!typeDraft.typeName?.trim()) {
      toast.error('业务类型名称不能为空');
      return;
    }
    try {
      const payload = { ...typeDraft, projectId: numericProjectId, typeName: typeDraft.typeName.trim() };
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
    <div className="flex h-screen flex-col overflow-hidden bg-white">
      <div className="z-20 flex items-center justify-between bg-gradient-to-r from-slate-800 to-slate-900 px-6 py-4 text-white shadow-md">
        <div className="flex min-w-0 items-center gap-4">
          <button
            onClick={() => navigate('/code-generate')}
            className="flex items-center gap-1 rounded-xl border border-slate-700 p-2 text-sm font-medium transition-colors hover:bg-white/10"
          >
            <ArrowLeft size={16} /> 返回主版
          </button>
          <div className="h-6 w-px bg-slate-700" />
          <div className="min-w-0">
            <h1 className="truncate text-lg font-bold tracking-tight text-cyan-300">
              数据库模板示例库 <span className="font-normal text-slate-300">/ {projectName}</span>
            </h1>
          </div>
        </div>
        <button
          onClick={handleSaveScript}
          disabled={!activeType || !scriptDraft || loadingScripts || savingScript}
          className="inline-flex items-center gap-2 rounded-xl bg-cyan-400 px-4 py-2 text-sm font-bold text-slate-950 shadow-[0_0_16px_rgba(34,211,238,0.25)] transition-all hover:bg-cyan-300 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {savingScript ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
          保存脚本
        </button>
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
                      >
                        <Edit2 size={14} />
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
          <div className="w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl">
            <h2 className="mb-5 text-xl font-bold text-slate-800">{typeDraft.ID ? '编辑业务类型' : '新增业务类型'}</h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-slate-700">业务类型名称</label>
                <input
                  value={typeDraft.typeName || ''}
                  onChange={(event) => setTypeDraft({ ...typeDraft, typeName: event.target.value })}
                  className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
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
                  className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm outline-none transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-500/20"
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
    </div>
  );
}
