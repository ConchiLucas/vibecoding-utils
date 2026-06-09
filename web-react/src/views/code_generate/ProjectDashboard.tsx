import React, { useState, useEffect, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, Copy, ClipboardCopy, Database, FileCode, Edit2, Trash2, Search, Folder, Check, X, Wand2, RefreshCw } from 'lucide-react';
import { getProjectList, createProject, updateProject, deleteProject, copyProject, generateProjectCode, getProjectInstanceList } from '@/api/code_generate_project';
import { getDbTemplateScripts, getDbTemplateTypes } from '@/api/db_template';
import { getPathList } from '@/api/path_model';
import toast from 'react-hot-toast';
import ProjectConfigDialog from './ProjectConfigDialog';
import DbTemplateLibrary from './DbTemplateLibrary';
import {
  buildDbTemplateSqlCopyText,
  buildDbTemplateSqlSection,
  mergeDbTemplatePlaceholders,
  type DbTemplatePlaceholder,
  type DbTemplatePlaceholderValues,
} from './dbTemplateCopy';
import {
  getProjectTypeLabel,
  matchesProjectCardSearch,
  normalizeProjectType,
  PROJECT_TYPE_BACKEND,
  PROJECT_TYPE_FRONTEND,
  shouldShowDbTemplateActions,
} from './projectDashboardActions';

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

const normalizeProjectRows = (value: any) => {
  if (Array.isArray(value)) return value;
  if (value && Array.isArray(value.list)) return value.list;
  return [];
};

const DEFAULT_BUSINESS_TYPE = '未分类';

const getProjectBusinessType = (project: any) => {
  const typeName = String(project?.businessType || '').trim();
  return typeName || DEFAULT_BUSINESS_TYPE;
};

const copyTextToClipboard = async (text: string) => {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '-9999px';
  textarea.setAttribute('readonly', 'readonly');
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand('copy');
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error('copy failed');
  }
};

const buildDbTemplateSqlPayload = async (project: any, placeholderValues: DbTemplatePlaceholderValues = {}) => {
  const projectId = Number(project.ID || 0);
  const typeRes: any = await getDbTemplateTypes(projectId);
  const types = Array.isArray(unwrapResponseData(typeRes)) ? unwrapResponseData(typeRes) : [];
  const sections: string[] = [];

  for (const typeObj of types) {
    const scriptRes: any = await getDbTemplateScripts(projectId, Number(typeObj.ID));
    const scripts = Array.isArray(unwrapResponseData(scriptRes)) ? unwrapResponseData(scriptRes) : [];

    scripts
      .filter((script: any) => String(script.content || '').trim())
      .forEach((script: any) => {
        sections.push(buildDbTemplateSqlSection(project, typeObj, script, placeholderValues));
      });
  }

  return {
    sections,
    placeholders: mergeDbTemplatePlaceholders(types),
  };
};

const parseGeneratePathIds = (value: string) => {
  return String(value || '')
    .split(',')
    .map((item) => Number(item.trim()))
    .filter((id) => id > 0);
};

const resolveGeneratePathFilter = (instance: any) => {
  const identity = String(instance?.selectedPathSetIdentity || '').trim();
  if (identity.startsWith('path-set-0-copy-')) {
    return {
      pathSet: 0,
      pathIds: parseGeneratePathIds(identity.replace('path-set-0-copy-', '')),
    };
  }
  if (identity.startsWith('path-set-')) {
    const pathSet = Number(identity.replace('path-set-', ''));
    if (!Number.isNaN(pathSet)) {
      return { pathSet, pathIds: [] };
    }
  }
  return { pathSet: 0, pathIds: [] };
};

const buildGenerateCodePlaceholderPayload = async (project: any) => {
  const templateProjectId = Number(project?.ID || 0);
  if (!templateProjectId) return { projectInstanceId: 0, pathSet: 0, pathIds: [] as number[], placeholders: [] as DbTemplatePlaceholder[] };

  const instanceRes: any = await getProjectInstanceList(templateProjectId, true);
  const instances = normalizeProjectRows(unwrapResponseData(instanceRes));
  const selectedInstanceId = Number(project?.selectedProjectInstanceId || 0);
  const instance = instances.find((item: any) => Number(item.ID || 0) === selectedInstanceId) || instances[0] || null;
  const projectInstanceId = Number(instance?.ID || 0);
  if (!projectInstanceId) return { projectInstanceId: 0, pathSet: 0, pathIds: [] as number[], placeholders: [] as DbTemplatePlaceholder[] };

  const { pathSet, pathIds } = resolveGeneratePathFilter(instance);
  const pathRes: any = await getPathList(projectInstanceId);
  const paths = normalizeProjectRows(unwrapResponseData(pathRes))
    .filter((pathObj: any) => Number(pathObj.enabled || 0) === 1)
    .filter((pathObj: any) => {
      if (pathIds.length > 0) {
        return pathIds.includes(Number(pathObj.ID || 0));
      }
      return Number(pathObj.pathSet || 0) === Number(pathSet || 0);
    });

  return {
    projectInstanceId,
    pathSet,
    pathIds,
    placeholders: mergeDbTemplatePlaceholders(paths),
  };
};

const buildGenerateCodexHandoffText = (result: any) => {
  if (!result) return '';
  const files = Array.isArray(result.files) ? result.files : [];
  const absolutePaths = files
    .map((file: any) => String(file?.absolutePath || file?.path || '').trim())
    .filter(Boolean);

  return [
    '生成文件绝对路径：',
    ...(absolutePaths.length > 0 ? absolutePaths.map((path, index) => `${index + 1}. ${path}`) : ['-']),
    '',
    '提示词接口地址（无需鉴权，Codex 可直接访问）：',
    String(result.promptUrl || '').trim() || '-',
  ].join('\n');
};

