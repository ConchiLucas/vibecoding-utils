import React, { useEffect, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import toast from 'react-hot-toast';
import clsx from 'clsx';
import {
  Check,
  Code,
  Download,
  FileCode,
  Folder,
  Layers,
  Loader2,
  Pencil,
  Play,
  Plus,
  ScrollText,
  Search,
  Server,
  Settings2,
  Terminal,
  Trash2,
  Upload,
  X,
} from 'lucide-react';
import { useUserStore } from '../../stores/useUserStore';
import {
  ScriptCategory,
  ScriptExecution,
  ScriptResourceCategory,
  ScriptResourceCategoryType,
  ScriptResourceConfig,
  ScriptResourceRow,
  ScriptStep,
  ScriptStepType,
  ScriptWorkflow,
  createScriptResourceCategory,
  createScriptResourceConfig,
  createScriptCategory,
  createScriptStep,
  createScriptWorkflow,
  deleteScriptResourceCategory,
  deleteScriptResourceConfig,
  deleteScriptCategory,
  deleteScriptStep,
  deleteScriptWorkflow,
  getScriptResourceCategories,
  getScriptCategories,
  getScriptExecutions,
  getScriptWorkflows,
  updateScriptResourceCategory,
  updateScriptResourceConfig,
  updateScriptCategory,
  updateScriptStep,
  updateScriptWorkflow,
} from '../../api/scriptManager';

const STEP_TYPE_META: Record<ScriptStepType, { label: string; icon: React.ElementType; className: string }> = {
  local_exec: {
    label: '本地执行',
    icon: Terminal,
    className: 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200',
  },
  local_upload: {
    label: '本地上传',
    icon: Upload,
    className: 'bg-sky-100 text-sky-700 hover:bg-sky-200',
  },
  target_download: {
    label: '目标下载',
    icon: Download,
    className: 'bg-amber-100 text-amber-800 hover:bg-amber-200',
  },
  target_exec: {
    label: '目标执行',
    icon: Server,
    className: 'bg-violet-100 text-violet-700 hover:bg-violet-200',
  },
};

const STEP_TYPES = Object.keys(STEP_TYPE_META) as ScriptStepType[];

const defaultStepScript: Record<ScriptStepType, string> = {
  local_exec: '# 在 easy-deploy 本机执行\npwd\n',
  local_upload: '# 在 easy-deploy 本机执行上传脚本，可写 scp/rsync 并使用资源配置用途变量\n# scp -P "$TARGET_SERVER_TARGET_TAILSCALE_PORT" /tmp/dump.sql "$TARGET_SERVER_USER@$TARGET_SERVER_TARGET_TAILSCALE_IP:/tmp/dump.sql"\n',
  target_download: '# 在目标服务器执行下载脚本，可使用资源配置用途变量\n# scp "$TARGET_SERVER_USER@$TARGET_SERVER_TARGET_TAILSCALE_IP:/tmp/dump.sql" /tmp/dump.sql\n',
  target_exec: '# 在目标服务器执行\npwd\n',
};

const emptyCategory: Partial<ScriptCategory> = { ID: 0, categoryName: '', description: '' };
const emptyWorkflow: Partial<ScriptWorkflow> = { ID: 0, categoryId: 0, workflowName: '', description: '' };

type PlaceholderValueKind = 'manual' | 'connection' | 'server' | 'resource';

interface PlaceholderDraft {
  placeholder: string;
  name?: string;
  valueKind: PlaceholderValueKind;
  value: string;
  resourceCategoryId?: number;
  resourceConfigId?: number;
  customValue?: string;
}

const emptyPlaceholder: PlaceholderDraft = {
  placeholder: '',
  name: '',
  valueKind: 'resource',
  value: '',
  resourceCategoryId: 0,
  resourceConfigId: 0,
  customValue: '',
};

const emptyOverridePlaceholder: PlaceholderDraft = {
  placeholder: '',
  name: '',
  valueKind: 'manual',
  value: '',
  resourceCategoryId: 0,
  resourceConfigId: 0,
  customValue: '',
};

function normalizeEnvName(value?: string) {
  return (value || '')
    .trim()
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
}

function inferPlaceholderName(category?: ScriptResourceCategory, stepType: ScriptStepType = 'local_exec', resourceRole = '') {
  const categoryName = category?.categoryName || '';
  const normalizedRole = normalizeEnvName(resourceRole);
  if (categoryName.includes('数据库')) {
    if (normalizedRole === 'SOURCE') return 'SOURCE_DB';
    if (normalizedRole === 'TARGET') return 'TARGET_DB';
    return stepType === 'target_exec' ? 'TARGET_DB' : 'SOURCE_DB';
  }
  if (categoryName.includes('服务器')) {
    if (normalizedRole === 'SOURCE') return 'SOURCE_SERVER';
    return 'TARGET_SERVER';
  }
  if (categoryName.includes('部署脚本') || categoryName.includes('流程')) {
    return 'DEPLOY_FLOW';
  }
  return normalizeEnvName(categoryName) || 'RESOURCE';
}

function inferResourceRole(category?: ScriptResourceCategory, stepType: ScriptStepType = 'local_exec') {
  const categoryName = category?.categoryName || '';
  if (categoryName.includes('数据库')) {
    return stepType === 'target_exec' ? 'target' : 'source';
  }
  if (categoryName.includes('服务器')) {
    return stepType === 'local_exec' ? 'source' : 'target';
  }
  return '';
}

function normalizeResourceRoleForStep(category: ScriptResourceCategory | undefined, stepType: ScriptStepType, role?: string, name?: string) {
  const categoryName = category?.categoryName || '';
  const normalizedRole = normalizeEnvName(role);
  const normalizedName = normalizeEnvName(name);
  if (categoryName.includes('服务器') && stepType === 'local_exec' && isAutoPlaceholderName(category, normalizedName)) {
    return 'source';
  }
  return normalizedRole ? normalizedRole.toLowerCase() : inferResourceRole(category, stepType);
}

function isAutoPlaceholderName(category: ScriptResourceCategory | undefined, name?: string) {
  const normalizedName = (name || '').trim().toUpperCase();
  if (!normalizedName) return true;
  const categoryName = category?.categoryName || '';
  if (categoryName.includes('数据库')) return normalizedName === 'SOURCE_DB' || normalizedName === 'TARGET_DB';
  if (categoryName.includes('服务器')) return normalizedName === 'SOURCE_SERVER' || normalizedName === 'TARGET_SERVER';
  if (categoryName.includes('部署脚本') || categoryName.includes('流程')) return normalizedName === 'DEPLOY_FLOW';
  return false;
}

function findResourceCategoryByConfigId(resourceCategories: ScriptResourceCategory[], configId?: number) {
  const numericConfigId = Number(configId || 0);
  if (!numericConfigId) return undefined;
  return resourceCategories.find((category) => (
    (category.configs || []).some((config) => config.ID === numericConfigId)
  ));
}

function isResourceBinding(item: PlaceholderDraft) {
  return item.valueKind !== 'manual';
}

function applyStepTypeToPlaceholderNames(placeholders: PlaceholderDraft[], resourceCategories: ScriptResourceCategory[], stepType: ScriptStepType) {
  return placeholders.map((placeholder) => {
    if (!isResourceBinding(placeholder)) {
      return placeholder;
    }
    const category = resourceCategories.find((resourceCategory) => resourceCategory.ID === Number(placeholder.resourceCategoryId || 0)) ||
      findResourceCategoryByConfigId(resourceCategories, placeholder.resourceConfigId);
    if (!category || !isAutoPlaceholderName(category, placeholder.name)) {
      return placeholder;
    }
      const customValue = normalizeResourceRoleForStep(category, stepType, placeholder.customValue, placeholder.name);
    return {
      ...placeholder,
      name: inferPlaceholderName(category, stepType, customValue),
      customValue,
      placeholder: placeholder.placeholder || category.categoryName,
      resourceCategoryId: placeholder.resourceCategoryId || category.ID,
    };
  });
}

function parsePlaceholders(raw?: string, resourceCategories: ScriptResourceCategory[] = [], stepType: ScriptStepType = 'local_exec'): PlaceholderDraft[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.map((item) => {
      const valueKind = (item.valueKind === 'connection' || item.valueKind === 'server' || item.valueKind === 'resource' ? item.valueKind : 'manual') as PlaceholderValueKind;
      if (valueKind === 'connection' || valueKind === 'server') {
        const legacyConfigId = Number(item.resourceConfigId || item.value || 0);
        const legacyCategory = resourceCategories.find((category) => (
          valueKind === 'connection' ? category.categoryName.includes('数据库') : category.categoryName.includes('服务器')
        ));
        const legacyConfig = (legacyCategory?.configs || []).find((config) => config.ID === legacyConfigId);
        if (legacyCategory && legacyConfig) {
          const customValue = normalizeResourceRoleForStep(legacyCategory, stepType, '', '');
          const name = inferPlaceholderName(legacyCategory, stepType, customValue);
          return {
            placeholder: legacyCategory.categoryName,
            name,
            valueKind: 'resource' as PlaceholderValueKind,
            value: String(legacyConfig.ID),
            resourceCategoryId: legacyCategory.ID,
            resourceConfigId: legacyConfig.ID,
            customValue,
          };
        }
      }
      const resourceConfigId = valueKind === 'resource'
        ? Number(item.resourceConfigId || item.value || 0)
        : Number(item.resourceConfigId || 0);
      let category = Number(item.resourceCategoryId || 0)
        ? resourceCategories.find((resourceCategory) => resourceCategory.ID === Number(item.resourceCategoryId || 0))
        : findResourceCategoryByConfigId(resourceCategories, resourceConfigId);
      let resolvedResourceConfigId = resourceConfigId;
      const rawName = String(item.name || '').trim();
      if (!category && valueKind === 'resource' && rawName.toUpperCase() === 'EXEC_PARAMS') {
        const dynamicCategory = resourceCategories.find((resourceCategory) => (
          resourceCategory.categoryType === 'dynamic' && (resourceCategory.configs || []).length > 0
        ));
        const dynamicConfig = dynamicCategory?.configs?.[0];
        if (dynamicCategory && dynamicConfig) {
          category = dynamicCategory;
          resolvedResourceConfigId = dynamicConfig.ID;
        }
      }
      if (category?.categoryType === 'dynamic' && valueKind === 'resource' && rawName.toUpperCase() === 'EXEC_PARAMS') {
        const configExistsInScope = (category.configs || []).some((config) => config.ID === resolvedResourceConfigId);
        const scopedConfig = category.configs?.[0];
        if (!configExistsInScope && scopedConfig) {
          resolvedResourceConfigId = scopedConfig.ID;
        }
      }
      const customValue = category?.categoryType === 'fixed'
        ? normalizeResourceRoleForStep(category, stepType, String(item.customValue || ''), rawName)
        : String(item.customValue || '');
      const name = rawName || inferPlaceholderName(category, stepType, customValue);
      return {
        placeholder: String(item.placeholder || item.description || category?.categoryName || name || ''),
        name,
        valueKind,
        value: valueKind === 'resource' ? String(resolvedResourceConfigId || item.value || '') : String(item.value || ''),
        resourceCategoryId: Number(item.resourceCategoryId || category?.ID || 0),
        resourceConfigId: resolvedResourceConfigId,
        customValue,
      };
    });
  } catch {
    return [];
  }
}

