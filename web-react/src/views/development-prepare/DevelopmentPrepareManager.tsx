import { useCallback, useEffect, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import clsx from 'clsx';
import toast from 'react-hot-toast';
import {
  ChevronRight,
  ClipboardCheck,
  Code2,
  Copy,
  Database,
  Folder,
  ListChecks,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Star,
  Trash2,
  X,
  type LucideIcon,
} from 'lucide-react';
import {
  DevelopmentPrepareItem,
  DevelopmentPrepareType,
  deleteDevelopmentPrepare,
  getDevelopmentPreparePage,
  saveOrUpdateDevelopmentPrepare,
} from '../../api/developmentPrepare';
import { useProjectStore } from '../../stores/useProjectStore';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

type DraftItem = Partial<DevelopmentPrepareItem> & {
  title: string;
  itemType: DevelopmentPrepareType;
  content: string;
};

const typeOptions: Array<{
  key: DevelopmentPrepareType;
  label: string;
  icon: LucideIcon;
  tone: string;
}> = [
  { key: 'script', label: 'SQL脚本', icon: Database, tone: 'text-emerald-600 bg-emerald-50 border-emerald-100' },
  { key: 'code', label: '代码', icon: Code2, tone: 'text-blue-600 bg-blue-50 border-blue-100' },
  { key: 'checklist', label: '清单', icon: ListChecks, tone: 'text-fuchsia-600 bg-fuchsia-50 border-fuchsia-100' },
];

const defaultLanguageByType: Record<DevelopmentPrepareType, string> = {
  script: 'sql',
  code: 'typescript',
  checklist: 'markdown',
};

const defaultTitleByType: Record<DevelopmentPrepareType, string> = {
  script: 'SQL脚本',
  code: '准备代码',
  checklist: '准备清单',
};

const ungroupedBusinessLabel = '未分组';

function getTypeOption(type?: string) {
  return typeOptions.find(option => option.key === type) || typeOptions[0];
}

function getBusinessGroupLabel(value?: string) {
  return value?.trim() || ungroupedBusinessLabel;
}

function formatDateTime(value?: string) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function getContentStats(content?: string) {
  const text = content || '';
  return {
    lines: text ? text.split(/\r\n|\r|\n/).length : 0,
    chars: text.length,
  };
}

export default function DevelopmentPrepareManager() {
  const { activeProject, activeProjectId } = useProjectStore();
  const [items, setItems] = useState<DevelopmentPrepareItem[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<DraftItem | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [collapsedBusinessGroups, setCollapsedBusinessGroups] = useState<Set<string>>(() => new Set());
  const [quickContent, setQuickContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [quickSaving, setQuickSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const { confirm, dialogProps } = useConfirm();

  const hasProjectContext = Boolean(activeProjectId || activeProject);
  const activeProjectLabel = activeProject || (activeProjectId ? `项目 #${activeProjectId}` : '未选择项目');

  const createDraft = useCallback((type?: DevelopmentPrepareType, businessGroup = ''): DraftItem => {
    const itemType = type || 'script';
    return {
      projectConfigId: activeProjectId || 0,
      projectConfigName: activeProject || activeProjectLabel,
      businessGroup,
      title: defaultTitleByType[itemType],
      itemType,
      language: defaultLanguageByType[itemType],
      tags: '',
      summary: '',
      content: '',
      isPinned: false,
      sort: 0,
    };
  }, [activeProject, activeProjectId, activeProjectLabel]);

  const cloneForDraft = useCallback((item: DevelopmentPrepareItem): DraftItem => ({
    ...item,
    title: item.title || defaultTitleByType[item.itemType || 'script'],
    itemType: item.itemType || 'script',
    content: item.content || '',
    businessGroup: item.businessGroup || '',
    projectConfigId: item.projectConfigId || activeProjectId || 0,
    projectConfigName: item.projectConfigName || activeProject || activeProjectLabel,
  }), [activeProject, activeProjectId, activeProjectLabel]);

  const loadItems = useCallback(async () => {
    if (!hasProjectContext) {
      setItems([]);
      return;
    }

    setLoading(true);
    try {
      const res: any = await getDevelopmentPreparePage({
        page: 1,
        pageSize: 100,
        ...(activeProjectId ? { projectConfigId: activeProjectId } : { projectConfigName: activeProject }),
      });
      const nextItems: DevelopmentPrepareItem[] = res.data?.list || [];
      setItems(nextItems);
      setSelectedId(previous => {
        if (previous && nextItems.some(item => item.ID === previous)) return previous;
        return nextItems[0]?.ID || null;
      });
    } catch {
      toast.error('开发准备加载失败');
    } finally {
      setLoading(false);
    }
  }, [activeProject, activeProjectId, hasProjectContext]);

  useEffect(() => {
    setCollapsedBusinessGroups(new Set());
    void loadItems();
  }, [loadItems]);

  const filteredItems = useMemo(() => {
    return items;
  }, [items]);

  const groupedItems = useMemo(() => {
    const groupMap = new Map<string, DevelopmentPrepareItem[]>();
    filteredItems.forEach(item => {
      const group = getBusinessGroupLabel(item.businessGroup);
      groupMap.set(group, [...(groupMap.get(group) || []), item]);
    });
    return Array.from(groupMap.entries())
      .sort(([left], [right]) => {
        if (left === ungroupedBusinessLabel) return 1;
        if (right === ungroupedBusinessLabel) return -1;
        return left.localeCompare(right, 'zh-CN');
      })
      .map(([group, groupItems]) => ({ group, items: groupItems }));
  }, [filteredItems]);

  const businessOptions = useMemo(() => {
    return Array.from(new Set(items.map(item => item.businessGroup?.trim()).filter(Boolean) as string[]))
      .sort((left, right) => left.localeCompare(right, 'zh-CN'));
  }, [items]);

  const patchDraft = <K extends keyof DraftItem>(key: K, value: DraftItem[K]) => {
    setDraft(previous => previous ? { ...previous, [key]: value } : previous);
  };

  const changeDraftType = (type: DevelopmentPrepareType) => {
    setDraft(previous => {
      if (!previous) return previous;
      const shouldUpdateLanguage = !previous.language || previous.language === defaultLanguageByType[previous.itemType];
      return {
        ...previous,
        itemType: type,
        language: shouldUpdateLanguage ? defaultLanguageByType[type] : previous.language,
      };
    });
  };

  const openCreate = (type?: DevelopmentPrepareType) => {
    if (!hasProjectContext) {
      toast.error('请先选择项目');
      return;
    }
    const currentItem = items.find(item => item.ID === selectedId);
    setDraft(createDraft(type, currentItem?.businessGroup?.trim() || ''));
    setSelectedId(null);
    setModalOpen(true);
  };

  const openEdit = (item: DevelopmentPrepareItem) => {
    setDraft(cloneForDraft(item));
    setSelectedId(item.ID);
    setModalOpen(true);
  };

  const closeModal = () => {
    if (saving) return;
    setModalOpen(false);
    setDraft(null);
  };

  const handleSave = async () => {
    if (!draft) return;
    if (!hasProjectContext) {
      toast.error('请先选择项目');
      return;
    }
    const title = draft.title.trim();
    const businessGroup = draft.businessGroup?.trim() || '';
    if (!title) {
      toast.error('标题不能为空');
      return;
    }

    setSaving(true);
    try {
      const res: any = await saveOrUpdateDevelopmentPrepare({
        ...draft,
        title,
        businessGroup,
        projectConfigId: activeProjectId || 0,
        projectConfigName: activeProject || draft.projectConfigName || activeProjectLabel,
      });
      if (res.code !== 0) {
        toast.error(res.msg || '保存失败');
        return;
      }
      toast.success('开发准备已保存');
      setModalOpen(false);
      setDraft(null);
      await loadItems();
    } catch {
      toast.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item: DevelopmentPrepareItem) => {
    const ok = await confirm(`确定删除「${item.title || '未命名'}」吗？`, {
      title: '删除开发准备',
      confirmText: '确定删除',
      cancelText: '取消',
    });
    if (!ok) return;

    setDeletingId(item.ID);
    try {
      const res: any = await deleteDevelopmentPrepare(item.ID);
      if (res.code !== 0) {
        toast.error(res.msg || '删除失败');
        return;
      }
      toast.success('已删除');
      await loadItems();
    } catch {
      toast.error('删除失败');
    } finally {
      setDeletingId(null);
    }
  };

  const handleCopy = async (item: DevelopmentPrepareItem) => {
    const content = item.ID === selectedItem?.ID ? quickContent : item.content;
    if (!content) {
      toast.error('暂无内容可复制');
      return;
    }
    try {
      await navigator.clipboard.writeText(content);
      toast.success('内容已复制');
    } catch {
      toast.error('复制失败');
    }
  };

  const draftLanguage = draft?.language?.trim() || defaultLanguageByType[draft?.itemType || 'script'];
  const selectedItem = useMemo(
    () => filteredItems.find(item => item.ID === selectedId) || filteredItems[0] || null,
    [filteredItems, selectedId]
  );

  useEffect(() => {
    if (!selectedItem) return;
    setQuickContent(selectedItem.content || '');
    const selectedGroup = getBusinessGroupLabel(selectedItem.businessGroup);
    setCollapsedBusinessGroups(previous => {
      if (!previous.has(selectedGroup)) return previous;
      const next = new Set(previous);
      next.delete(selectedGroup);
      return next;
    });
  }, [selectedItem]);

  const handleQuickSave = async () => {
    if (!selectedItem || quickSaving) return;
    setQuickSaving(true);
    try {
      const res: any = await saveOrUpdateDevelopmentPrepare({
        ...selectedItem,
        content: quickContent,
        projectConfigId: activeProjectId || selectedItem.projectConfigId || 0,
        projectConfigName: activeProject || selectedItem.projectConfigName || activeProjectLabel,
      });
      if (res.code !== 0) {
        toast.error(res.msg || '保存失败');
        return;
      }
      toast.success('内容已保存');
      await loadItems();
    } catch {
      toast.error('保存失败');
    } finally {
      setQuickSaving(false);
    }
  };

  const toggleBusinessGroup = (group: string) => {
    setCollapsedBusinessGroups(previous => {
      const next = new Set(previous);
      if (next.has(group)) {
        next.delete(group);
      } else {
        next.add(group);
      }
      return next;
    });
  };

  return (
    <div className="min-h-[calc(100vh-64px)] bg-gray-50 text-gray-900">
      <main className="grid min-h-[calc(100vh-64px)] grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="border-r border-gray-200 bg-white p-4">
          <div className="space-y-3">
            {!hasProjectContext ? (
              <div className="rounded-xl border border-dashed border-gray-300 bg-gray-50 px-4 py-12 text-center">
                <ClipboardCheck size={28} className="mx-auto mb-3 text-gray-300" />
                <div className="text-sm font-bold text-gray-700">请选择项目</div>
              </div>
            ) : loading ? (
              <div className="flex items-center justify-center py-12 text-sm text-gray-400">
                <RefreshCw size={16} className="mr-2 animate-spin" />
                加载中...
              </div>
            ) : filteredItems.length === 0 ? (
              <div className="rounded-xl border border-dashed border-gray-300 bg-gray-50 px-4 py-12 text-center">
                <ClipboardCheck size={28} className="mx-auto mb-3 text-gray-300" />
                <div className="text-sm font-bold text-gray-700">暂无准备项</div>
              </div>
            ) : (
              groupedItems.map(group => {
                const collapsed = collapsedBusinessGroups.has(group.group);
                return (
                <div key={group.group} className="space-y-2">
                  <button
                    type="button"
                    onClick={() => toggleBusinessGroup(group.group)}
                    className="flex h-7 w-full items-center gap-1 rounded-md px-1 text-left text-xs font-extrabold text-gray-400 transition hover:bg-gray-50 hover:text-gray-700"
                    title={collapsed ? '展开业务' : '收起业务'}
                  >
                    <ChevronRight size={14} className={clsx('shrink-0 transition-transform', !collapsed && 'rotate-90')} />
                    <span className="truncate">{group.group}</span>
                  </button>
                  {!collapsed && group.items.map(item => {
                    const option = getTypeOption(item.itemType);
                    const Icon = option.icon;
                    const active = selectedItem?.ID === item.ID;
                    return (
                      <button
                        key={item.ID}
                        type="button"
                        onClick={() => setSelectedId(item.ID)}
                        className={clsx(
                          'group w-full rounded-xl border p-4 text-left transition',
                          active ? 'border-blue-200 bg-blue-50 shadow-sm' : 'border-gray-200 bg-white hover:border-gray-300 hover:bg-gray-50'
                        )}
                      >
                        <div className="flex min-w-0 items-center gap-3">
                          <div className={clsx(
                            'flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border transition',
                            active ? option.tone : 'border-gray-200 bg-gray-50 text-gray-500 group-hover:text-gray-700'
                          )}>
                            <Icon size={18} />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <div className={clsx('truncate text-sm font-extrabold', active ? 'text-blue-900' : 'text-gray-900')}>
                                {item.title || '未命名'}
                              </div>
                              {item.isPinned && <Star size={13} className="shrink-0 fill-amber-400 text-amber-400" />}
                            </div>
                            <div className="mt-1 flex min-w-0 items-center gap-2 text-xs font-semibold text-gray-400">
                              <span>{option.label}</span>
                              <span className="h-1 w-1 rounded-full bg-gray-300" />
                              <span className="truncate">{item.language || defaultLanguageByType[item.itemType]}</span>
                            </div>
                          </div>
                        </div>
                      </button>
                    );
                  })}
                </div>
                );
              })
            )}
          </div>
        </aside>

        <section className="min-w-0 p-6">
        {!hasProjectContext ? (
          <div className="flex min-h-[420px] items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white text-center">
            <div>
              <ClipboardCheck size={36} className="mx-auto mb-3 text-gray-300" />
              <h3 className="text-base font-bold text-gray-900">请选择项目</h3>
            </div>
          </div>
        ) : loading ? (
          <div className="flex min-h-[420px] items-center justify-center text-sm text-gray-400">
            <RefreshCw size={18} className="mr-2 animate-spin" />
            加载中...
          </div>
        ) : !selectedItem ? (
          <div className="flex min-h-[420px] items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white text-center">
            <div>
              <ClipboardCheck size={36} className="mx-auto mb-3 text-gray-300" />
              <h3 className="text-base font-bold text-gray-900">暂无准备项</h3>
              <button
                type="button"
                onClick={() => openCreate()}
                className="mt-4 inline-flex h-9 items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 text-xs font-bold text-gray-700 transition hover:bg-gray-50"
              >
                <Plus size={14} />
                新增
              </button>
            </div>
          </div>
        ) : (
          <article className="flex min-h-[520px] flex-col rounded-lg border border-gray-200 bg-white shadow-sm">
            {(() => {
              const option = getTypeOption(selectedItem.itemType);
              const Icon = option.icon;
              const stats = getContentStats(quickContent);
              const previewLanguage = selectedItem.language || defaultLanguageByType[selectedItem.itemType];
              const hasQuickChanges = quickContent !== (selectedItem.content || '');
              const businessGroup = getBusinessGroupLabel(selectedItem.businessGroup);
              const displayTitle = businessGroup === ungroupedBusinessLabel
                ? (selectedItem.title || '未命名')
                : `${businessGroup} / ${selectedItem.title || '未命名'}`;
              return (
                <>
                  <div className="flex items-start justify-between gap-4 border-b border-gray-100 p-5">
                    <div className="flex min-w-0 items-start gap-3">
                      <div className={clsx('flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border', option.tone)}>
                        <Icon size={18} />
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <h2 className="truncate text-lg font-extrabold text-gray-900">{displayTitle}</h2>
                          {selectedItem.isPinned && <Star size={14} className="shrink-0 fill-amber-400 text-amber-400" />}
                        </div>
                        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-400">
                          <span>{option.label}</span>
                          <span className="h-1 w-1 rounded-full bg-gray-300" />
                          <span>{previewLanguage}</span>
                          <span className="h-1 w-1 rounded-full bg-gray-300" />
                          <span>{stats.lines} 行</span>
                          <span className="h-1 w-1 rounded-full bg-gray-300" />
                          <span>{formatDateTime(selectedItem.UpdatedAt)}</span>
                        </div>
                      </div>
                    </div>
                    <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
                      <div className="hidden h-9 items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 text-xs font-bold text-gray-500 xl:flex">
                        <ClipboardCheck size={14} className="text-blue-500" />
                        <span>开发准备</span>
                        <span className="h-1 w-1 rounded-full bg-gray-300" />
                        <Folder size={13} />
                        <span className="max-w-28 truncate">{activeProjectLabel}</span>
                      </div>
                      <button
                        type="button"
                        onClick={() => openCreate()}
                        disabled={!hasProjectContext}
                        className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-blue-600 px-3 text-xs font-bold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-gray-200"
                        title="新增"
                      >
                        <Plus size={14} />
                        新增
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleQuickSave()}
                        disabled={!hasQuickChanges || quickSaving}
                        className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-gray-900 px-3 text-xs font-bold text-white transition hover:bg-gray-800 disabled:cursor-not-allowed disabled:bg-gray-200 disabled:text-gray-400"
                        title="保存内容"
                      >
                        {quickSaving ? <RefreshCw size={14} className="animate-spin" /> : <Save size={14} />}
                        保存
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleCopy(selectedItem)}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-500 transition hover:bg-gray-50 hover:text-gray-900"
                        title="复制"
                      >
                        <Copy size={15} />
                      </button>
                      <button
                        type="button"
                        onClick={() => openEdit(selectedItem)}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-500 transition hover:bg-gray-50 hover:text-blue-600"
                        title="编辑"
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleDelete(selectedItem)}
                        disabled={deletingId === selectedItem.ID}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-100 bg-red-50 text-red-500 transition hover:bg-red-100 disabled:cursor-wait disabled:opacity-50"
                        title="删除"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </div>
                  <div className="min-h-0 flex-1 p-5">
                    {selectedItem.summary && <p className="mb-4 text-sm leading-6 text-gray-600">{selectedItem.summary}</p>}
                    <div className="overflow-hidden rounded-lg border border-gray-200">
                      <div className="flex h-9 items-center gap-2 border-b border-gray-200 bg-gray-50 px-3 text-xs font-bold text-gray-500">
                        <Code2 size={14} />
                        <span>{previewLanguage}</span>
                      </div>
                      <div className="h-[560px] min-h-[360px]">
                        <Editor
                          height="100%"
                          language={previewLanguage}
                          value={quickContent}
                          onChange={value => setQuickContent(value || '')}
                          theme="vs"
                          options={{
                            minimap: { enabled: false },
                            fontSize: 13,
                            fontFamily: '"Fira Code", Monaco, "Courier New", monospace',
                            lineNumbers: 'on',
                            scrollBeyondLastLine: false,
                            wordWrap: 'off',
                            automaticLayout: true,
                            tabSize: 2,
                            padding: { top: 12, bottom: 12 },
                            renderLineHighlight: 'none',
                          }}
                        />
                      </div>
                    </div>
                  </div>
                </>
              );
            })()}
          </article>
        )}
        </section>
      </main>

      {modalOpen && draft && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 p-4">
          <div className="flex max-h-[88vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl">
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
              <h2 className="text-base font-extrabold text-gray-900">{draft.ID ? '编辑开发准备' : '新增开发准备'}</h2>
              <button
                type="button"
                onClick={closeModal}
                className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 transition hover:bg-gray-100 hover:text-gray-900"
                title="关闭"
              >
                <X size={17} />
              </button>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto p-4 scrollbar-thin">
              <div className="space-y-3">
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-[180px_minmax(0,1fr)]">
                  <input
                    value={draft.businessGroup || ''}
                    onChange={event => patchDraft('businessGroup', event.target.value)}
                    list="development-prepare-business-options"
                    placeholder="选择或输入业务"
                    className="h-10 w-full rounded-lg border border-gray-200 bg-gray-50 px-3 text-sm font-bold text-gray-900 outline-none transition focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5"
                  />
                  <datalist id="development-prepare-business-options">
                    {businessOptions.map(option => (
                      <option key={option} value={option} />
                    ))}
                  </datalist>
                  <input
                    value={draft.title}
                    onChange={event => patchDraft('title', event.target.value)}
                    placeholder="标题"
                    className="h-10 w-full rounded-lg border border-gray-200 bg-gray-50 px-3 text-sm font-bold text-gray-900 outline-none transition focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5"
                  />
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <div className="flex flex-wrap gap-2">
                    {typeOptions.map(option => {
                      const Icon = option.icon;
                      const active = draft.itemType === option.key;
                      return (
                        <button
                          key={option.key}
                          type="button"
                          onClick={() => changeDraftType(option.key)}
                          className={clsx(
                            'inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border px-2.5 text-xs font-bold transition',
                            active ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50'
                          )}
                          title={option.label}
                        >
                          <Icon size={13} />
                          {option.label}
                        </button>
                      );
                    })}
                  </div>

                  <input
                    value={draft.language || ''}
                    onChange={event => patchDraft('language', event.target.value)}
                    placeholder="语言"
                    className="h-8 w-full rounded-lg border border-gray-200 bg-gray-50 px-2.5 text-xs font-semibold text-gray-700 outline-none transition focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5 sm:ml-auto sm:w-36"
                  />
                </div>

                <div className="overflow-hidden rounded-lg border border-gray-200">
                  <div className="flex h-9 items-center gap-2 border-b border-gray-200 bg-gray-50 px-3 text-xs font-bold text-gray-500">
                    <Code2 size={14} />
                    <span>{draftLanguage}</span>
                  </div>
                  <div className="h-[420px]">
                    <Editor
                      height="100%"
                      language={draftLanguage}
                      value={draft.content}
                      onChange={value => patchDraft('content', value || '')}
                      theme="vs"
                      options={{
                        minimap: { enabled: false },
                        fontSize: 13,
                        fontFamily: '"Fira Code", Monaco, "Courier New", monospace',
                        lineNumbers: 'on',
                        scrollBeyondLastLine: false,
                        wordWrap: 'on',
                        automaticLayout: true,
                        tabSize: 2,
                        padding: { top: 12, bottom: 12 },
                      }}
                    />
                  </div>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 border-t border-gray-200 bg-gray-50 px-4 py-3">
              <button
                type="button"
                onClick={closeModal}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-gray-200 bg-white px-3 text-sm font-bold text-gray-600 transition hover:bg-gray-100"
              >
                取消
              </button>
              <button
                type="button"
                onClick={() => void handleSave()}
                disabled={saving}
                className="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-gray-900 px-3 text-sm font-bold text-white transition hover:bg-gray-800 disabled:cursor-wait disabled:bg-gray-300"
              >
                {saving ? <RefreshCw size={15} className="animate-spin" /> : <Save size={15} />}
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