export default function ProjectDashboard() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [currentProject, setCurrentProject] = useState<any>({});
  const [projectModalMode, setProjectModalMode] = useState<'create' | 'edit' | 'clone'>('create');
  const [configProject, setConfigProject] = useState<any | null>(null);
  const [configProjectInstanceId, setConfigProjectInstanceId] = useState<number | null>(null);
  const [configPathDetailTarget, setConfigPathDetailTarget] = useState<{
    pathSetKey: string;
    pathSet: number;
    pathGroupKey: string;
  } | null>(null);
  const [generateProject, setGenerateProject] = useState<any | null>(null);
  const [generateDraft, setGenerateDraft] = useState({ module: '', tableName: '', overwrite: false });
  const [generateResult, setGenerateResult] = useState<any | null>(null);
  const [dbTemplateProject, setDbTemplateProject] = useState<any | null>(null);
  const [generatingTemplateProjectId, setGeneratingTemplateProjectId] = useState<number | null>(null);
  const [copyingTemplateProjectId, setCopyingTemplateProjectId] = useState<number | null>(null);
  const [dbSqlPreviewOpen, setDbSqlPreviewOpen] = useState(false);
  const [dbSqlPreviewTitle, setDbSqlPreviewTitle] = useState('');
  const [dbSqlPreviewContent, setDbSqlPreviewContent] = useState('');
  const [copyingDbSqlPreview, setCopyingDbSqlPreview] = useState(false);
  const [dbPlaceholderProject, setDbPlaceholderProject] = useState<any | null>(null);
  const [dbPlaceholderRows, setDbPlaceholderRows] = useState<DbTemplatePlaceholder[]>([]);
  const [applyingDbPlaceholders, setApplyingDbPlaceholders] = useState(false);
  const [generatePlaceholderProject, setGeneratePlaceholderProject] = useState<any | null>(null);
  const [generatePlaceholderRows, setGeneratePlaceholderRows] = useState<DbTemplatePlaceholder[]>([]);
  const [generatePlaceholderMeta, setGeneratePlaceholderMeta] = useState<{ projectInstanceId: number; pathSet: number; pathIds: number[] } | null>(null);
  const [applyingGeneratePlaceholders, setApplyingGeneratePlaceholders] = useState(false);
  const [selectedBusinessType, setSelectedBusinessType] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [editingBusinessType, setEditingBusinessType] = useState<string | null>(null);
  const [editingBusinessTypeName, setEditingBusinessTypeName] = useState('');
  const [savingBusinessType, setSavingBusinessType] = useState(false);
  const [savingProject, setSavingProject] = useState(false);
  const businessTypeInputRef = useRef<HTMLInputElement>(null);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const res = await getProjectList();
      setProjects(normalizeProjectRows(unwrapResponseData(res)));
    } catch (e) {
      toast.error('获取项目列表失败');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  useEffect(() => {
    const configProjectId = Number(searchParams.get('configProjectId') || 0);
    if (!configProjectId || loading) return;

    const projectInstanceId = Number(searchParams.get('projectInstanceId') || 0);
    const configView = String(searchParams.get('configView') || '').trim();
    const pathSetKey = String(searchParams.get('pathSetKey') || '').trim();
    const pathSet = Number(searchParams.get('pathSet') || 0);
    const pathGroupKey = String(searchParams.get('pathGroupKey') || '').trim();
    const nextProject = projects.find((project) => Number(project.ID || 0) === configProjectId);
    if (nextProject) {
      setConfigProjectInstanceId(projectInstanceId || null);
      setConfigPathDetailTarget(configView === 'pathDetail' && pathGroupKey ? {
        pathSetKey,
        pathSet,
        pathGroupKey,
      } : null);
      setConfigProject(nextProject);
    }
    setSearchParams(new URLSearchParams(), { replace: true });
  }, [loading, projects, searchParams, setSearchParams]);

  const businessTypes = useMemo(() => {
    const counts = new Map<string, number>();
    projects.forEach((project) => {
      const typeName = getProjectBusinessType(project);
      counts.set(typeName, (counts.get(typeName) || 0) + 1);
    });
    return Array.from(counts.entries())
      .map(([typeName, count]) => ({ typeName, count }))
      .sort((a, b) => {
        if (a.typeName === DEFAULT_BUSINESS_TYPE) return 1;
        if (b.typeName === DEFAULT_BUSINESS_TYPE) return -1;
        return a.typeName.localeCompare(b.typeName, 'zh-Hans-CN');
      });
  }, [projects]);

  useEffect(() => {
    if (selectedBusinessType && !businessTypes.some((item) => item.typeName === selectedBusinessType)) {
      setSelectedBusinessType(null);
    }
  }, [businessTypes, selectedBusinessType]);

  useEffect(() => {
    if (editingBusinessType && businessTypeInputRef.current) {
      businessTypeInputRef.current.focus();
      businessTypeInputRef.current.select();
    }
  }, [editingBusinessType]);

  const filteredProjects = useMemo(() => {
    const keyword = searchQuery.trim().toLowerCase();
    return projects.filter((project) => {
      const typeOk = selectedBusinessType === null || getProjectBusinessType(project) === selectedBusinessType;
      const searchOk = matchesProjectCardSearch(project, keyword);
      return typeOk && searchOk;
    });
  }, [projects, searchQuery, selectedBusinessType]);

  const openCreateProject = () => {
    setProjectModalMode('create');
    setCurrentProject({
      businessType: selectedBusinessType && selectedBusinessType !== DEFAULT_BUSINESS_TYPE ? selectedBusinessType : '',
      projectType: PROJECT_TYPE_BACKEND,
    });
    setShowModal(true);
  };

  const openEditProject = (project: any) => {
    setProjectModalMode('edit');
    setCurrentProject(project);
    setShowModal(true);
  };

  const openCloneProject = (project: any) => {
    setProjectModalMode('clone');
    setCurrentProject({
      sourceProjectId: Number(project.ID || 0),
      projectName: `${String(project.projectName || '').trim() || '未命名卡片'}_copy`,
      businessType: String(project.businessType || '').trim(),
      projectType: normalizeProjectType(project.projectType),
      remark: String(project.remark || '').trim(),
      userName: project.userName || 'conchi',
    });
    setShowModal(true);
  };

  const resetBusinessTypeEditing = () => {
    setEditingBusinessType(null);
    setEditingBusinessTypeName('');
  };

  const openRenameBusinessType = (typeName: string) => {
    if (typeName === DEFAULT_BUSINESS_TYPE) return;
    setEditingBusinessType(typeName);
    setEditingBusinessTypeName(typeName);
  };

  const updateProjectsBusinessType = async (typeName: string, nextTypeName: string) => {
    const nextBusinessType = nextTypeName === DEFAULT_BUSINESS_TYPE ? '' : nextTypeName;
    const affectedProjects = projects.filter((project) => getProjectBusinessType(project) === typeName);

    await Promise.all(
      affectedProjects.map((project) => updateProject({
        ...project,
        businessType: nextBusinessType,
        projectConfigId: 0,
      }))
    );
  };

  const handleRenameBusinessType = async (typeName: string) => {
    const nextTypeName = editingBusinessTypeName.trim();
    if (!nextTypeName) {
      toast.error('业务类型名称不能为空');
      return;
    }
    if (nextTypeName === typeName) {
      resetBusinessTypeEditing();
      return;
    }

    setSavingBusinessType(true);
    try {
      await updateProjectsBusinessType(typeName, nextTypeName);
      toast.success('业务类型已重命名');
      resetBusinessTypeEditing();
      await fetchProjects();
      setSelectedBusinessType(nextTypeName === DEFAULT_BUSINESS_TYPE ? DEFAULT_BUSINESS_TYPE : nextTypeName);
    } catch (e) {
      toast.error('重命名业务类型失败');
    } finally {
      setSavingBusinessType(false);
    }
  };

  const handleDeleteBusinessType = async (typeName: string, count: number) => {
    if (typeName === DEFAULT_BUSINESS_TYPE) return;
    if (!confirm(`确定删除业务类型「${typeName}」吗？该类型下 ${count} 张卡片将归入「${DEFAULT_BUSINESS_TYPE}」。`)) return;

    setSavingBusinessType(true);
    try {
      await updateProjectsBusinessType(typeName, DEFAULT_BUSINESS_TYPE);
      toast.success(`已删除业务类型，${count} 张卡片已归入${DEFAULT_BUSINESS_TYPE}`);
      resetBusinessTypeEditing();
      await fetchProjects();
      setSelectedBusinessType(DEFAULT_BUSINESS_TYPE);
    } catch (e) {
      toast.error('删除业务类型失败');
    } finally {
      setSavingBusinessType(false);
    }
  };

  const handleSave = async () => {
    setSavingProject(true);
    try {
      const payload = {
        ...currentProject,
        businessType: String(currentProject.businessType || '').trim(),
        projectType: normalizeProjectType(currentProject.projectType),
        projectConfigId: 0,
      };
      if (projectModalMode === 'clone') {
        await copyProject({
          sourceProjectId: Number(currentProject.sourceProjectId || 0),
          projectName: String(payload.projectName || '').trim(),
          businessType: payload.businessType,
          projectType: payload.projectType,
          remark: String(payload.remark || '').trim(),
          userName: currentProject.userName || 'conchi',
        });
        toast.success('克隆成功');
      } else if (currentProject.ID) {
        await updateProject(payload);
        toast.success('更新成功');
      } else {
        await createProject({ ...payload, userName: currentProject.userName || 'conchi' });
        toast.success('创建成功');
      }
      setShowModal(false);
      await fetchProjects();
      setSelectedBusinessType(payload.businessType || DEFAULT_BUSINESS_TYPE);
    } catch (e) {
      toast.error('保存失败');
    } finally {
      setSavingProject(false);
    }
  };

  const handleDelete = async (data: any) => {
    if (confirm('确定删除该项目及其所有配置吗？')) {
      try {
        await deleteProject(data);
        toast.success('删除成功');
        fetchProjects();
      } catch (e) {
        toast.error('删除失败');
      }
    }
  };

  const openGenerateDialog = (project: any) => {
    setGenerateProject(project);
    setGenerateDraft({ module: '', tableName: '', overwrite: false });
    setGenerateResult(null);
  };

  const closeGenerateDialog = () => {
    if (generatingTemplateProjectId) return;
    setGenerateProject(null);
    setGenerateResult(null);
    setGeneratePlaceholderProject(null);
    setGeneratePlaceholderRows([]);
    setGeneratePlaceholderMeta(null);
  };

  const runGenerateCode = async (
    project: any,
    placeholderValues: DbTemplatePlaceholderValues = {},
    meta?: { projectInstanceId: number; pathSet: number; pathIds: number[] } | null,
  ) => {
    const templateProjectId = Number(project?.ID || 0);
    const module = generateDraft.module.trim();
    const tableName = generateDraft.tableName.trim();

    setGeneratingTemplateProjectId(templateProjectId);
    try {
      const res: any = await generateProjectCode({
        templateProjectId,
        projectInstanceId: Number(meta?.projectInstanceId || 0),
        pathSet: Number(meta?.pathSet || 0),
        pathIds: Array.isArray(meta?.pathIds) ? meta?.pathIds : [],
        module,
        tableName,
        overwrite: generateDraft.overwrite,
        placeholderValues,
      });
      if (typeof res?.code !== 'undefined' && Number(res.code) !== 0) {
        throw new Error(res.msg || 'generate failed');
      }
      const result = unwrapResponseData(res);
      setGenerateResult(result);
      toast.success(`Codex 任务已准备：${Number((result?.files || []).length)} 个目标文件`);
      return true;
    } catch (e) {
      toast.error('生成代码失败');
      return false;
    } finally {
      setGeneratingTemplateProjectId(null);
    }
  };

  const handleGenerateCode = async () => {
    const templateProjectId = Number(generateProject?.ID || 0);
    const module = generateDraft.module.trim();
    const tableName = generateDraft.tableName.trim();

    if (!templateProjectId) return;
    if (!module) {
      toast.error('module 不能为空');
      return;
    }
    if (!tableName) {
      toast.error('TableName 不能为空');
      return;
    }

    setGeneratingTemplateProjectId(templateProjectId);
    try {
      const payload = await buildGenerateCodePlaceholderPayload(generateProject);
      if (payload.placeholders.length > 0) {
        setGeneratePlaceholderProject(generateProject);
        setGeneratePlaceholderRows(payload.placeholders.map((item) => ({ ...item })));
        setGeneratePlaceholderMeta({
          projectInstanceId: payload.projectInstanceId,
          pathSet: payload.pathSet,
          pathIds: payload.pathIds,
        });
        return;
      }
      await runGenerateCode(generateProject, {}, payload);
    } catch (e) {
      toast.error('读取动态占位符失败');
    } finally {
      setGeneratingTemplateProjectId(null);
    }
  };

  const handleCopyDbTemplateSql = async (project: any) => {
    const projectId = Number(project.ID || 0);
    if (!projectId) return;

    setCopyingTemplateProjectId(projectId);
    try {
      const { sections, placeholders } = await buildDbTemplateSqlPayload(project);

      if (sections.length === 0) {
        toast.error('该项目暂无可复制的 SQL 内容');
        return;
      }

      if (placeholders.length > 0) {
        setDbPlaceholderProject(project);
        setDbPlaceholderRows(placeholders.map((item) => ({ ...item })));
        return;
      }

      await openDbSqlPreview(project, sections);
    } catch (e) {
      toast.error('复制数据库模板失败');
    } finally {
      setCopyingTemplateProjectId(null);
    }
  };

  const openDbSqlPreview = async (project: any, sections: string[]) => {
    const projectId = Number(project?.ID || 0);
    const copyText = buildDbTemplateSqlCopyText(sections);
    await copyTextToClipboard(copyText);
    setDbSqlPreviewTitle(project?.projectName || `Project ${projectId}`);
    setDbSqlPreviewContent(copyText);
    setDbSqlPreviewOpen(true);
    toast.success('数据库模板 SQL 已复制');
  };

  const updateDbPlaceholderRow = (index: number, value: string) => {
    setDbPlaceholderRows((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, value } : row));
  };

  const closeDbPlaceholderDialog = () => {
    if (applyingDbPlaceholders) return;
    setDbPlaceholderProject(null);
    setDbPlaceholderRows([]);
  };

  const handleApplyDbPlaceholders = async () => {
    if (!dbPlaceholderProject) return;
    const projectId = Number(dbPlaceholderProject.ID || 0);
    const values = dbPlaceholderRows.reduce<DbTemplatePlaceholderValues>((next, row) => {
      const key = String(row.key || '').trim();
      if (key) next[key] = String(row.value ?? '');
      return next;
    }, {});

    setApplyingDbPlaceholders(true);
    setCopyingTemplateProjectId(projectId);
    try {
      const { sections } = await buildDbTemplateSqlPayload(dbPlaceholderProject, values);
      if (sections.length === 0) {
        toast.error('该项目暂无可复制的 SQL 内容');
        return;
      }
      await openDbSqlPreview(dbPlaceholderProject, sections);
      setDbPlaceholderProject(null);
      setDbPlaceholderRows([]);
    } catch (e) {
      toast.error('复制数据库模板失败');
    } finally {
      setApplyingDbPlaceholders(false);
      setCopyingTemplateProjectId(null);
    }
  };

  const handleCopyDbSqlPreview = async () => {
    if (!dbSqlPreviewContent.trim()) return;
    setCopyingDbSqlPreview(true);
    try {
      await copyTextToClipboard(dbSqlPreviewContent);
      toast.success('数据库模板 SQL 已复制');
    } catch (e) {
      toast.error('复制数据库模板失败');
    } finally {
      setCopyingDbSqlPreview(false);
    }
  };

  const updateGeneratePlaceholderRow = (index: number, value: string) => {
    setGeneratePlaceholderRows((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, value } : row));
  };

  const closeGeneratePlaceholderDialog = () => {
    if (applyingGeneratePlaceholders) return;
    setGeneratePlaceholderProject(null);
    setGeneratePlaceholderRows([]);
    setGeneratePlaceholderMeta(null);
  };

  const handleApplyGeneratePlaceholders = async () => {
    if (!generatePlaceholderProject) return;
    const values = generatePlaceholderRows.reduce<DbTemplatePlaceholderValues>((next, row) => {
      const key = String(row.key || '').trim();
      if (key) next[key] = String(row.value ?? '');
      return next;
    }, {});

    setApplyingGeneratePlaceholders(true);
    try {
      const ok = await runGenerateCode(generatePlaceholderProject, values, generatePlaceholderMeta);
      if (ok) {
        setGeneratePlaceholderProject(null);
        setGeneratePlaceholderRows([]);
        setGeneratePlaceholderMeta(null);
      }
    } catch (e) {
      toast.error('生成代码失败');
    } finally {
      setApplyingGeneratePlaceholders(false);
    }
  };

  return (
    <div className="w-full flex bg-white animate-fade-in">
      <aside className="w-60 shrink-0 bg-white border-r border-gray-100 min-h-[calc(100vh-64px)] flex flex-col">
        <div className="px-3 pt-4 pb-2">
          <div className="relative mb-3">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="搜索代码卡片..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full bg-gray-50 border border-gray-200 rounded-md py-1.5 pl-8 pr-2 text-xs focus:outline-none focus:ring-2 focus:ring-black/5 focus:border-gray-300"
            />
          </div>
          <div className="flex items-center justify-between mb-2 px-1">
            <span className="text-[10px] font-bold text-gray-400 uppercase tracking-wider">业务类型</span>
            <button
              onClick={openCreateProject}
              className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors flex items-center justify-center"
              title="新建卡片"
            >
              <Plus size={14} strokeWidth={2.5} />
            </button>
          </div>
        </div>

        <div className="px-3 flex flex-col gap-0.5">
          {businessTypes.map((item) => {
            const active = selectedBusinessType === item.typeName;
            const editing = editingBusinessType === item.typeName;
            return (
              <div key={item.typeName} className="group/item flex items-center gap-1">
                {editing ? (
                  <>
                    <input
                      ref={businessTypeInputRef}
                      value={editingBusinessTypeName}
                      disabled={savingBusinessType}
                      onChange={e => setEditingBusinessTypeName(e.target.value)}
                      onKeyDown={e => {
                        if (e.key === 'Enter') handleRenameBusinessType(item.typeName);
                        if (e.key === 'Escape') resetBusinessTypeEditing();
                      }}
                      className="min-w-0 flex-1 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-medium outline-none focus:ring-2 focus:ring-black/10 disabled:opacity-60"
                    />
                    <button
                      onClick={() => handleRenameBusinessType(item.typeName)}
                      disabled={savingBusinessType}
                      className="p-2 rounded-lg bg-gray-900 text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-60"
                      title="保存业务类型"
                    >
                      <Check size={14} />
                    </button>
                    <button
                      onClick={resetBusinessTypeEditing}
                      disabled={savingBusinessType}
                      className="p-2 rounded-lg text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-60"
                      title="取消"
                    >
                      <X size={14} />
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      onClick={() => setSelectedBusinessType(item.typeName)}
                      className={`min-w-0 flex-1 flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${active ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'}`}
                    >
                      <Folder size={14} className={active ? 'text-white' : 'text-gray-400'} />
                      <span className="truncate flex-1 text-left" title={item.typeName}>{item.typeName}</span>
                      <span className={`text-xs px-1.5 py-0.5 rounded-full font-mono ${active ? 'bg-white/20 text-white' : 'bg-gray-100 text-gray-500'}`}>{item.count}</span>
                    </button>
                    <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover/item:opacity-100 group-focus-within/item:opacity-100">
                      <button
                        onClick={() => openRenameBusinessType(item.typeName)}
                        disabled={savingBusinessType || item.typeName === DEFAULT_BUSINESS_TYPE}
                        className="p-1.5 rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-40"
                        title={item.typeName === DEFAULT_BUSINESS_TYPE ? '未分类为系统兜底类型，不能重命名' : '重命名业务类型'}
                      >
                        <Edit2 size={13} />
                      </button>
                      <button
                        onClick={() => handleDeleteBusinessType(item.typeName, item.count)}
                        disabled={savingBusinessType || item.typeName === DEFAULT_BUSINESS_TYPE}
                        className="p-1.5 rounded-md text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                        title={item.typeName === DEFAULT_BUSINESS_TYPE ? '未分类为系统兜底类型，不能删除' : `删除业务类型，${item.count} 张卡片将归入未分类`}
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </>
                )}
              </div>
            );
          })}
        </div>
      </aside>

      <main className="flex-1 min-w-0 px-6 py-6">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-5">
          <div className="min-w-0">
            <h1 className="text-2xl font-extrabold text-gray-900">代码生成</h1>
            <p className="text-sm text-gray-500 mt-1">
              {selectedBusinessType || '全部业务类型'} · {filteredProjects.length} 张卡片
            </p>
          </div>
          <button
            onClick={openCreateProject}
            className="flex items-center gap-2 bg-gray-900 hover:bg-gray-800 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm"
          >
            <Plus size={16} />
            <span>新建卡片</span>
          </button>
        </div>

        {loading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
            {[1, 2, 3].map(i => (
              <div key={i} className="h-56 bg-gray-100 rounded-lg animate-pulse"></div>
            ))}
          </div>
        ) : filteredProjects.length === 0 ? (
          <div className="border border-dashed border-gray-300 rounded-lg p-12 text-center bg-gray-50 mt-2">
            <Folder size={32} className="text-gray-300 mx-auto mb-3" />
            <h3 className="text-base font-medium text-gray-900 mb-1">
              {selectedBusinessType ? '该业务类型暂无代码卡片' : '还没有任何代码卡片'}
            </h3>
            <p className="text-sm text-gray-400 mb-5">
              从左侧或右上角新建一张代码生成卡片
            </p>
            <button onClick={openCreateProject} className="bg-black hover:bg-gray-800 text-white font-medium py-2 px-5 rounded-lg text-sm transition-colors">
              新建卡片
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
            <AnimatePresence>
              {filteredProjects.map((p: any) => {
                const showDbTemplateActions = shouldShowDbTemplateActions(p);

                return (
                  <motion.div
                    layout
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.96 }}
                    key={p.ID}
                    data-testid="code-project-card"
                    onClick={() => {
                      setConfigProjectInstanceId(null);
                      setConfigPathDetailTarget(null);
                      setConfigProject(p);
                    }}
                    className="group min-w-0 bg-white rounded-lg shadow-sm border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200 overflow-hidden flex flex-col cursor-pointer"
                  >
                  <div className="p-5">
                    <div className="flex justify-between items-start gap-3 mb-4">
                      <div className="flex min-w-0 items-center gap-3">
                        <div className="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center border border-gray-200 text-teal-600">
                          <FileCode size={20} />
                        </div>
                        <div className="min-w-0">
                          <h3 className="font-bold text-base text-gray-900 truncate" title={p.projectName}>{p.projectName}</h3>
                          <p className="text-xs text-gray-400 font-mono">ID: {p.ID}</p>
                        </div>
                      </div>
                      <div className="flex shrink-0 gap-1">
                        <button
                          onClick={(event) => {
                            event.stopPropagation();
                            openEditProject(p);
                          }}
                          className="p-1.5 text-gray-400 hover:text-teal-600 hover:bg-teal-50 rounded-md transition-colors"
                          title="编辑卡片"
                        >
                          <Edit2 size={15} />
                        </button>
                        <button
                          onClick={(event) => {
                            event.stopPropagation();
                            handleDelete(p);
                          }}
                          className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                          title="删除卡片"
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </div>

                    <div className="space-y-3">
                      <div className="flex flex-wrap gap-1.5">
                        <div className="inline-flex max-w-full items-center gap-1.5 px-2 py-1 rounded-md bg-gray-100 text-xs font-medium text-gray-600">
                          <Folder size={12} className="text-gray-400" />
                          <span className="truncate" title={getProjectBusinessType(p)}>{getProjectBusinessType(p)}</span>
                        </div>
                        <div className="inline-flex items-center rounded-md bg-slate-900 px-2 py-1 text-xs font-bold text-white">
                          {getProjectTypeLabel(p.projectType)}
                        </div>
                      </div>
                      <div className="text-sm text-gray-500 line-clamp-2 min-h-[2.5rem]">
                        {p.remark || <span className="italic text-gray-400">暂无备注说明</span>}
                      </div>
                    </div>
                  </div>

                  <div className="mt-auto border-t border-gray-100 bg-gray-50/50 p-4 flex flex-col gap-2">
                    {showDbTemplateActions && (
                      <div className="flex min-h-[38px] gap-2">
                        <button
                          onClick={(event) => {
                            event.stopPropagation();
                            setDbTemplateProject(p);
                          }}
                          className="flex-1 min-w-0 flex justify-center items-center gap-1.5 py-2 px-2 text-sm text-cyan-800 bg-cyan-50 hover:bg-cyan-100 rounded-md transition-colors font-bold border border-cyan-200"
                        >
                          <Database size={15} /> <span className="truncate">数据库模板</span>
                        </button>
                        <button
                          onClick={(event) => {
                            event.stopPropagation();
                            handleCopyDbTemplateSql(p);
                          }}
                          disabled={copyingTemplateProjectId === Number(p.ID)}
                          className="flex-1 min-w-0 flex justify-center items-center gap-1.5 py-2 px-2 text-sm text-emerald-700 bg-emerald-50 hover:bg-emerald-100 rounded-md transition-colors font-bold border border-emerald-200 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                          <ClipboardCopy size={15} /> <span className="truncate">{copyingTemplateProjectId === Number(p.ID) ? '复制中' : '复制 SQL'}</span>
                        </button>
                      </div>
                    )}
                    <div className="flex gap-2">
                      <button
                        onClick={(event) => {
                          event.stopPropagation();
                          setConfigProjectInstanceId(null);
                          setConfigPathDetailTarget(null);
                          setConfigProject(p);
                        }}
                        className="flex-1 min-w-0 flex justify-center items-center gap-1.5 py-2 px-2 text-sm text-indigo-700 bg-indigo-50 hover:bg-indigo-100 rounded-md transition-colors font-bold border border-indigo-200"
                      >
                        <FileCode size={15} /> <span className="truncate">编辑代码模版</span>
                      </button>
                      <button
                        onClick={(event) => {
                          event.stopPropagation();
                          openGenerateDialog(p);
                        }}
                        className="flex-1 min-w-0 flex justify-center items-center gap-1.5 py-2 px-2 text-sm text-slate-950 bg-teal-100 hover:bg-teal-200 rounded-md transition-colors font-bold border border-teal-300"
                      >
                        <Wand2 size={15} /> <span className="truncate">生成代码</span>
                      </button>
                    </div>
                    <button
                      onClick={(event) => {
                        event.stopPropagation();
                        openCloneProject(p);
                      }}
                      className="flex w-full min-w-0 justify-center items-center gap-1.5 py-2 px-2 text-sm text-gray-600 bg-white hover:bg-gray-50 border border-gray-200 rounded-md transition-colors"
                    >
                      <Copy size={15} /> <span className="truncate">克隆</span>
                    </button>
                    {!showDbTemplateActions && <div className="min-h-[38px]" aria-hidden="true" />}
                  </div>
                  </motion.div>
                );
              })}
            </AnimatePresence>
          </div>
        )}
      </main>

      {configProject && (
        <ProjectConfigDialog
          project={configProject}
          initialProjectInstanceId={configProjectInstanceId}
          initialPathSetKey={configPathDetailTarget?.pathSetKey || null}
          initialPathSet={configPathDetailTarget?.pathSet || null}
          initialPathGroupKey={configPathDetailTarget?.pathGroupKey || null}
          onClose={() => {
            setConfigProject(null);
            setConfigProjectInstanceId(null);
            setConfigPathDetailTarget(null);
          }}
          onProjectSaved={fetchProjects}
        />
      )}

      <AnimatePresence>
        {dbTemplateProject && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[60] bg-slate-950 text-white"
          >
            <DbTemplateLibrary
              projectIdOverride={dbTemplateProject.ID}
              fullscreenDialog
              onClose={() => setDbTemplateProject(null)}
            />
          </motion.div>
        )}
      </AnimatePresence>

      <AnimatePresence>
        {generateProject && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              className="absolute inset-0 bg-slate-900/50 backdrop-blur-sm"
              onClick={closeGenerateDialog}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="relative flex max-h-[88vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="border-b border-slate-200 px-6 py-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-xs font-bold uppercase tracking-wider text-teal-600">生成代码</div>
                    <h2 className="mt-1 truncate text-xl font-bold text-slate-900" title={generateProject.projectName || ''}>
                      {generateProject.projectName || '未命名卡片'}
                    </h2>
                  </div>
                  <button
                    type="button"
                    onClick={closeGenerateDialog}
                    disabled={Boolean(generatingTemplateProjectId)}
                    className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
                    title="关闭"
                  >
                    <X size={18} />
                  </button>
                </div>
                {generateResult?.diskPath && (
                  <div className="mt-3 truncate rounded-md bg-slate-50 px-3 py-2 font-mono text-xs font-semibold text-slate-500" title={generateResult.diskPath || ''}>
                    {generateResult.diskPath}
                  </div>
                )}
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div>
                    <label className="mb-2 block text-sm font-bold text-slate-700">module</label>
                    <input
                      type="text"
                      value={generateDraft.module}
                      onChange={e => setGenerateDraft({ ...generateDraft, module: e.target.value })}
                      className="w-full rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-sm font-bold text-slate-900 outline-none transition focus:border-teal-400 focus:bg-white focus:ring-2 focus:ring-teal-500/20"
                      placeholder="btStation"
                      autoFocus
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-bold text-slate-700">TableName</label>
                    <input
                      type="text"
                      value={generateDraft.tableName}
                      onChange={e => setGenerateDraft({ ...generateDraft, tableName: e.target.value })}
                      className="w-full rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-sm font-bold text-slate-900 outline-none transition focus:border-teal-400 focus:bg-white focus:ring-2 focus:ring-teal-500/20"
                      placeholder="BtStation"
                    />
                  </div>
                </div>

                <label className="mt-5 flex cursor-pointer items-center justify-between gap-4 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3">
                  <span className="text-sm font-bold text-slate-700">覆盖已存在目标文件</span>
                  <input
                    type="checkbox"
                    checked={generateDraft.overwrite}
                    onChange={e => setGenerateDraft({ ...generateDraft, overwrite: e.target.checked })}
                    className="h-5 w-5 accent-teal-500"
                  />
                </label>

                {generateResult && (
                  <div className="mt-5 space-y-4">
                    <div className="rounded-lg border border-teal-200 bg-teal-50">
                      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-teal-200 px-4 py-3">
                        <div>
                          <div className="text-sm font-bold text-teal-900">Codex 任务已准备</div>
                          <div className="mt-0.5 text-xs font-semibold text-teal-700">
                            目标文件 {Number((generateResult.files || []).length)} 个 / 已写入 {Number(generateResult.generatedCount || 0)} 个 / 跳过 {Number(generateResult.skippedCount || 0)} 个
                          </div>
                        </div>
                        <div className="rounded-full bg-white px-2 py-1 text-xs font-bold text-teal-700 ring-1 ring-teal-200">
                          pathSet {Number(generateResult.pathSet || 0)}
                        </div>
                      </div>
                      <div className="space-y-3 px-4 py-3">
                        <div>
                          <div className="mb-1 text-xs font-bold uppercase tracking-wider text-teal-700">提示词接口地址</div>
                          <div className="flex items-center gap-2 rounded-md bg-white px-3 py-2 ring-1 ring-teal-200">
                            <span className="min-w-0 flex-1 truncate font-mono text-xs font-semibold text-slate-700" title={generateResult.promptUrl || ''}>
                              {generateResult.promptUrl || '-'}
                            </span>
                            <button
                              type="button"
                              onClick={async () => {
                                try {
                                  await copyTextToClipboard(generateResult.promptUrl || '');
                                  toast.success('提示词接口已复制');
                                } catch (e) {
                                  toast.error('复制提示词接口失败');
                                }
                              }}
                              disabled={!generateResult.promptUrl}
                              className="shrink-0 rounded-md p-1.5 text-teal-700 transition-colors hover:bg-teal-100 disabled:cursor-not-allowed disabled:opacity-50"
                              title="复制提示词接口"
                            >
                              <Copy size={14} />
                            </button>
                          </div>
                        </div>
                        <div className="rounded-md bg-white px-3 py-2 text-xs font-semibold leading-5 text-slate-600 ring-1 ring-teal-200">
                          {generateResult.modifyInstructions}
                        </div>
                      </div>
                    </div>

                    <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
                      <div className="flex items-center justify-between gap-3 border-b border-slate-200 bg-slate-50 px-4 py-3">
                        <div>
                          <div className="text-sm font-bold text-slate-800">给 Codex 的文本</div>
                          <div className="mt-0.5 text-xs font-semibold text-slate-400">包含生成文件绝对路径和无需鉴权的提示词接口地址</div>
                        </div>
                        <button
                          type="button"
                          onClick={async () => {
                            try {
                              await copyTextToClipboard(buildGenerateCodexHandoffText(generateResult));
                              toast.success('Codex 文本已复制');
                            } catch (e) {
                              toast.error('复制 Codex 文本失败');
                            }
                          }}
                          className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md bg-white px-3 text-xs font-bold text-slate-700 ring-1 ring-slate-200 transition-colors hover:bg-slate-100"
                        >
                          <Copy size={13} />
                          复制
                        </button>
                      </div>
                      <textarea
                        readOnly
                        value={buildGenerateCodexHandoffText(generateResult)}
                        className="h-44 w-full resize-none border-0 bg-white p-4 font-mono text-xs font-semibold leading-6 text-slate-700 outline-none"
                      />
                    </div>

                    <div className="rounded-lg border border-slate-200 bg-slate-50">
                      <div className="border-b border-slate-200 px-4 py-3 text-sm font-bold text-slate-800">
                        目标文件绝对路径与修改方式
                      </div>
                      <div className="max-h-72 overflow-y-auto p-2">
                        {(generateResult.files || []).map((file: any) => (
                          <div key={`${file.pathId}-${file.relativePath}`} className="rounded-md px-2 py-2 text-xs hover:bg-white">
                            <div className="mb-1 flex items-center gap-2">
                              <span className={`shrink-0 rounded-full px-2 py-0.5 font-bold ${file.status === 'skipped' ? 'bg-amber-100 text-amber-700' : 'bg-emerald-100 text-emerald-700'}`}>
                                {file.status === 'skipped' ? '跳过' : file.status === 'overwritten' ? '覆盖' : '生成'}
                              </span>
                              <span className="min-w-0 flex-1 truncate font-mono font-bold text-slate-700" title={file.absolutePath || file.path}>
                                {file.absolutePath || file.path}
                              </span>
                            </div>
                            <div className="truncate pl-12 font-mono text-[11px] font-semibold text-slate-400" title={file.relativePath}>
                              {file.relativePath}
                            </div>
                            <div className="mt-1 pl-12 text-xs font-semibold leading-5 text-slate-500">
                              {file.instruction}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-3 border-t border-slate-200 bg-white px-6 py-4">
                <button
                  type="button"
                  onClick={closeGenerateDialog}
                  disabled={Boolean(generatingTemplateProjectId)}
                  className="rounded-lg px-5 py-2.5 text-sm font-bold text-slate-600 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  取消
                </button>
                <button
                  type="button"
                  onClick={handleGenerateCode}
                  disabled={Boolean(generatingTemplateProjectId)}
                  className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-slate-900/15 transition-colors hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {generatingTemplateProjectId ? <RefreshCw size={16} className="animate-spin" /> : <Wand2 size={16} />}
                  准备任务
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* Modal */}
      <AnimatePresence>
        {showModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm"
              onClick={() => setShowModal(false)}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="relative w-full max-w-lg bg-white rounded-lg shadow-2xl p-6"
            >
              <h2 className="text-xl font-bold text-slate-800 mb-6">
                {projectModalMode === 'clone' ? '克隆代码卡片' : currentProject.ID ? '编辑代码卡片' : '新建代码卡片'}
              </h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">项目名称</label>
                  <input
                    type="text"
                    value={currentProject.projectName || ''}
                    onChange={e => setCurrentProject({ ...currentProject, projectName: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all"
                    placeholder="输入项目名, 比如 Easy Backend..."
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">业务类型</label>
                  <input
                    type="text"
                    value={currentProject.businessType || ''}
                    onChange={e => setCurrentProject({ ...currentProject, businessType: e.target.value })}
                    list="code-generate-business-types"
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all"
                    placeholder="例如: 后端 CRUD / 前端页面 / SQL 模板"
                    autoComplete="off"
                  />
                  <datalist id="code-generate-business-types">
                    {businessTypes
                      .filter(item => item.typeName !== DEFAULT_BUSINESS_TYPE)
                      .map(item => <option key={item.typeName} value={item.typeName} />)}
                  </datalist>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">项目类型</label>
                  <div className="grid grid-cols-2 gap-2 rounded-lg bg-slate-100 p-1">
                    {[
                      { value: PROJECT_TYPE_BACKEND, label: '后端' },
                      { value: PROJECT_TYPE_FRONTEND, label: '前端' },
                    ].map((option) => {
                      const active = normalizeProjectType(currentProject.projectType) === option.value;
                      return (
                        <button
                          key={option.value}
                          type="button"
                          onClick={() => setCurrentProject({ ...currentProject, projectType: option.value })}
                          className={`rounded-md px-3 py-2 text-sm font-bold transition-colors ${
                            active
                              ? 'bg-slate-900 text-white shadow-sm'
                              : 'text-slate-500 hover:bg-white hover:text-slate-800'
                          }`}
                        >
                          {option.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">项目描述</label>
                  <textarea
                    value={currentProject.remark || ''}
                    onChange={e => setCurrentProject({ ...currentProject, remark: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all min-h-[100px]"
                    placeholder="简单的描述记录..."
                  />
                </div>
              </div>
              <div className="mt-8 flex justify-end gap-3">
                <button
                  onClick={() => setShowModal(false)}
                  disabled={savingProject}
                  className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-lg transition-colors font-medium"
                >
                  取消
                </button>
                <button
                  onClick={handleSave}
                  disabled={savingProject}
                  className="px-5 py-2.5 bg-slate-800 hover:bg-slate-900 text-white rounded-lg transition-colors font-medium shadow-lg shadow-slate-900/20 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {savingProject ? '保存中...' : '保存设置'}
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <AnimatePresence>
        {generatePlaceholderProject && (
          <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 bg-slate-900/50 backdrop-blur-sm"
              onClick={closeGeneratePlaceholderDialog}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="relative flex max-h-[88vh] w-full max-w-5xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
                <div className="min-w-0">
                  <div className="text-xs font-bold uppercase tracking-wider text-teal-600">代码生成动态占位符</div>
                  <h2 className="mt-1 truncate text-xl font-bold text-slate-900" title={generatePlaceholderProject.projectName || ''}>
                    {generatePlaceholderProject.projectName || '代码生成'}
                  </h2>
                </div>
                <button
                  type="button"
                  onClick={closeGeneratePlaceholderDialog}
                  disabled={applyingGeneratePlaceholders}
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
                  title="关闭"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto p-6">
                <div className="overflow-hidden rounded-lg border border-slate-200">
                  <div className="grid grid-cols-[minmax(140px,0.8fr)_minmax(200px,1.2fr)_minmax(220px,1.4fr)] border-b border-slate-200 bg-slate-50 text-sm font-bold text-slate-600">
                    <div className="px-4 py-3">占位符 key</div>
                    <div className="px-4 py-3">描述</div>
                    <div className="px-4 py-3">value</div>
                  </div>
                  {generatePlaceholderRows.map((row, index) => (
                    <div
                      key={`${row.key}-${index}`}
                      className="grid grid-cols-[minmax(140px,0.8fr)_minmax(200px,1.2fr)_minmax(220px,1.4fr)] items-center border-b border-slate-100 last:border-b-0"
                    >
                      <div className="break-all px-4 py-3 font-mono text-sm font-bold text-slate-700">{row.key}</div>
                      <div className="px-4 py-3 text-sm font-medium text-slate-500">{row.description || '-'}</div>
                      <div className="p-2">
                        <input
                          value={row.value || ''}
                          onChange={(event) => updateGeneratePlaceholderRow(index, event.target.value)}
                          className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-sm font-semibold text-slate-900 outline-none transition focus:border-teal-400 focus:bg-white focus:ring-2 focus:ring-teal-500/20"
                          autoFocus={index === 0}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex justify-end gap-3 border-t border-slate-200 bg-white px-6 py-4">
                <button
                  type="button"
                  onClick={closeGeneratePlaceholderDialog}
                  disabled={applyingGeneratePlaceholders}
                  className="rounded-lg px-5 py-2.5 text-sm font-bold text-slate-600 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  取消
                </button>
                <button
                  type="button"
                  onClick={handleApplyGeneratePlaceholders}
                  disabled={applyingGeneratePlaceholders}
                  className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-slate-900/15 transition-colors hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {applyingGeneratePlaceholders ? <RefreshCw size={16} className="animate-spin" /> : <Wand2 size={16} />}
                  生成代码
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <AnimatePresence>
        {dbPlaceholderProject && (
          <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 bg-slate-900/50 backdrop-blur-sm"
              onClick={closeDbPlaceholderDialog}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="relative flex max-h-[88vh] w-full max-w-5xl flex-col overflow-hidden rounded-lg bg-white shadow-2xl"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
                <div className="min-w-0">
                  <div className="text-xs font-bold uppercase tracking-wider text-emerald-600">动态占位符</div>
                  <h2 className="mt-1 truncate text-xl font-bold text-slate-900" title={dbPlaceholderProject.projectName || ''}>
                    {dbPlaceholderProject.projectName || '数据库模板 SQL'}
                  </h2>
                </div>
                <button
                  type="button"
                  onClick={closeDbPlaceholderDialog}
                  disabled={applyingDbPlaceholders}
                  className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
                  title="关闭"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="min-h-0 flex-1 overflow-y-auto p-6">
                <div className="overflow-hidden rounded-lg border border-slate-200">
                  <div className="grid grid-cols-[minmax(140px,0.8fr)_minmax(200px,1.2fr)_minmax(220px,1.4fr)] border-b border-slate-200 bg-slate-50 text-sm font-bold text-slate-600">
                    <div className="px-4 py-3">占位符 key</div>
                    <div className="px-4 py-3">描述</div>
                    <div className="px-4 py-3">value</div>
                  </div>
                  {dbPlaceholderRows.map((row, index) => (
                    <div
                      key={`${row.key}-${index}`}
                      className="grid grid-cols-[minmax(140px,0.8fr)_minmax(200px,1.2fr)_minmax(220px,1.4fr)] items-center border-b border-slate-100 last:border-b-0"
                    >
                      <div className="break-all px-4 py-3 font-mono text-sm font-bold text-slate-700">{row.key}</div>
                      <div className="px-4 py-3 text-sm font-medium text-slate-500">{row.description || '-'}</div>
                      <div className="p-2">
                        <input
                          value={row.value || ''}
                          onChange={(event) => updateDbPlaceholderRow(index, event.target.value)}
                          className="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-sm font-semibold text-slate-900 outline-none transition focus:border-emerald-400 focus:bg-white focus:ring-2 focus:ring-emerald-500/20"
                          autoFocus={index === 0}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex justify-end gap-3 border-t border-slate-200 bg-white px-6 py-4">
                <button
                  type="button"
                  onClick={closeDbPlaceholderDialog}
                  disabled={applyingDbPlaceholders}
                  className="rounded-lg px-5 py-2.5 text-sm font-bold text-slate-600 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  取消
                </button>
                <button
                  type="button"
                  onClick={handleApplyDbPlaceholders}
                  disabled={applyingDbPlaceholders}
                  className="inline-flex items-center gap-2 rounded-lg bg-slate-900 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-slate-900/15 transition-colors hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {applyingDbPlaceholders ? <RefreshCw size={16} className="animate-spin" /> : <ClipboardCopy size={16} />}
                  生成复制结果
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <AnimatePresence>
        {dbSqlPreviewOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[60] flex flex-col bg-slate-950 text-white"
          >
            <div className="flex min-h-[72px] items-center justify-between border-b border-slate-800 bg-slate-900 px-6">
              <div className="min-w-0">
                <div className="text-xs font-bold uppercase tracking-wider text-emerald-300">数据库模板 SQL</div>
                <h2 className="mt-1 truncate text-lg font-extrabold text-white" title={dbSqlPreviewTitle}>
                  {dbSqlPreviewTitle || '复制内容'}
                </h2>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  onClick={handleCopyDbSqlPreview}
                  disabled={copyingDbSqlPreview || !dbSqlPreviewContent.trim()}
                  className="inline-flex items-center gap-2 rounded-lg bg-emerald-400 px-4 py-2 text-sm font-bold text-slate-950 transition-colors hover:bg-emerald-300 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {copyingDbSqlPreview ? <RefreshCw size={16} className="animate-spin" /> : <ClipboardCopy size={16} />}
                  复制
                </button>
                <button
                  type="button"
                  onClick={() => setDbSqlPreviewOpen(false)}
                  className="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-slate-700 text-slate-300 transition-colors hover:bg-slate-800 hover:text-white"
                  title="关闭"
                >
                  <X size={18} />
                </button>
              </div>
            </div>

            <div className="min-h-0 flex-1 bg-slate-950 p-5">
              <textarea
                value={dbSqlPreviewContent}
                readOnly
                spellCheck={false}
                className="h-full w-full resize-none rounded-lg border border-slate-800 bg-[#101418] p-5 font-mono text-sm font-semibold leading-6 text-slate-100 outline-none selection:bg-emerald-300/25"
              />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

    </div>
  );
};