function serializePlaceholders(placeholders: PlaceholderDraft[], resourceCategories: ScriptResourceCategory[], stepType: ScriptStepType = 'local_exec') {
  const cleaned = placeholders
    .map((item) => {
      const resourceCategoryId = Number(item.resourceCategoryId || 0);
      const resourceConfigId = Number(item.resourceConfigId || 0);
      const category = resourceCategories.find((resourceCategory) => resourceCategory.ID === resourceCategoryId) ||
        findResourceCategoryByConfigId(resourceCategories, resourceConfigId);
      const isResource = item.valueKind === 'resource';
      const isLegacyBinding = item.valueKind === 'connection' || item.valueKind === 'server';
      const customValue = item.customValue?.trim() || '';
      const name = item.name?.trim() || (isResource ? inferPlaceholderName(category, stepType, customValue) : '');
      return {
        placeholder: item.placeholder.trim() || (isResource ? category?.categoryName : '') || name,
        name,
        valueKind: (isResource ? 'resource' : isLegacyBinding ? item.valueKind : 'manual') as PlaceholderValueKind,
        value: isResource ? (resourceConfigId ? String(resourceConfigId) : item.value.trim()) : item.value.trim(),
        resourceCategoryId: Number(category?.ID || resourceCategoryId || 0),
        resourceConfigId,
        customValue,
      };
    })
    .filter((item) => (item.valueKind === 'resource' ? item.resourceConfigId : item.name || item.value));
  return cleaned.length > 0 ? JSON.stringify(cleaned) : '';
}

const emptyResourceRow: ScriptResourceRow = { name: '', placeholder: '', value: '' };

function parseResourceRows(raw?: string): ScriptResourceRow[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.map((item) => ({
      name: String(item.name || ''),
      placeholder: String(item.placeholder || item.description || ''),
      value: String(item.value || ''),
    }));
  } catch {
    return [];
  }
}

function serializeResourceRows(rows: ScriptResourceRow[]) {
  const cleaned = rows
    .map((item) => ({
      name: item.name.trim(),
      placeholder: item.placeholder.trim(),
      value: item.value.trim(),
    }))
    .filter((item) => item.name || item.placeholder || item.value);
  return cleaned.length > 0 ? JSON.stringify(cleaned) : '';
}

function configWorkflowId(config?: ScriptResourceConfig) {
  return Number(config?.workflowId || 0);
}

function scopeResourceCategories(categories: ScriptResourceCategory[], workflowId?: number): ScriptResourceCategory[] {
  const numericWorkflowId = Number(workflowId || 0);
  return categories.map((category) => {
    const categoryConfigs = category.configs || [];
    let configs = category.categoryType === 'dynamic'
      ? categoryConfigs.filter((config) => configWorkflowId(config) === numericWorkflowId)
      : categoryConfigs.filter((config) => configWorkflowId(config) === 0);
    if (category.categoryType === 'dynamic' && numericWorkflowId !== 0 && configs.length === 0) {
      configs = categoryConfigs
        .filter((config) => configWorkflowId(config) === 0)
        .map((config) => ({ ...config, workflowId: numericWorkflowId }));
    }
    return { ...category, configs };
  });
}

export default function ScriptManager() {
  const [categories, setCategories] = useState<ScriptCategory[]>([]);
  const [workflows, setWorkflows] = useState<ScriptWorkflow[]>([]);
  const [resourceCategories, setResourceCategories] = useState<ScriptResourceCategory[]>([]);
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);

  const [categoryDraft, setCategoryDraft] = useState<Partial<ScriptCategory> | null>(null);
  const [workflowDraft, setWorkflowDraft] = useState<Partial<ScriptWorkflow> | null>(null);
  const [stepDraft, setStepDraft] = useState<Partial<ScriptStep> | null>(null);
  const [placeholderDrafts, setPlaceholderDrafts] = useState<PlaceholderDraft[]>([]);
  const [resourceManagerTarget, setResourceManagerTarget] = useState<{ mode: ScriptResourceCategoryType; workflow?: ScriptWorkflow } | null>(null);
  const [scriptDraft, setScriptDraft] = useState<{ step: ScriptStep; content: string } | null>(null);
  const [logState, setLogState] = useState<{ open: boolean; title: string; executions: ScriptExecution[]; active?: ScriptExecution }>({
    open: false,
    title: '',
    executions: [],
  });
  const [executeTarget, setExecuteTarget] = useState<{ target: 'workflow' | 'step'; id: number; title: string } | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [categoryRes, workflowRes, resourceCategoryRes] = await Promise.all([
        getScriptCategories(),
        getScriptWorkflows({ page: 1, pageSize: 200 }),
        getScriptResourceCategories(),
      ]);
      if (categoryRes.code === 0) setCategories(categoryRes.data || []);
      if (workflowRes.code === 0) setWorkflows(workflowRes.data?.list || []);
      if (resourceCategoryRes.code === 0) setResourceCategories(resourceCategoryRes.data || []);
    } catch {
      toast.error('脚本库数据加载失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const filteredWorkflows = useMemo(() => {
    const keyword = searchQuery.trim().toLowerCase();
    return workflows.filter((workflow) => {
      const categoryOk = selectedCategoryId === null || workflow.categoryId === selectedCategoryId;
      const searchOk = !keyword ||
        workflow.workflowName.toLowerCase().includes(keyword) ||
        (workflow.description || '').toLowerCase().includes(keyword);
      return categoryOk && searchOk;
    });
  }, [workflows, selectedCategoryId, searchQuery]);

  const categoryCount = (categoryId: number) => workflows.filter((item) => item.categoryId === categoryId).length;

  const openWorkflowDrawer = (workflow?: ScriptWorkflow) => {
    setWorkflowDraft(workflow ? { ...workflow } : {
      ...emptyWorkflow,
      categoryId: selectedCategoryId || categories[0]?.ID || 0,
    });
  };

  const openStepDrawer = (workflow: ScriptWorkflow, step?: ScriptStep) => {
    const stepType: ScriptStepType = step?.stepType || 'local_exec';
    const workflowResourceCategories = scopeResourceCategories(resourceCategories, workflow.ID);
    setPlaceholderDrafts(parsePlaceholders(step?.placeholders, workflowResourceCategories, stepType));
    setStepDraft(step ? { ...step } : {
      ID: 0,
      workflowId: workflow.ID,
      stepName: '',
      stepType,
      scriptContent: defaultStepScript[stepType],
      placeholders: '',
    });
  };

  const closeStepDrawer = () => {
    setStepDraft(null);
    setPlaceholderDrafts([]);
  };

  const saveCategory = async () => {
    if (!categoryDraft?.categoryName?.trim()) {
      toast.error('请填写分类名称');
      return;
    }
    try {
      const res = categoryDraft.ID
        ? await updateScriptCategory(categoryDraft.ID, categoryDraft)
        : await createScriptCategory(categoryDraft);
      if (res.code === 0) {
        toast.success('分类已保存');
        setCategoryDraft(null);
        fetchData();
      }
    } catch {
      toast.error('分类保存失败');
    }
  };

  const saveWorkflow = async () => {
    if (!workflowDraft?.workflowName?.trim()) {
      toast.error('请填写流程名称');
      return;
    }
    try {
      const res = workflowDraft.ID
        ? await updateScriptWorkflow(workflowDraft.ID, workflowDraft)
        : await createScriptWorkflow(workflowDraft);
      if (res.code === 0) {
        toast.success('流程已保存');
        setWorkflowDraft(null);
        fetchData();
      }
    } catch {
      toast.error('流程保存失败');
    }
  };

  const saveStep = async () => {
    if (!stepDraft?.stepName?.trim()) {
      toast.error('请填写步骤名称');
      return;
    }
    const workflowResourceCategories = scopeResourceCategories(resourceCategories, stepDraft.workflowId);
    const payload = {
      ...stepDraft,
      placeholders: serializePlaceholders(placeholderDrafts, workflowResourceCategories, stepDraft.stepType || 'local_exec'),
    };
    try {
      const res = payload.ID
        ? await updateScriptStep(payload.ID, payload)
        : await createScriptStep(payload);
      if (res.code === 0) {
        toast.success('步骤已保存');
        closeStepDrawer();
        fetchData();
      }
    } catch {
      toast.error('步骤保存失败');
    }
  };

  const saveScriptContent = async () => {
    if (!scriptDraft) return;
    try {
      const res = await updateScriptStep(scriptDraft.step.ID, { ...scriptDraft.step, scriptContent: scriptDraft.content });
      if (res.code === 0) {
        toast.success('脚本已保存');
        setScriptDraft(null);
        fetchData();
      }
    } catch {
      toast.error('脚本保存失败');
    }
  };

  const removeCategory = async (category: ScriptCategory) => {
    if (!window.confirm(`确定删除分类「${category.categoryName}」吗？`)) return;
    try {
      const res = await deleteScriptCategory(category.ID);
      if (res.code === 0) {
        toast.success('分类已删除');
        if (selectedCategoryId === category.ID) setSelectedCategoryId(null);
        fetchData();
      }
    } catch {
      toast.error('分类删除失败');
    }
  };

  const removeWorkflow = async (workflow: ScriptWorkflow) => {
    if (!window.confirm(`确定删除流程「${workflow.workflowName}」吗？`)) return;
    try {
      const res = await deleteScriptWorkflow(workflow.ID);
      if (res.code === 0) {
        toast.success('流程已删除');
        fetchData();
      }
    } catch {
      toast.error('流程删除失败');
    }
  };

  const removeStep = async (step: ScriptStep) => {
    if (!window.confirm(`确定删除步骤「${step.stepName}」吗？`)) return;
    try {
      const res = await deleteScriptStep(step.ID);
      if (res.code === 0) {
        toast.success('步骤已删除');
        fetchData();
      }
    } catch {
      toast.error('步骤删除失败');
    }
  };

  const openLogs = async (target: 'workflow' | 'step', entity: ScriptWorkflow | ScriptStep) => {
    const isWorkflow = target === 'workflow';
    const title = isWorkflow ? (entity as ScriptWorkflow).workflowName : (entity as ScriptStep).stepName;
    const params = isWorkflow
      ? { page: 1, pageSize: 20, workflowId: entity.ID, scope: 'workflow' as const }
      : { page: 1, pageSize: 20, stepId: entity.ID, scope: 'step' as const };
    try {
      const res = await getScriptExecutions(params);
      if (res.code === 0) {
        const executions = res.data?.list || [];
        setLogState({ open: true, title, executions, active: executions[0] });
      }
    } catch {
      toast.error('日志加载失败');
    }
  };

  const currentCategoryName = selectedCategoryId === null
    ? '全部脚本'
    : categories.find((item) => item.ID === selectedCategoryId)?.categoryName || '未分类';
  const stepResourceCategories = useMemo(() => (
    scopeResourceCategories(resourceCategories, stepDraft?.workflowId)
  ), [resourceCategories, stepDraft?.workflowId]);
  const stepEditorContext = useMemo<ScriptStep | undefined>(() => {
    if (!stepDraft) return undefined;
    const stepType = stepDraft.stepType || 'local_exec';
    return {
      ID: Number(stepDraft.ID || 0),
      workflowId: Number(stepDraft.workflowId || 0),
      stepName: stepDraft.stepName || '',
      stepType,
      scriptContent: stepDraft.scriptContent || '',
      placeholders: serializePlaceholders(placeholderDrafts, stepResourceCategories, stepType),
    };
  }, [placeholderDrafts, stepDraft, stepResourceCategories]);

  return (
    <div className="w-full flex gap-0 relative">
      <aside className="w-56 shrink-0 bg-white border-r border-gray-100 min-h-[calc(100vh-64px)] flex flex-col">
        <div className="px-3 pt-4 pb-2">
          <div className="relative mb-3">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="搜索脚本..."
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              className="w-full bg-gray-50 border border-gray-200 rounded-md py-1.5 pl-8 pr-2 text-xs focus:outline-none focus:ring-2 focus:ring-black/5 focus:border-gray-300"
            />
          </div>
          <div className="flex items-center justify-between mb-2 px-1">
            <span className="text-[10px] font-bold text-gray-400 uppercase tracking-wider">业务分类</span>
            <button
              type="button"
              onClick={() => setCategoryDraft({ ...emptyCategory })}
              className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors"
              title="新增分类"
            >
              <Plus size={14} strokeWidth={2.5} />
            </button>
          </div>
        </div>

        <div className="px-3 flex flex-col gap-0.5">
          <button
            type="button"
            onClick={() => setSelectedCategoryId(null)}
            className={clsx(
              'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
              selectedCategoryId === null ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
            )}
          >
            <Folder size={14} className={selectedCategoryId === null ? 'text-white' : 'text-gray-400'} />
            <span className="truncate">全部脚本</span>
            <span className={clsx(
              'ml-auto text-xs px-1.5 py-0.5 rounded-full font-mono',
              selectedCategoryId === null ? 'bg-white/20 text-white' : 'bg-gray-100 text-gray-500'
            )}>{workflows.length}</span>
          </button>

          <div className="ml-4 pl-3 border-l-2 border-gray-200 flex flex-col gap-0.5">
            {categories.map((category) => {
              const active = selectedCategoryId === category.ID;
              return (
                <div key={category.ID} className="group/item relative">
                  <button
                    type="button"
                    onClick={() => setSelectedCategoryId(category.ID)}
                    className={clsx(
                      'w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                      active ? 'bg-gray-900 text-white' : 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'
                    )}
                  >
                    <Folder size={12} className={active ? 'text-white' : 'text-gray-400'} />
                    <span className="truncate flex-1 text-left">{category.categoryName}</span>
                    <span className={clsx(
                      'text-xs px-1.5 py-0.5 rounded-full font-mono',
                      active ? 'bg-white/20 text-white' : 'bg-gray-100 text-gray-500'
                    )}>{categoryCount(category.ID)}</span>
                    {!active && (
                      <span className="hidden group-hover/item:flex items-center gap-0.5 absolute right-2">
                        <span
                          onClick={(event) => { event.stopPropagation(); setCategoryDraft({ ...category }); }}
                          className="p-1 rounded text-gray-400 hover:text-gray-700 hover:bg-gray-100 cursor-pointer"
                          title="编辑分类"
                        ><Pencil size={11} /></span>
                        <span
                          onClick={(event) => { event.stopPropagation(); removeCategory(category); }}
                          className="p-1 rounded text-gray-400 hover:text-red-600 hover:bg-red-50 cursor-pointer"
                          title="删除分类"
                        ><Trash2 size={11} /></span>
                      </span>
                    )}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </aside>

      <main className="flex-1 min-w-0 px-6 py-6">
        <div className="mb-5 flex flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-gray-900">{currentCategoryName}</h1>
            <p className="mt-1 text-xs text-gray-400">{filteredWorkflows.length} 个脚本流程</p>
          </div>
          <div className="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              onClick={() => setResourceManagerTarget({ mode: 'fixed' })}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm transition hover:bg-gray-50"
            >
              <Folder size={15} />
              资源配置
            </button>
            <button
              type="button"
              onClick={() => setResourceManagerTarget({ mode: 'constant' })}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm transition hover:bg-gray-50"
            >
              <Settings2 size={15} />
              常量配置
            </button>
            <button
              type="button"
              onClick={() => openWorkflowDrawer()}
              className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-gray-700"
            >
              <Plus size={15} />
              新建流程
            </button>
          </div>
        </div>

        {loading ? (
          <div className="h-40 flex items-center justify-center text-gray-400">资源加载中...</div>
        ) : filteredWorkflows.length === 0 ? (
          <div className="border border-dashed border-gray-300 rounded-lg p-12 text-center bg-gray-50 mt-2">
            <FileCode size={32} className="text-gray-300 mx-auto mb-3" />
            <h3 className="text-base font-medium text-gray-900 mb-4">暂无脚本流程</h3>
            <button
              type="button"
              onClick={() => openWorkflowDrawer()}
              className="bg-black hover:bg-gray-800 text-white font-medium py-2 px-5 rounded-lg text-sm transition-colors"
            >
              新建流程
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-5">
            {filteredWorkflows.map((workflow) => (
              <div key={workflow.ID} className="group bg-white rounded-lg shadow-sm border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200 overflow-hidden flex flex-col">
                <div className="px-5 py-4 border-b border-gray-100 flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="text-base font-bold text-gray-900 truncate">{workflow.workflowName}</h2>
                    <p className="text-xs text-gray-400 line-clamp-2 mt-1 min-h-[1.1rem]">{workflow.description || '暂无描述'}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setExecuteTarget({ target: 'workflow', id: workflow.ID, title: workflow.workflowName })}
                      className="p-2 rounded-lg bg-gray-900 text-white hover:bg-gray-700 transition-colors"
                      title="执行全部"
                    >
                      <Play size={15} fill="currentColor" />
                    </button>
                    <button
                      type="button"
                      onClick={() => openLogs('workflow', workflow)}
                      className="p-2 rounded-lg border border-emerald-100 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 transition-colors"
                      title="流程日志"
                    >
                      <ScrollText size={15} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setResourceManagerTarget({ mode: 'dynamic', workflow })}
                      className="p-2 rounded-lg border border-emerald-100 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 transition-colors"
                      title="动态配置"
                    >
                      <Layers size={15} />
                    </button>
                    <button
                      type="button"
                      onClick={() => openWorkflowDrawer(workflow)}
                      className="p-2 rounded-lg border border-gray-200 bg-white text-gray-700 hover:bg-gray-100 transition-colors"
                      title="编辑流程"
                    >
                      <Pencil size={15} />
                    </button>
                    <button
                      type="button"
                      onClick={() => removeWorkflow(workflow)}
                      className="p-2 rounded-lg border border-red-100 bg-red-50 text-red-600 hover:bg-red-100 transition-colors"
                      title="删除流程"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </div>

                <div className="p-4 bg-gray-50/50 flex-1">
                  <div className="flex flex-col gap-2">
                    <span className="text-[10px] font-bold text-gray-400 uppercase tracking-wider mb-0.5">执行步骤</span>
                    {(workflow.steps || []).map((step) => {
                      const meta = STEP_TYPE_META[step.stepType] || STEP_TYPE_META.local_exec;
                      const Icon = meta.icon;
                      return (
                        <div key={step.ID} className="flex items-center gap-1.5">
                          <button
                            type="button"
                            onClick={() => setExecuteTarget({ target: 'step', id: step.ID, title: step.stepName })}
                            className={clsx('flex-1 min-w-0 flex items-center justify-between px-3 py-1.5 rounded-md text-xs font-medium transition-colors', meta.className)}
                            title="执行步骤"
                          >
                            <div className="min-w-0 flex items-center gap-1.5">
                              <Icon size={12} />
                              <span className="shrink-0">{meta.label}</span>
                              <span className="truncate text-current/80">{step.stepName}</span>
                            </div>
                            <Play size={10} className="shrink-0 opacity-30" />
                          </button>
                          <button
                            type="button"
                            onClick={() => setScriptDraft({ step, content: step.scriptContent || '' })}
                            className="flex items-center justify-center p-1.5 rounded-md border border-blue-100 bg-blue-50 text-blue-600 hover:bg-blue-100 transition-colors"
                            title="查看脚本"
                          >
                            <Code size={13} />
                          </button>
                          <button
                            type="button"
                            onClick={() => openLogs('step', step)}
                            className="flex items-center justify-center p-1.5 rounded-md border border-emerald-100 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 transition-colors"
                            title="查看日志"
                          >
                            <ScrollText size={13} />
                          </button>
                          <button
                            type="button"
                            onClick={() => openStepDrawer(workflow, step)}
                            className="flex items-center justify-center p-1.5 rounded-md border border-gray-200 bg-white text-gray-700 hover:bg-gray-100 transition-colors"
                            title="编辑步骤"
                          >
                            <Pencil size={13} />
                          </button>
                          <button
                            type="button"
                            onClick={() => removeStep(step)}
                            className="flex items-center justify-center p-1.5 rounded-md border border-red-100 bg-red-50 text-red-600 hover:bg-red-100 transition-colors"
                            title="删除步骤"
                          >
                            <Trash2 size={13} />
                          </button>
                        </div>
                      );
                    })}
                    <button
                      type="button"
                      onClick={() => openStepDrawer(workflow)}
                      className="mt-0.5 w-full flex items-center justify-center gap-1.5 py-1.5 border border-dashed border-gray-300 rounded-md text-xs text-gray-400 hover:text-gray-700 hover:bg-gray-50 hover:border-gray-400 transition-colors"
                    >
                      <Plus size={11} /> 添加步骤
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {resourceManagerTarget && (
        <ResourceManagerModal
          categories={resourceCategories}
          resourceMode={resourceManagerTarget.mode}
          workflow={resourceManagerTarget.workflow}
          onClose={() => setResourceManagerTarget(null)}
          onRefresh={fetchData}
        />
      )}

      {categoryDraft && (
        <SidePanel title={categoryDraft.ID ? '编辑分类' : '新增分类'} icon={Folder} onClose={() => setCategoryDraft(null)} width="max-w-md">
          <div className="space-y-4">
            <TextField label="分类名称" value={categoryDraft.categoryName || ''} onChange={(value) => setCategoryDraft({ ...categoryDraft, categoryName: value })} />
            <TextArea label="分类描述" value={categoryDraft.description || ''} onChange={(value) => setCategoryDraft({ ...categoryDraft, description: value })} rows={3} />
          </div>
          <PanelActions onCancel={() => setCategoryDraft(null)} onSave={saveCategory} />
        </SidePanel>
      )}

      {workflowDraft && (
        <SidePanel title={workflowDraft.ID ? '编辑脚本流程' : '新增脚本流程'} icon={FileCode} onClose={() => setWorkflowDraft(null)} width="max-w-lg">
          <div className="space-y-4">
            <TextField label="流程名称" value={workflowDraft.workflowName || ''} onChange={(value) => setWorkflowDraft({ ...workflowDraft, workflowName: value })} />
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">业务分类</label>
              <select
                value={workflowDraft.categoryId || 0}
                onChange={(event) => setWorkflowDraft({ ...workflowDraft, categoryId: Number(event.target.value) })}
                className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 bg-white"
              >
                <option value={0}>未分类</option>
                {categories.map((category) => (
                  <option key={category.ID} value={category.ID}>{category.categoryName}</option>
                ))}
              </select>
            </div>
            <TextArea label="流程描述" value={workflowDraft.description || ''} onChange={(value) => setWorkflowDraft({ ...workflowDraft, description: value })} rows={3} />
          </div>
          <PanelActions onCancel={() => setWorkflowDraft(null)} onSave={saveWorkflow} />
        </SidePanel>
      )}

      {stepDraft && (
        <SidePanel title={stepDraft.ID ? '编辑步骤' : '新增步骤'} icon={Terminal} onClose={closeStepDrawer} width="max-w-3xl">
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-4">
              <TextField label="步骤名词" value={stepDraft.stepName || ''} onChange={(value) => setStepDraft({ ...stepDraft, stepName: value })} />
              <StepTypeSelect
                value={stepDraft.stepType || 'local_exec'}
                onChange={(value) => {
                  setStepDraft({
                    ...stepDraft,
                    stepType: value,
                    scriptContent: stepDraft.scriptContent || defaultStepScript[value],
                  });
                  setPlaceholderDrafts((items) => applyStepTypeToPlaceholderNames(items, stepResourceCategories, value));
                }}
              />
            </div>
            <PlaceholderEditor
              placeholders={placeholderDrafts}
              onChange={setPlaceholderDrafts}
              resourceCategories={stepResourceCategories}
              stepType={stepDraft.stepType || 'local_exec'}
              currentStep={stepEditorContext}
            />
          </div>
          <PanelActions onCancel={closeStepDrawer} onSave={saveStep} />
        </SidePanel>
      )}

      {scriptDraft && (
        <SidePanel title={scriptDraft.step.stepName} icon={Code} onClose={() => setScriptDraft(null)} width="max-w-4xl">
          <div className="h-[calc(100vh-180px)] overflow-hidden rounded-lg border border-gray-700 bg-[#1e1e1e]">
            <Editor
              height="100%"
              language="shell"
              theme="vs-dark"
              value={scriptDraft.content}
              onChange={(value) => setScriptDraft({ ...scriptDraft, content: value || '' })}
              options={{ minimap: { enabled: false }, fontSize: 13, wordWrap: 'on' }}
            />
          </div>
          <PanelActions onCancel={() => setScriptDraft(null)} onSave={saveScriptContent} />
        </SidePanel>
      )}

      {logState.open && (
        <LogModal
          title={logState.title}
          executions={logState.executions}
          active={logState.active}
          onSelect={(execution) => setLogState({ ...logState, active: execution })}
          onClose={() => setLogState({ open: false, title: '', executions: [] })}
        />
      )}

      {executeTarget && (
        <ScriptExecutePanel
          target={executeTarget.target}
          id={executeTarget.id}
          title={executeTarget.title}
          onClose={() => {
            setExecuteTarget(null);
            fetchData();
          }}
        />
      )}
    </div>
  );
}

function SidePanel({ title, icon: Icon, width, onClose, children }: {
  title: string;
  icon: React.ElementType;
  width: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-[100] flex justify-end">
      <div className="absolute inset-0 bg-black/20 backdrop-blur-sm" onClick={onClose} />
      <div className={clsx('relative w-full bg-gray-50 h-full shadow-2xl flex flex-col animate-in slide-in-from-right-8 duration-300 border-l border-gray-200', width)}>
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 bg-white">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-black text-white rounded-md"><Icon size={16} /></div>
            <h2 className="text-base font-bold text-gray-900 leading-tight">{title}</h2>
          </div>
          <button onClick={onClose} className="p-2 text-gray-400 hover:text-gray-900 rounded-full hover:bg-gray-100 transition-colors"><X size={18} /></button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">{children}</div>
      </div>
    </div>
  );
}

function PanelActions({ onCancel, onSave }: { onCancel: () => void; onSave: () => void }) {
  return (
    <div className="border-t border-gray-200 p-5 flex items-center justify-end gap-3 bg-white">
      <button type="button" onClick={onCancel} className="px-4 py-2.5 border border-gray-300 text-gray-700 rounded-lg text-sm font-bold hover:bg-gray-100 transition-colors">取消</button>
      <button type="button" onClick={onSave} className="px-5 py-2.5 bg-black text-white rounded-lg text-sm font-bold hover:bg-gray-800 transition-colors shadow-sm flex items-center gap-2">
        <Check size={14} /> 保存
      </button>
    </div>
  );
}

function TextField({ label, value, onChange, type = 'text', placeholder = '' }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5"
      />
    </div>
  );
}

function StepTypeSelect({ value, onChange }: {
  value: ScriptStepType;
  onChange: (value: ScriptStepType) => void;
}) {
  const meta = STEP_TYPE_META[value] || STEP_TYPE_META.local_exec;
  const Icon = meta.icon;

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">步骤类型</label>
      <div className="relative">
        <Icon size={16} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2" />
        <select
          value={value}
          onChange={(event) => onChange(event.target.value as ScriptStepType)}
          className={clsx(
            'w-full rounded-lg border border-transparent py-2.5 pl-9 pr-3 text-sm font-semibold outline-none focus:ring-2 focus:ring-black/10',
            meta.className
          )}
        >
          {STEP_TYPES.map((type) => (
            <option key={type} value={type}>{STEP_TYPE_META[type].label}</option>
          ))}
        </select>
      </div>
    </div>
  );
}

function PlaceholderEditor({ placeholders, onChange, resourceCategories, stepType, currentStep }: {
  placeholders: PlaceholderDraft[];
  onChange: (items: PlaceholderDraft[]) => void;
  resourceCategories: ScriptResourceCategory[];
  stepType: ScriptStepType;
  currentStep?: ScriptStep;
}) {
  const resourceCategoryById = (id?: number) => resourceCategories.find((category) => category.ID === Number(id || 0));
  const resourceCategoryForPlaceholder = (item: PlaceholderDraft) => (
    resourceCategoryById(item.resourceCategoryId) || findResourceCategoryByConfigId(resourceCategories, item.resourceConfigId)
  );
  const dynamicResourceBindings = placeholders.filter((item) => (
    isResourceBinding(item) && resourceCategoryForPlaceholder(item)?.categoryType === 'dynamic'
  ));
  const resourceBindings = placeholders.filter((item) => (
    isResourceBinding(item) &&
    resourceCategoryForPlaceholder(item)?.categoryType !== 'dynamic' &&
    resourceCategoryForPlaceholder(item)?.categoryType !== 'constant'
  ));
  const constantBindings = placeholders.filter((item) => (
    isResourceBinding(item) && resourceCategoryForPlaceholder(item)?.categoryType === 'constant'
  ));
  const overrideRows = placeholders.filter((item) => !isResourceBinding(item));

  const emitChange = (
    nextBindings: PlaceholderDraft[],
    nextOverrides: PlaceholderDraft[],
    nextDynamicBindings: PlaceholderDraft[] = dynamicResourceBindings,
    nextConstantBindings: PlaceholderDraft[] = constantBindings
  ) => {
    onChange([...nextBindings, ...nextConstantBindings, ...nextDynamicBindings, ...nextOverrides]);
  };

  const updateBinding = (index: number, patch: Partial<PlaceholderDraft>) => {
    emitChange(resourceBindings.map((item, rowIndex) => (
      rowIndex === index ? { ...item, ...patch } : item
    )), overrideRows);
  };

  const updateOverride = (index: number, patch: Partial<PlaceholderDraft>) => {
    emitChange(resourceBindings, overrideRows.map((item, rowIndex) => (
      rowIndex === index ? { ...item, ...patch } : item
    )));
  };

  const removeBinding = (index: number) => {
    const binding = resourceBindings[index];
    const nextBindings = resourceBindings.filter((_, rowIndex) => rowIndex !== index);
    const nextOverrides = overrideRows.filter((row) => Number(row.resourceCategoryId || 0) !== Number(binding?.resourceCategoryId || 0));
    emitChange(nextBindings, nextOverrides);
  };

  const removeOverride = (index: number) => {
    emitChange(resourceBindings, overrideRows.filter((_, rowIndex) => rowIndex !== index));
  };

  const resourceSelectableCategories = resourceCategories.filter((category) => category.categoryType === 'fixed');
  const dynamicCategories = resourceCategories.filter((category) => category.categoryType === 'dynamic');
  const firstDynamicOverride = overrideRows.find((item) => resourceCategoryForPlaceholder(item)?.categoryType === 'dynamic');
  const selectedDynamicPlaceholder = dynamicResourceBindings[0] || firstDynamicOverride;
  const selectedDynamicCategory = selectedDynamicPlaceholder
    ? resourceCategoryForPlaceholder(selectedDynamicPlaceholder)
    : dynamicCategories[0];
  const selectedDynamicConfigId = Number(selectedDynamicPlaceholder?.resourceConfigId || selectedDynamicPlaceholder?.value || 0);
  const selectedDynamicConfig = selectedDynamicCategory?.configs?.find((config) => config.ID === selectedDynamicConfigId) ||
    selectedDynamicCategory?.configs?.[0];
  const selectedDynamicRows = parseResourceRows(selectedDynamicConfig?.rows);
  const visibleDynamicRows = selectedDynamicConfig
    ? dynamicRowsForStep(selectedDynamicConfig, selectedDynamicRows, currentStep)
    : [];
  const resourceGridClass = 'grid grid-cols-[1fr_1.25fr_0.75fr_40px] gap-3';
  const constantGridClass = 'grid grid-cols-[1.1fr_1.35fr_1.7fr] gap-3';
  const overrideGridClass = 'grid grid-cols-[1.35fr_1.15fr_1.25fr_40px] gap-3';

  const constantRows = constantBindings.flatMap((placeholder) => {
    const selectedCategory = resourceCategoryForPlaceholder(placeholder);
    const selectedConfigs = selectedCategory?.configs || [];
    const selectedConfig = selectedConfigs.find((config) => config.ID === Number(placeholder.resourceConfigId || 0)) || selectedConfigs[0];
    return constantRowsForStep(parseResourceRows(selectedConfig?.rows), currentStep).map(({ row, index }) => ({
      row,
      index,
      bindingKey: `${placeholder.resourceConfigId || selectedConfig?.ID || 'constant'}-${index}`,
    }));
  });

  const newResourceBinding = () => {
    const category = resourceSelectableCategories[0];
    const config = category?.configs?.[0];
    return {
      ...emptyPlaceholder,
      resourceCategoryId: Number(category?.ID || 0),
      resourceConfigId: Number(config?.ID || 0),
      value: config?.ID ? String(config.ID) : '',
      customValue: inferResourceRole(category, stepType),
      name: inferPlaceholderName(category, stepType, inferResourceRole(category, stepType)),
      placeholder: category?.categoryName || '',
    };
  };

  const dynamicRowName = (row: ScriptResourceRow) => normalizeEnvName(row.name || row.placeholder);

  const findDynamicOverrideIndex = (row: ScriptResourceRow) => {
    const rowName = dynamicRowName(row);
    const configId = Number(selectedDynamicConfig?.ID || 0);
    return overrideRows.findIndex((placeholder) => {
      const category = resourceCategoryForPlaceholder(placeholder);
      if (category?.categoryType && category.categoryType !== 'dynamic') return false;
      const placeholderConfigId = Number(placeholder.resourceConfigId || 0);
      if (configId && placeholderConfigId && placeholderConfigId !== configId) return false;
      const placeholderName = normalizeEnvName(placeholder.name || placeholder.placeholder);
      return (rowName && placeholderName === rowName) ||
        (!!row.placeholder && placeholder.placeholder === row.placeholder);
    });
  };

  const ensureSelectedDynamicBinding = () => {
    const configId = Number(selectedDynamicConfig?.ID || 0);
    if (!configId) return dynamicResourceBindings;
    const exists = dynamicResourceBindings.some((item) => Number(item.resourceConfigId || item.value || 0) === configId);
    if (exists) return dynamicResourceBindings;
    return [
      ...dynamicResourceBindings,
      {
        ...emptyPlaceholder,
        valueKind: 'resource' as PlaceholderValueKind,
        resourceCategoryId: Number(selectedDynamicCategory?.ID || 0),
        resourceConfigId: configId,
        value: String(configId),
        name: 'EXEC_PARAMS',
        placeholder: selectedDynamicCategory?.categoryName || selectedDynamicConfig?.configName || '动态配置',
      },
    ];
  };

  const dynamicOverrideDraft = (row: ScriptResourceRow, value: string): PlaceholderDraft => ({
    ...emptyOverridePlaceholder,
    resourceCategoryId: Number(selectedDynamicCategory?.ID || 0),
    resourceConfigId: Number(selectedDynamicConfig?.ID || 0),
    placeholder: row.placeholder || row.name,
    name: dynamicRowName(row),
    value,
  });

  const updateDynamicOverride = (row: ScriptResourceRow, value: string) => {
    const overrideIndex = findDynamicOverrideIndex(row);
    if (overrideIndex >= 0) {
      emitChange(
        resourceBindings,
        overrideRows.map((item, rowIndex) => (
          rowIndex === overrideIndex ? dynamicOverrideDraft(row, value) : item
        )),
        ensureSelectedDynamicBinding()
      );
      return;
    }
    emitChange(
      resourceBindings,
      [...overrideRows, dynamicOverrideDraft(row, value)],
      ensureSelectedDynamicBinding()
    );
  };

  const removeDynamicOverride = (row: ScriptResourceRow) => {
    const overrideIndex = findDynamicOverrideIndex(row);
    if (overrideIndex >= 0) {
      removeOverride(overrideIndex);
    }
  };

  const nextDynamicOverrideRow = visibleDynamicRows.find(({ row }) => findDynamicOverrideIndex(row) < 0)?.row;

  const addDynamicOverride = () => {
    if (!nextDynamicOverrideRow) return;
    emitChange(
      resourceBindings,
      [...overrideRows, dynamicOverrideDraft(nextDynamicOverrideRow, nextDynamicOverrideRow.value || '')],
      ensureSelectedDynamicBinding()
    );
  };

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-gray-200 bg-white overflow-hidden">
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <div>
            <h3 className="text-sm font-bold text-gray-800">资源配置</h3>
          </div>
          <button
            type="button"
            onClick={() => emitChange([...resourceBindings, newResourceBinding()], overrideRows)}
            className="inline-flex items-center gap-1 rounded-md bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-700"
          >
            <Plus size={13} />
            添加配置
          </button>
        </div>
        <div className={clsx(resourceGridClass, 'border-b border-gray-100 bg-gray-50 px-3 py-2 text-xs font-bold text-gray-500')}>
          <div>业务分类</div>
          <div>配置</div>
          <div>用途</div>
          <div />
        </div>
        {resourceBindings.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-gray-400">暂无资源配置</div>
        ) : (
          <div className="divide-y divide-gray-100">
            {resourceBindings.map((placeholder, index) => {
              const selectedCategory = resourceCategoryById(placeholder.resourceCategoryId);
              const selectedConfigs = selectedCategory?.categoryType === 'fixed' ? selectedCategory.configs || [] : [];
              return (
                <div key={index} className={clsx(resourceGridClass, 'px-3 py-3')}>
                  <select
                    value={placeholder.resourceCategoryId || 0}
                    onChange={(event) => {
                      const category = resourceCategoryById(Number(event.target.value));
                      const config = category?.configs?.[0];
                      updateBinding(index, {
                        valueKind: 'resource',
                        resourceCategoryId: Number(event.target.value),
                        resourceConfigId: Number(config?.ID || 0),
                        value: config?.ID ? String(config.ID) : '',
                        customValue: category?.categoryType === 'fixed' ? inferResourceRole(category, stepType) : '',
                        name: inferPlaceholderName(category, stepType, category?.categoryType === 'fixed' ? inferResourceRole(category, stepType) : ''),
                        placeholder: category?.categoryName || '',
                      });
                    }}
                    className="rounded-md border border-gray-300 px-2 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5 bg-white"
                  >
                    {resourceSelectableCategories.length === 0 && <option value={0} disabled>暂无资源分类</option>}
                    {resourceSelectableCategories.map((category) => (
                      <option key={category.ID} value={category.ID}>
                        {category.categoryName}
                      </option>
                    ))}
                  </select>
                  <select
                    value={placeholder.resourceConfigId || 0}
                    onChange={(event) => updateBinding(index, {
                      valueKind: 'resource',
                      resourceConfigId: Number(event.target.value),
                      value: event.target.value,
                    })}
                    className="min-w-0 rounded-md border border-gray-300 px-2 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5 bg-white"
                  >
                    <option value={0}>选择配置</option>
                    {selectedConfigs.map((config) => (
                      <option key={config.ID} value={config.ID}>{config.configName}</option>
                    ))}
                  </select>
                  <select
                    value={placeholder.customValue || ''}
                    onChange={(event) => {
                      const category = resourceCategoryForPlaceholder(placeholder);
                      const nextRole = event.target.value;
                      const shouldSyncName = category?.categoryType === 'fixed' && isAutoPlaceholderName(category, placeholder.name);
                      updateBinding(index, {
                        customValue: nextRole,
                        ...(shouldSyncName ? { name: inferPlaceholderName(category, stepType, nextRole) } : {}),
                      });
                    }}
                    className="min-w-0 rounded-md border border-gray-300 px-2 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5 bg-white"
                  >
                    <option value="">默认</option>
                    <option value="source">source</option>
                    <option value="target">target</option>
                  </select>
                  <button
                    type="button"
                    onClick={() => removeBinding(index)}
                    className="flex h-9 w-9 items-center justify-center rounded-md text-gray-400 hover:bg-red-50 hover:text-red-600"
                    title="删除资源配置"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="rounded-lg border border-gray-200 bg-white overflow-hidden">
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h3 className="text-sm font-bold text-gray-800">常量配置</h3>
        </div>
        <div className={clsx(constantGridClass, 'border-b border-gray-100 bg-gray-50 px-3 py-2 text-xs font-bold text-gray-500')}>
          <div>占位符名称</div>
          <div>占位符解释</div>
          <div>占位符值</div>
        </div>
        {constantRows.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-gray-400">暂无常量配置</div>
        ) : (
          <div className="divide-y divide-gray-100">
            {constantRows.map(({ row, bindingKey }) => {
              return (
                <div key={bindingKey} className={clsx(constantGridClass, 'px-3 py-3')}>
                  <input
                    value={normalizeEnvName(row.name || row.placeholder)}
                    readOnly
                    placeholder="占位符名称"
                    className="min-w-0 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-2 text-sm text-gray-600 outline-none"
                  />
                  <input
                    value={row.placeholder || row.name || ''}
                    readOnly
                    placeholder="占位符解释"
                    className="min-w-0 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-2 text-sm text-gray-600 outline-none"
                  />
                  <input
                    value={row.value || ''}
                    readOnly
                    placeholder="占位符值"
                    className="min-w-0 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-2 text-sm text-gray-600 outline-none"
                  />
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="rounded-lg border border-gray-200 bg-white overflow-hidden">
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h3 className="text-sm font-bold text-gray-800">动态配置</h3>
          <button
            type="button"
            onClick={addDynamicOverride}
            disabled={!nextDynamicOverrideRow}
            className={clsx(
              'inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-3 py-1.5 text-xs font-semibold',
              nextDynamicOverrideRow
                ? 'text-gray-700 hover:bg-gray-50'
                : 'cursor-not-allowed text-gray-300'
            )}
          >
            <Plus size={13} />
            添加行
          </button>
        </div>
        <div className={clsx(overrideGridClass, 'border-b border-gray-100 bg-gray-50 px-3 py-2 text-xs font-bold text-gray-500')}>
          <div>占位符解释</div>
          <div>占位符名称</div>
          <div>占位符值</div>
          <div />
        </div>
        {!selectedDynamicConfig ? (
          <div className="px-4 py-8 text-center text-sm text-gray-400">暂无动态配置</div>
        ) : visibleDynamicRows.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-gray-400">当前步骤未引用动态配置</div>
        ) : (
          <div className="divide-y divide-gray-100">
            {visibleDynamicRows.map(({ row, index }) => {
              const overrideIndex = findDynamicOverrideIndex(row);
              const override = overrideIndex >= 0 ? overrideRows[overrideIndex] : undefined;
              const hasOverride = Boolean(override);
              const envName = dynamicRowName(row);
              return (
                <div key={index} className={clsx(overrideGridClass, 'px-3 py-3')}>
                  <input
                    value={row.placeholder || row.name || ''}
                    readOnly
                    placeholder="占位符解释"
                    className="min-w-0 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-2 text-sm text-gray-600 outline-none"
                  />
                  <input
                    value={envName}
                    readOnly
                    placeholder="自动带出"
                    className="min-w-0 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-2 text-sm text-gray-600 outline-none"
                  />
                  <input
                    value={override?.value ?? row.value}
                    onChange={(event) => updateDynamicOverride(row, event.target.value)}
                    placeholder="占位符值"
                    className="min-w-0 rounded-md border border-gray-300 px-2.5 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                  />
                  <button
                    type="button"
                    onClick={() => removeDynamicOverride(row)}
                    disabled={!hasOverride}
                    className={clsx(
                      'flex h-9 w-9 items-center justify-center rounded-md',
                      hasOverride
                        ? 'text-gray-400 hover:bg-red-50 hover:text-red-600'
                        : 'cursor-not-allowed text-gray-300'
                    )}
                    title={hasOverride ? '清除覆盖值' : '当前使用默认值'}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

function ResourceManagerModal({ categories, resourceMode, workflow, onClose, onRefresh }: {
  categories: ScriptResourceCategory[];
  resourceMode: ScriptResourceCategoryType;
  workflow?: ScriptWorkflow;
  onClose: () => void;
  onRefresh: () => void;
}) {
  const scopedCategories = useMemo(() => (
    scopeResourceCategories(categories, resourceMode === 'dynamic' ? workflow?.ID : 0)
  ), [categories, resourceMode, workflow?.ID]);
  const visibleCategories = useMemo(() => (
    scopedCategories.filter((category) => category.categoryType === resourceMode)
  ), [scopedCategories, resourceMode]);
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(
    scopedCategories.find((category) => category.categoryType === resourceMode)?.ID || null
  );
  const [configSearch, setConfigSearch] = useState('');
  const [createConfigDialogOpen, setCreateConfigDialogOpen] = useState(false);
  const [newConfigName, setNewConfigName] = useState('');
  const selectedCategory = visibleCategories.find((category) => category.ID === selectedCategoryId) || visibleCategories[0];
  const workflowSteps = workflow?.steps || [];
  const [selectedDynamicStepId, setSelectedDynamicStepId] = useState<number | null>(
    resourceMode === 'dynamic' ? workflowSteps[0]?.ID || null : null
  );
  const selectedDynamicStep = resourceMode === 'dynamic'
    ? workflowSteps.find((step) => step.ID === selectedDynamicStepId) || workflowSteps[0]
    : undefined;
  const filteredConfigs = useMemo(() => {
    const keyword = configSearch.trim().toLowerCase();
    const configs = selectedCategory?.configs || [];
    if (!keyword) return configs;
    return configs.filter((config) => {
      const rows = parseResourceRows(config.rows);
      return config.configName.toLowerCase().includes(keyword) ||
        rows.some((row) =>
          row.name.toLowerCase().includes(keyword) ||
          row.placeholder.toLowerCase().includes(keyword) ||
          row.value.toLowerCase().includes(keyword)
        );
    });
  }, [selectedCategory, configSearch]);

  useEffect(() => {
    if (selectedCategory && selectedCategory.ID !== selectedCategoryId) {
      setSelectedCategoryId(selectedCategory.ID);
      return;
    }
    if (!selectedCategory && selectedCategoryId !== null) {
      setSelectedCategoryId(null);
    }
  }, [selectedCategory, selectedCategoryId]);

  useEffect(() => {
    setConfigSearch('');
  }, [resourceMode, selectedCategoryId]);

  useEffect(() => {
    if (resourceMode !== 'dynamic') return;
    const currentExists = workflowSteps.some((step) => step.ID === selectedDynamicStepId);
    if (!currentExists) {
      setSelectedDynamicStepId(workflowSteps[0]?.ID || null);
    }
  }, [resourceMode, workflow?.ID, workflowSteps.length, selectedDynamicStepId]);

  const createCategory = async () => {
    const categoryType = resourceMode;
    if (categoryType === 'dynamic' && !workflow?.ID) {
      toast.error('请选择脚本流程后再配置动态参数');
      return;
    }
    const name = window.prompt(
      categoryType === 'dynamic'
        ? '动态资源分类名称'
        : categoryType === 'constant'
          ? '常量配置分类名称'
          : '资源配置分类名称'
    );
    if (!name?.trim()) return;
    try {
      const res = await createScriptResourceCategory({ categoryName: name.trim(), categoryType });
      if (res.code === 0) {
        if ((categoryType === 'dynamic' || categoryType === 'constant') && res.data?.ID) {
          await createScriptResourceConfig({
            categoryId: res.data.ID,
            workflowId: categoryType === 'dynamic' ? workflow?.ID || 0 : 0,
            configName: name.trim(),
            rows: JSON.stringify([{ ...emptyResourceRow }]),
          });
        }
        toast.success('资源分类已创建');
        setSelectedCategoryId(res.data?.ID || null);
        onRefresh();
      }
    } catch {
      toast.error('资源分类创建失败');
    }
  };

  const renameCategory = async (category: ScriptResourceCategory) => {
    const name = window.prompt('资源分类名称', category.categoryName);
    if (!name?.trim()) return;
    try {
      const res = await updateScriptResourceCategory(category.ID, { ...category, categoryName: name.trim() });
      if (res.code === 0) {
        toast.success('资源分类已保存');
        onRefresh();
      }
    } catch {
      toast.error('资源分类保存失败');
    }
  };

  const removeCategory = async (category: ScriptResourceCategory) => {
    const categoryLabel = category.categoryType === 'constant'
      ? '常量配置分类'
      : category.categoryType === 'dynamic'
        ? '动态配置分类'
        : '资源配置分类';
    if (!window.confirm(`确定删除${categoryLabel}「${category.categoryName}」吗？`)) return;
    if (!window.confirm(`删除后「${category.categoryName}」下的配置行会一并删除，且不可恢复。确认继续删除吗？`)) return;
    try {
      const res = await deleteScriptResourceCategory(category.ID);
      if (res.code === 0) {
        toast.success('资源分类已删除');
        setSelectedCategoryId(null);
        onRefresh();
      }
    } catch {
      toast.error('资源分类删除失败');
    }
  };

  const createConfig = async () => {
    if (!selectedCategory) return;
    if (selectedCategory.categoryType === 'constant') {
      toast.error('常量配置只需要添加行');
      return;
    }
    if (selectedCategory.categoryType === 'dynamic' && !workflow?.ID) {
      toast.error('请选择脚本流程后再配置动态参数');
      return;
    }
    if (selectedCategory.categoryType === 'fixed') {
      setNewConfigName('');
      setCreateConfigDialogOpen(true);
      return;
    }
    const configName = window.prompt('配置名称')?.trim() || '';
    if (!configName) return;
    try {
      const res = await createScriptResourceConfig({
        categoryId: selectedCategory.ID,
        workflowId: selectedCategory.categoryType === 'dynamic' ? workflow?.ID || 0 : 0,
        configName,
        rows: serializeResourceRows([{ ...emptyResourceRow }]),
      });
      if (res.code === 0) {
        toast.success('配置已创建');
        onRefresh();
      }
    } catch {
      toast.error('配置创建失败');
    }
  };

  const closeCreateConfigDialog = () => {
    setCreateConfigDialogOpen(false);
    setNewConfigName('');
  };

  const submitFixedConfig = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedCategory || selectedCategory.categoryType !== 'fixed') return;
    const configName = newConfigName.trim();
    if (!configName) {
      toast.error('请填写名称');
      return;
    }
    try {
      const res = await createScriptResourceConfig({
        categoryId: selectedCategory.ID,
        workflowId: 0,
        configName,
        rows: serializeResourceRows([{ ...emptyResourceRow }]),
      });
      if (res.code === 0) {
        toast.success('配置已创建');
        closeCreateConfigDialog();
        onRefresh();
      }
    } catch {
      toast.error('配置创建失败');
    }
  };

  const addDynamicRow = async () => {
    if (!selectedCategory) return;
    if (!workflow?.ID) {
      toast.error('请选择脚本流程后再配置动态参数');
      return;
    }
    const config = selectedCategory.configs?.[0];
    try {
      if (config) {
        const rows = parseResourceRows(config.rows);
        const res = await updateScriptResourceConfig(config.ID, {
          ...config,
          rows: serializeResourceRows([...rows, { ...emptyResourceRow }]),
        });
        if (res.code === 0) {
          toast.success('配置行已添加');
          onRefresh();
        }
        return;
      }
      const res = await createScriptResourceConfig({
        categoryId: selectedCategory.ID,
        workflowId: workflow.ID,
        configName: selectedCategory.categoryName,
        rows: JSON.stringify([{ ...emptyResourceRow }]),
      });
      if (res.code === 0) {
        toast.success('配置行已添加');
        onRefresh();
      }
    } catch {
      toast.error('配置行添加失败');
    }
  };

  const addConstantRow = async () => {
    if (!selectedCategory) return;
    const config = selectedCategory.configs?.[0];
    try {
      if (config) {
        const rows = parseResourceRows(config.rows);
        const res = await updateScriptResourceConfig(config.ID, {
          ...config,
          rows: serializeResourceRows([...rows, { ...emptyResourceRow }]),
        });
        if (res.code === 0) {
          toast.success('配置行已添加');
          onRefresh();
        }
        return;
      }
      const res = await createScriptResourceConfig({
        categoryId: selectedCategory.ID,
        workflowId: 0,
        configName: selectedCategory.categoryName,
        rows: JSON.stringify([{ ...emptyResourceRow }]),
      });
      if (res.code === 0) {
        toast.success('配置行已添加');
        onRefresh();
      }
    } catch {
      toast.error('配置行添加失败');
    }
  };

  const modeTitle = resourceMode === 'fixed' ? '资源配置' : resourceMode === 'constant' ? '常量配置' : '动态配置';
  const modalTitle = resourceMode === 'dynamic' && workflow?.workflowName
    ? `${modeTitle} · ${workflow.workflowName}`
    : modeTitle;
  const dynamicRowCount = selectedCategory?.categoryType === 'dynamic'
    ? filteredConfigs.slice(0, 1).reduce((count, config) => {
      const rows = parseResourceRows(config.rows);
      return count + dynamicRowsForStep(config, rows, selectedDynamicStep).length;
    }, 0)
    : 0;
  const showCategorySidebar = resourceMode === 'fixed' || resourceMode === 'constant';
  const showStepSidebar = resourceMode === 'dynamic';

  return (
    <div className="fixed inset-0 z-[120] bg-white flex flex-col">
      <div className="h-16 shrink-0 border-b border-gray-200 px-6 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-md bg-gray-900 text-white"><Settings2 size={16} /></div>
          <h2 className="text-lg font-bold text-gray-900">{modalTitle}</h2>
        </div>
        <button type="button" onClick={onClose} className="p-2 rounded-full text-gray-400 hover:text-gray-900 hover:bg-gray-100">
          <X size={20} />
        </button>
      </div>
      <div className="flex-1 min-h-0 flex">
        {showStepSidebar && (
          <aside className="w-72 shrink-0 border-r border-gray-100 bg-gray-50 p-4 flex flex-col">
            <div className="mb-3">
              <span className="text-[11px] font-bold text-gray-400 uppercase tracking-wider">执行步骤</span>
            </div>
            <div className="flex-1 overflow-y-auto space-y-1">
              {workflowSteps.length === 0 ? (
                <div className="rounded-lg border border-dashed border-gray-300 bg-white p-4 text-center text-xs text-gray-400">暂无执行步骤</div>
              ) : workflowSteps.map((step) => {
                const active = selectedDynamicStep?.ID === step.ID;
                const meta = STEP_TYPE_META[step.stepType] || STEP_TYPE_META.local_exec;
                const Icon = meta.icon;
                return (
                  <button
                    key={step.ID}
                    type="button"
                    onClick={() => setSelectedDynamicStepId(step.ID)}
                    className={clsx(
                      'w-full rounded-lg px-3 py-2 text-left transition',
                      active ? 'bg-gray-900 text-white shadow-sm' : 'bg-white text-gray-700 hover:bg-gray-100'
                    )}
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <Icon size={14} className={active ? 'text-white' : 'text-gray-400'} />
                      <span className="shrink-0 text-[11px] font-bold text-current/70">{meta.label}</span>
                    </div>
                    <div className="mt-1 truncate text-sm font-semibold">{step.stepName}</div>
                  </button>
                );
              })}
            </div>
          </aside>
        )}
        {showCategorySidebar && (
          <aside className="w-64 shrink-0 border-r border-gray-100 bg-gray-50 p-4 flex flex-col">
            <div className="flex items-center justify-between mb-3">
              <span className="text-[11px] font-bold text-gray-400 uppercase tracking-wider">业务分类</span>
              <button
                type="button"
                onClick={createCategory}
                className="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-2 py-1 text-[11px] font-bold text-gray-600 hover:bg-gray-100"
              >
                <Plus size={12} />
                新增
              </button>
            </div>
            <div className="flex-1 overflow-y-auto space-y-1">
              {visibleCategories.length === 0 ? (
                <div className="rounded-lg border border-dashed border-gray-300 bg-white p-4 text-center text-xs text-gray-400">暂无资源分类</div>
              ) : visibleCategories.map((category) => {
                const active = selectedCategory?.ID === category.ID;
                return (
                  <div key={category.ID} className="group/category relative">
                    <button
                      type="button"
                      onClick={() => setSelectedCategoryId(category.ID)}
                      className={clsx(
                        'w-full flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-semibold transition',
                        active ? 'bg-gray-900 text-white' : 'bg-white text-gray-700 hover:bg-gray-100'
                      )}
                    >
                      <Folder size={14} className={active ? 'text-white' : 'text-gray-400'} />
                      <span className="min-w-0 flex-1 truncate">{category.categoryName}</span>
                    </button>
                    <div className="absolute right-2 top-1/2 hidden -translate-y-1/2 items-center gap-1 group-hover/category:flex">
                      <button type="button" onClick={(event) => { event.stopPropagation(); renameCategory(category); }} className="rounded p-1 text-gray-400 hover:bg-white hover:text-gray-700"><Pencil size={12} /></button>
                      <button type="button" onClick={(event) => { event.stopPropagation(); removeCategory(category); }} className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 size={12} /></button>
                    </div>
                  </div>
                );
              })}
            </div>
          </aside>
        )}
        <main className="flex-1 min-w-0 overflow-y-auto bg-white p-6">
          {!selectedCategory ? (
            <div className="h-full min-h-[360px] flex items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 text-sm text-gray-400">请选择或新增资源分类</div>
          ) : (
            <div className={clsx('mx-auto', resourceMode === 'dynamic' ? 'max-w-7xl' : 'max-w-6xl')}>
              <div className="mb-5 flex items-center justify-between gap-4">
                <div>
                  <h3 className="text-xl font-bold text-gray-900">{selectedCategory.categoryName}</h3>
                  <p className="mt-1 text-xs text-gray-400">
                    {selectedCategory.categoryType === 'fixed'
                      ? '资源配置在步骤里只选择，不填写覆盖值'
                      : selectedCategory.categoryType === 'constant'
                        ? '常量配置按分类维护一组占位符值'
                        : selectedDynamicStep
                          ? `当前步骤：${selectedDynamicStep.stepName} · 引用 ${dynamicRowCount} 项动态配置`
                          : '动态配置只对当前脚本流程生效'}
                  </p>
                </div>
                {selectedCategory.categoryType === 'fixed' ? (
                  <button type="button" onClick={createConfig} className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700">
                    <Plus size={15} />
                    添加配置
                  </button>
                ) : selectedCategory.categoryType === 'constant' && (selectedCategory.configs || []).length === 0 ? (
                  <button type="button" onClick={addConstantRow} className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700">
                    <Plus size={15} />
                    添加行
                  </button>
                ) : (selectedCategory.configs || []).length === 0 ? (
                  <button type="button" onClick={addDynamicRow} className="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700">
                    <Plus size={15} />
                    添加行
                  </button>
                ) : null}
              </div>
              <div className="mb-4 flex items-center justify-between gap-3">
                <div className="relative w-full max-w-md">
                  <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    value={configSearch}
                    onChange={(event) => setConfigSearch(event.target.value)}
                    placeholder="筛选配置名称、占位符或值"
                    className="w-full rounded-lg border border-gray-300 bg-white py-2.5 pl-9 pr-3 text-sm outline-none focus:ring-2 focus:ring-black/5"
                  />
                </div>
                <span className="shrink-0 text-xs text-gray-400">
                  {selectedCategory.categoryType === 'dynamic'
                    ? `${dynamicRowCount} 行`
                    : selectedCategory.categoryType === 'constant'
                      ? `${filteredConfigs.slice(0, 1).reduce((count, config) => count + parseResourceRows(config.rows).length, 0)} 行`
                      : `${filteredConfigs.length} / ${(selectedCategory.configs || []).length}`}
                </span>
              </div>
              <div className="space-y-4">
                {(selectedCategory.configs || []).length === 0 ? (
                  <div className="rounded-lg border border-dashed border-gray-300 bg-gray-50 p-12 text-center text-sm text-gray-400">
                    {selectedCategory.categoryType === 'fixed' ? '暂无配置卡片' : '暂无配置行'}
                  </div>
                ) : filteredConfigs.length === 0 ? (
                  <div className="rounded-lg border border-dashed border-gray-300 bg-gray-50 p-12 text-center text-sm text-gray-400">没有匹配的配置</div>
                ) : (selectedCategory.categoryType === 'dynamic' || selectedCategory.categoryType === 'constant' ? filteredConfigs.slice(0, 1) : filteredConfigs).map((config) => (
                  <ResourceConfigCard
                    key={config.ID}
                    categoryType={selectedCategory.categoryType}
                    config={config}
                    workflow={workflow}
                    focusedStep={selectedDynamicStep}
                    onRefresh={onRefresh}
                  />
                ))}
              </div>
            </div>
          )}
	        </main>
	      </div>
	      {createConfigDialogOpen && selectedCategory?.categoryType === 'fixed' && (
	        <div className="fixed inset-0 z-[140] flex items-center justify-center bg-black/45 px-4">
	          <form
	            onSubmit={submitFixedConfig}
	            className="w-full max-w-md rounded-lg border border-gray-200 bg-white shadow-xl"
	          >
	            <div className="flex items-center justify-between border-b border-gray-100 px-5 py-4">
	              <h3 className="text-base font-bold text-gray-900">新增配置</h3>
	              <button
	                type="button"
	                onClick={closeCreateConfigDialog}
	                className="rounded-full p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
	              >
	                <X size={16} />
	              </button>
	            </div>
	            <div className="space-y-4 px-5 py-4">
	              <label className="block">
	                <span className="mb-1.5 block text-xs font-bold text-gray-500">名称</span>
	                <input
	                  autoFocus
	                  value={newConfigName}
	                  onChange={(event) => setNewConfigName(event.target.value)}
	                  placeholder="中铁开发环境 / 192.168.0.141"
	                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
	                />
	              </label>
	            </div>
	            <div className="flex justify-end gap-2 border-t border-gray-100 px-5 py-4">
	              <button
	                type="button"
	                onClick={closeCreateConfigDialog}
	                className="rounded-md border border-gray-200 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
	              >
	                取消
	              </button>
	              <button
	                type="submit"
	                className="inline-flex items-center gap-1 rounded-md bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700"
	              >
	                <Check size={14} />
	                确定
	              </button>
	            </div>
	          </form>
	        </div>
	      )}
	    </div>
	  );
	}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function scriptReferencesEnvName(script: string | undefined, envName: string) {
  const normalizedName = normalizeEnvName(envName);
  if (!script || !normalizedName) return false;
  return new RegExp(`(^|[^A-Z0-9_])${escapeRegExp(normalizedName)}([^A-Z0-9_]|$)`).test(script.toUpperCase());
}

function stepPlaceholderRefs(raw?: string): Array<{ name: string; resourceConfigId: number }> {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.map((item) => ({
      name: normalizeEnvName(String(item.name || item.placeholder || '')),
      resourceConfigId: Number(item.resourceConfigId || 0),
    }));
  } catch {
    return [];
  }
}

function stepUsesDynamicEnvName(step: ScriptStep | undefined, config: ScriptResourceConfig, envName: string) {
  const normalizedName = normalizeEnvName(envName);
  if (!step || !normalizedName) return false;
  const refs = stepPlaceholderRefs(step.placeholders);
  const overrideReference = refs.some((ref) => (
    ref.resourceConfigId === config.ID && ref.name === normalizedName
  ));
  return overrideReference || scriptReferencesEnvName(step.scriptContent, normalizedName);
}

function dynamicRowsForStep(config: ScriptResourceConfig, rows: ScriptResourceRow[], step?: ScriptStep) {
  if (!step) {
    return rows.map((row, index) => ({ row, index }));
  }
  return rows
    .map((row, index) => ({ row, index }))
    .filter(({ row }) => {
      const envName = normalizeEnvName(row.name || row.placeholder);
      return !envName || stepUsesDynamicEnvName(step, config, envName);
    });
}

function constantRowsForStep(rows: ScriptResourceRow[], step?: ScriptStep) {
  if (!step) {
    return rows.map((row, index) => ({ row, index }));
  }
  return rows
    .map((row, index) => ({ row, index }))
    .filter(({ row }) => {
      const envName = normalizeEnvName(row.name || row.placeholder);
      return !!envName && scriptReferencesEnvName(step.scriptContent, envName);
    });
}

function dynamicRowUsages(workflow: ScriptWorkflow | undefined, config: ScriptResourceConfig, rows: ScriptResourceRow[]) {
  const usage: Record<string, string[]> = {};
  if (!workflow?.steps?.length) return usage;
  rows.forEach((row) => {
    const envName = normalizeEnvName(row.name || row.placeholder);
    if (!envName) return;
    const stepNames = workflow.steps || [];
    usage[envName] = stepNames
      .filter((step) => stepUsesDynamicEnvName(step, config, envName))
      .map((step) => step.stepName);
  });
  return usage;
}

function ResourceConfigCard({ categoryType, config, workflow, focusedStep, onRefresh }: {
  categoryType: ScriptResourceCategoryType;
  config: ScriptResourceConfig;
  workflow?: ScriptWorkflow;
  focusedStep?: ScriptStep;
  onRefresh: () => void;
}) {
  const [configName, setConfigName] = useState(config.configName || '');
  const [rows, setRows] = useState<ScriptResourceRow[]>(parseResourceRows(config.rows));
  const rowUsageByName = useMemo(() => (
    categoryType === 'dynamic' ? dynamicRowUsages(workflow, config, rows) : {}
  ), [categoryType, workflow, config, rows]);
  const visibleRows = useMemo(() => (
    categoryType === 'dynamic' ? dynamicRowsForStep(config, rows, focusedStep) : rows.map((row, index) => ({ row, index }))
  ), [categoryType, config, rows, focusedStep]);
  const gridClass = categoryType === 'dynamic'
    ? 'grid grid-cols-[1fr_1.1fr_1.25fr_1.6fr_40px] gap-3'
    : 'grid grid-cols-[1fr_1.2fr_1.5fr_40px] gap-3';

  useEffect(() => {
    setConfigName(config.configName || '');
    setRows(parseResourceRows(config.rows));
  }, [config]);

  const updateRow = (index: number, patch: Partial<ScriptResourceRow>) => {
    setRows(rows.map((row, rowIndex) => rowIndex === index ? { ...row, ...patch } : row));
  };

  const removeRow = (index: number) => {
    const row = rows[index];
    const rowName = row?.name || row?.placeholder || `第 ${index + 1} 行`;
    if (!window.confirm(`确定删除配置行「${rowName}」吗？`)) return;
    setRows(rows.filter((_, rowIndex) => rowIndex !== index));
  };

  const saveConfig = async () => {
    const nextConfigName = configName.trim() || (categoryType === 'dynamic' ? '动态占位符' : categoryType === 'constant' ? '常量配置' : '');
    if (!nextConfigName) {
      toast.error('请填写配置名称');
      return;
    }
    try {
      const res = await updateScriptResourceConfig(config.ID, {
        ...config,
        configName: nextConfigName,
        rows: serializeResourceRows(rows),
      });
      if (res.code === 0) {
        toast.success('配置已保存');
        onRefresh();
      }
    } catch {
      toast.error('配置保存失败');
    }
  };

  const removeConfig = async () => {
    if (!window.confirm(`确定删除配置「${config.configName}」吗？`)) return;
    try {
      const res = await deleteScriptResourceConfig(config.ID);
      if (res.code === 0) {
        toast.success('配置已删除');
        onRefresh();
      }
    } catch {
      toast.error('配置删除失败');
    }
  };

  return (
    <div className="rounded-lg border border-gray-200 bg-white shadow-sm overflow-hidden">
      <div className="flex items-center justify-between gap-4 border-b border-gray-100 px-4 py-3">
        {categoryType === 'fixed' ? (
          <input
            value={configName}
            onChange={(event) => setConfigName(event.target.value)}
            className="min-w-0 flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm font-semibold outline-none focus:ring-2 focus:ring-black/5"
            placeholder="配置名称"
          />
        ) : categoryType === 'constant' ? (
          <div className="min-w-0 flex-1 text-sm font-bold text-gray-800">常量占位符</div>
        ) : (
          <div className="min-w-0 flex-1 text-sm font-bold text-gray-800">动态占位符</div>
        )}
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setRows([...rows, { ...emptyResourceRow }])} className="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 hover:bg-gray-50">
            <Plus size={13} /> 添加行
          </button>
          <button type="button" onClick={saveConfig} className="inline-flex items-center gap-1 rounded-md bg-gray-900 px-3 py-2 text-xs font-semibold text-white hover:bg-gray-700">
            <Check size={13} /> 保存
          </button>
          {categoryType === 'fixed' && (
            <button type="button" onClick={removeConfig} className="rounded-md border border-red-100 bg-red-50 p-2 text-red-600 hover:bg-red-100">
              <Trash2 size={14} />
            </button>
          )}
        </div>
      </div>
      <div className={clsx(gridClass, 'border-b border-gray-100 bg-gray-50 px-4 py-2 text-xs font-bold text-gray-500')}>
        <div>占位符名称</div>
        <div>占位符解释</div>
        <div>{categoryType === 'dynamic' ? '占位符默认值' : '占位符值'}</div>
        {categoryType === 'dynamic' && <div>{focusedStep ? '其他引用步骤' : '引用步骤'}</div>}
        <div />
      </div>
      {rows.length === 0 ? (
        <div className="px-4 py-8 text-center text-sm text-gray-400">暂无配置行</div>
      ) : visibleRows.length === 0 ? (
        <div className="px-4 py-8 text-center text-sm text-gray-400">当前步骤未引用动态配置</div>
      ) : (
        <div className="divide-y divide-gray-100">
          {visibleRows.map(({ row, index }) => {
            const usage = rowUsageByName[normalizeEnvName(row.name || row.placeholder)] || [];
            const visibleUsage = focusedStep
              ? usage.filter((stepName) => stepName !== focusedStep.stepName)
              : usage;
            return (
              <div key={index} className={clsx(gridClass, 'px-4 py-3')}>
                <input
                  value={row.name}
                  onChange={(event) => updateRow(index, { name: event.target.value })}
                  placeholder="TARGET_SERVER_IP"
                  className="min-w-0 rounded-md border border-gray-300 px-2.5 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                />
                <input
                  value={row.placeholder}
                  onChange={(event) => updateRow(index, { placeholder: event.target.value })}
                  placeholder="目标服务器 IP"
                  className="min-w-0 rounded-md border border-gray-300 px-2.5 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                />
                <input
                  value={row.value}
                  onChange={(event) => updateRow(index, { value: event.target.value })}
                  placeholder={categoryType === 'dynamic' ? '默认值' : '配置值'}
                  className="min-w-0 rounded-md border border-gray-300 px-2.5 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                />
                {categoryType === 'dynamic' && (
                  <div className="min-w-0 flex flex-wrap items-center gap-1">
                    {visibleUsage.length === 0 ? (
                      <span className="text-xs text-gray-400">
                        {focusedStep ? '仅当前步骤' : '未被步骤脚本引用'}
                      </span>
                    ) : visibleUsage.map((stepName) => (
                      <span
                        key={stepName}
                        className="max-w-full truncate rounded-full bg-blue-50 px-2 py-1 text-xs font-semibold text-blue-700"
                        title={stepName}
                      >
                        {stepName}
                      </span>
                    ))}
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => removeRow(index)}
                  className="flex h-9 w-9 items-center justify-center rounded-md text-gray-400 hover:bg-red-50 hover:text-red-600"
                  title="删除配置行"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function TextArea({ label, value, onChange, rows = 3 }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  rows?: number;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        rows={rows}
        className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 resize-none"
      />
    </div>
  );
}

function LogModal({ title, executions, active, onSelect, onClose }: {
  title: string;
  executions: ScriptExecution[];
  active?: ScriptExecution;
  onSelect: (execution: ScriptExecution) => void;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-[180] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-gray-900 rounded-lg shadow-2xl w-[960px] max-w-[92vw] max-h-[82vh] overflow-hidden flex flex-col border border-gray-700 animate-in zoom-in-95 duration-300">
        <div className="flex items-center justify-between px-5 py-3 border-b border-gray-700/50">
          <div className="flex items-center gap-3">
            <div className="p-1.5 bg-emerald-500/20 text-emerald-300 rounded-lg"><ScrollText size={16} /></div>
            <div>
              <h3 className="text-sm font-bold text-white leading-tight">执行日志</h3>
              <p className="text-xs text-gray-400">{title}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"><X size={16} /></button>
        </div>
        <div className="grid grid-cols-[260px_minmax(0,1fr)] h-[min(70vh,560px)] min-h-0 overflow-hidden">
          <div className="min-h-0 border-r border-gray-700/50 overflow-y-auto p-3">
            {executions.length === 0 ? (
              <div className="text-sm text-gray-500 text-center mt-10">暂无日志</div>
            ) : executions.map((execution) => (
              <button
                key={execution.ID}
                type="button"
                onClick={() => onSelect(execution)}
                className={clsx(
                  'w-full text-left rounded-lg px-3 py-2 mb-1 transition-colors',
                  active?.ID === execution.ID ? 'bg-gray-700 text-white' : 'text-gray-400 hover:bg-gray-800 hover:text-white'
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-mono">#{execution.ID}</span>
                  <span className={clsx('text-[11px]', execution.status === 'success' ? 'text-green-400' : execution.status === 'failed' ? 'text-red-400' : 'text-blue-400')}>{execution.status}</span>
                </div>
                <div className="mt-1 text-[11px] text-gray-500 truncate">{execution.startedAt || '-'}</div>
              </button>
            ))}
          </div>
          <div className="min-h-0 min-w-0 overflow-auto px-5 py-4 font-mono text-xs leading-relaxed">
            {active ? (
              (active.logText || '').split('\n').map((line, index) => (
                <div key={`${active.ID}-${index}`} className="py-0.5">
                  <span className="text-gray-500 select-none mr-3">{String(index + 1).padStart(3, ' ')}</span>
                  <span className={line.includes('失败') || line.includes('error') ? 'text-red-300' : line.includes('完成') ? 'text-green-300' : 'text-gray-300'}>{line}</span>
                </div>
              ))
            ) : (
              <div className="h-full flex items-center justify-center text-gray-500">暂无日志</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ScriptExecutePanel({ target, id, title, onClose }: {
  target: 'workflow' | 'step';
  id: number;
  title: string;
  onClose: () => void;
}) {
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState<'connecting' | 'running' | 'success' | 'error'>('connecting');

  useEffect(() => {
    const token = useUserStore.getState().token;
    const isWails = !!(window as any).__wails__ ||
      window.location.protocol === 'wails:' ||
      window.location.hostname === 'wails.localhost';
    const sseBaseUrl = isWails ? 'http://127.0.0.1:48009' : (import.meta.env.VITE_BASE_API || '/api');
    const path = target === 'workflow'
      ? `/script-manager/workflows/${id}/executeStream`
      : `/script-manager/steps/${id}/executeStream`;
    const es = new EventSource(`${sseBaseUrl}${path}?token=${encodeURIComponent(token || '')}`);

    es.onopen = () => {
      setStatus('running');
      setLogs((prev) => [...prev, `已连接，开始执行: ${title}`]);
    };
    es.addEventListener('log', (event: MessageEvent) => {
      setLogs((prev) => [...prev, event.data]);
    });
    es.addEventListener('done', (event: MessageEvent) => {
      setLogs((prev) => [...prev, event.data]);
      setStatus('success');
      es.close();
    });
    es.addEventListener('error', (event: MessageEvent) => {
      if (event.data) setLogs((prev) => [...prev, event.data]);
      setStatus('error');
      es.close();
    });
    es.onerror = () => {
      if (es.readyState !== EventSource.CLOSED) {
        setLogs((prev) => [...prev, '连接中断']);
        setStatus('error');
        es.close();
      }
    };
    return () => es.close();
  }, [target, id, title]);

  const running = status === 'connecting' || status === 'running';

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-gray-900 rounded-lg shadow-2xl w-[820px] max-w-[90vw] max-h-[80vh] flex flex-col border border-gray-700 animate-in zoom-in-95 duration-300">
        <div className="flex items-center justify-between px-5 py-3 border-b border-gray-700/50">
          <div className="flex items-center gap-3">
            <div className="p-1.5 bg-blue-500/20 text-blue-300 rounded-lg"><Terminal size={16} /></div>
            <div>
              <h3 className="text-sm font-bold text-white leading-tight">执行输出</h3>
              <p className="text-xs text-gray-400">{title}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <span className={clsx('flex items-center gap-1.5 text-xs font-medium', status === 'success' ? 'text-green-400' : status === 'error' ? 'text-red-400' : 'text-blue-400')}>
              {running ? <Loader2 size={14} className="animate-spin" /> : null}
              {status === 'success' ? '执行成功' : status === 'error' ? '执行失败' : '执行中'}
            </span>
            <button onClick={onClose} disabled={running} className={clsx('p-1.5 rounded-lg transition-colors', running ? 'text-gray-600 cursor-not-allowed' : 'text-gray-400 hover:text-white hover:bg-gray-700')}>
              <X size={16} />
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto px-5 py-4 font-mono text-xs leading-relaxed min-h-[320px] max-h-[60vh]">
          {logs.map((line, index) => (
            <div key={index} className="py-0.5">
              <span className="text-gray-500 select-none mr-3">{String(index + 1).padStart(3, ' ')}</span>
              <span className={line.includes('失败') || line.includes('error') ? 'text-red-300' : line.includes('完成') ? 'text-green-300' : 'text-gray-300'}>{line}</span>
            </div>
          ))}
        </div>
        <div className="border-t border-gray-700/50 px-5 py-3 flex items-center justify-between">
          <span className="text-xs text-gray-500">{logs.length} 行日志</span>
          <button
            type="button"
            onClick={onClose}
            disabled={running}
            className={clsx('px-4 py-1.5 text-xs font-medium rounded-lg transition-colors', running ? 'bg-gray-700 text-gray-400 cursor-not-allowed' : 'bg-white text-gray-900 hover:bg-gray-200')}
          >
            关闭
          </button>
        </div>
      </div>
    </div>
  );
}
