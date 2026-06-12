import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, Braces, Check, Copy, Edit2, FileCode, Folder, Plus, RefreshCw, Save, Search, Trash2, Wand2, X } from 'lucide-react';
import Editor from '@monaco-editor/react';
import toast from 'react-hot-toast';
import { getPathGroupDeleteState, getPathGroupSwitchOptions } from './pathGroupActions';
import {
  createProjectInstance,
  deleteProjectInstance,
  getProject,
  getProjectInstanceList,
  updateProjectInstance,
  updateProjectInstanceSelectedPathSet,
  updateProjectSelectedInstance,
} from '@/api/code_generate_project';
import {
  copyPathSet as copyPathSetApi,
  createModel,
  createPath,
  createPathGroup,
  deletePathGroup,
  deletePathSet as deletePathSetApi,
  deletePath,
  getPathGroupList,
  getModelListByPathId,
  getPathList,
  renamePathSet as renamePathSetApi,
  updateModel,
  updatePath,
  updatePathGroup,
  updatePathEnabled,
} from '@/api/path_model';
import {
  parseDbTemplatePlaceholders,
  stringifyDbTemplatePlaceholders,
  type DbTemplatePlaceholder,
} from './dbTemplateCopy';

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

const emptyProjectDraft = (templateProject: any) => ({
  ID: 0,
  templateProjectId: Number(templateProject?.ID || 0),
  projectName: templateProject?.projectName || '',
  diskPath: templateProject?.diskPath || '',
  remark: templateProject?.remark || '',
  userName: templateProject?.userName || 'conchi',
});

const emptyPathDraft = (projectId: number) => ({
  ID: 0,
  projectId,
  projectInstanceId: projectId,
  pathGroupId: 0,
  pathSet: 0,
  fileUrl: '',
  fileName: '',
  dynamicPlaceholders: '',
  enabled: 1,
  incremented: 0,
});

const emptyPathGroupDraft = (projectId: number) => ({
  ID: 0,
  projectId,
  projectInstanceId: projectId,
  pathSet: 0,
  pathSetName: '',
  basePath: '',
  sort: 0,
});

const normalizeRelativeDir = (value: string) => {
  return String(value || '')
    .replace(/\\/g, '/')
    .replace(/\/+/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+$/, '')
    .trim();
};

const normalizeFileName = (value: string) => {
  return String(value || '')
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .trim();
};

const formatPathLabel = (pathObj: any) => {
  const dir = normalizeRelativeDir(pathObj.fileUrl || '');
  const fileName = normalizeFileName(pathObj.fileName || '');
  if (!dir) return fileName || '未命名文件';
  return `${dir}/${fileName}`.replace(/\/+/g, '/');
};

const formatPathFileLabel = (pathObj: any) => {
  const dir = normalizeRelativeDir(pathObj.fileUrl || '');
  const fileName = normalizeFileName(pathObj.fileName || '').replace(/\/+/g, '/');
  if (!fileName) return '未命名文件';
  if (!dir) return fileName;

  const prefix = `${dir}/`.replace(/\/+/g, '/');
  if (fileName.startsWith(prefix)) {
    return fileName.slice(prefix.length) || fileName;
  }

  const fullPath = formatPathLabel(pathObj);
  if (fullPath.startsWith(prefix)) {
    return fullPath.slice(prefix.length) || fileName;
  }

  return fileName;
};

const PATH_PLACEHOLDER_DEFAULTS: Record<string, string> = {
  module: 'btStation',
  Module: 'BtStation',
  moduleName: 'btStation',
  ModuleName: 'BtStation',
  TableName: 'BtStation',
  tableName: 'btStation',
  table_name: 'bt_station',
  TABLE_NAME: 'BT_STATION',
};

const PATH_PLACEHOLDER_DESCRIPTIONS: Record<string, string> = {
  module: '模块名，小驼峰',
  Module: '模块名，大驼峰',
  moduleName: '模块名，小驼峰',
  ModuleName: '模块名，大驼峰',
  TableName: '实体/类名，大驼峰',
  tableName: '实体/变量名，小驼峰',
  table_name: '表名/SQL 名，下划线',
  TABLE_NAME: '常量名，大写下划线',
};

const PATH_FIELD_SNIPPET_PLACEHOLDER_KEYS = new Set([
  'TableName',
  'tableName',
  'kebabTableName',
  'TABLE_NAME',
  'table_name',
  'commentName',
  'javaEntityFields',
  'javaAccessors',
  'javaQueryFields',
  'javaQueryAccessors',
  'tsModelFields',
  'tsQueryFields',
  'vueTableColumns',
  'vueQueryOpts',
  'vueFormItems',
  'vueFormRules',
  'sqlSelectColumns',
  'sqlWhereConditions',
  'sqlCreateColumns',
  'sqlInsertColumns',
  'sqlInsertValues',
  'sqlUpdateAssignments',
  'sqlBatchInsertColumns',
  'sqlBatchInsertValues',
]);

const isPathFieldSnippetScope = (scope: string) => {
  const value = String(scope || '').trim().toLowerCase();
  return value === 'field' || value === 'snippet' || value === 'parsed';
};

const isPathFieldSnippetPlaceholderKey = (key: string) => (
  PATH_FIELD_SNIPPET_PLACEHOLDER_KEYS.has(String(key || '').trim())
);

const extractPathPlaceholdersFromText = (text: any): DbTemplatePlaceholder[] => {
  const raw = String(text || '');
  const keys = new Set<string>();
  [
    /\{\{\s*(?:(manual|field|snippet|parsed)\s*:\s*)?<?\s*([A-Za-z][A-Za-z0-9_]*)\s*>?\s*\}\}/g,
    /\$\{\s*(?:(manual|field|snippet|parsed)\s*:\s*)?<?\s*([A-Za-z][A-Za-z0-9_]*)\s*>?\s*\}/g,
  ].forEach((pattern) => {
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(raw)) !== null) {
      if (isPathFieldSnippetScope(match[1] || '')) continue;
      const key = match[2];
      if (key) keys.add(key);
    }
  });
  const bracketPattern = /\{\[\s*<\s*([A-Za-z][A-Za-z0-9_]*)\s*>\s*\]\}/g;
  let bracketMatch: RegExpExecArray | null;
  while ((bracketMatch = bracketPattern.exec(raw)) !== null) {
    if (bracketMatch[1]) {
      keys.add(bracketMatch[1]);
    }
  }
  return Array.from(keys).map((key) => ({
    key,
    description: PATH_PLACEHOLDER_DESCRIPTIONS[key] || '',
    value: PATH_PLACEHOLDER_DEFAULTS[key] || '',
  }));
};

const mergePathPlaceholders = (items: DbTemplatePlaceholder[]): DbTemplatePlaceholder[] => {
  const merged = new Map<string, DbTemplatePlaceholder>();
  items.forEach((item) => {
    const key = String(item?.key || '').trim();
    if (!key) return;
    const current = merged.get(key);
    if (!current) {
      merged.set(key, {
        key,
        description: String(item.description || PATH_PLACEHOLDER_DESCRIPTIONS[key] || '').trim(),
        value: String(item.value || PATH_PLACEHOLDER_DEFAULTS[key] || '').trim(),
      });
      return;
    }
    if (!current.description && item.description) current.description = item.description;
    if (!current.value && item.value) current.value = item.value;
  });
  return Array.from(merged.values());
};

const getLanguageType = (filename: string) => {
  const fn = (filename || '').toLowerCase();
  if (fn.includes('.json')) return 'json';
  if (fn.includes('.sh') || fn.includes('.bash')) return 'shell';
  if (fn.includes('.py')) return 'python';
  if (fn.includes('.js') || fn.includes('.ts') || fn.includes('.vue')) return 'javascript';
  if (fn.includes('.java')) return 'java';
  if (fn.includes('.go')) return 'go';
  if (fn.includes('.xml') || fn.includes('.html')) return 'xml';
  if (fn.includes('.md')) return 'markdown';
  if (fn.includes('.yaml') || fn.includes('.yml')) return 'yaml';
  if (fn.includes('.sql')) return 'sql';
  return 'plaintext';
};

const replacePathPrefix = (value: string, oldPrefix: string, nextPrefix: string) => {
  const currentValue = normalizeRelativeDir(value || '');
  const currentPrefix = normalizeRelativeDir(oldPrefix || '');
  const targetPrefix = normalizeRelativeDir(nextPrefix || '');

  if (!currentValue || !currentPrefix || currentPrefix === '.') return currentValue;
  if (currentValue === currentPrefix) return targetPrefix;
  if (currentValue.startsWith(`${currentPrefix}/`)) {
    const suffix = currentValue.slice(currentPrefix.length + 1);
    return targetPrefix ? `${targetPrefix}/${suffix}`.replace(/\/+/g, '/') : suffix;
  }
  return currentValue;
};

const getCommonPathPrefix = (rows: any[]) => {
  const dirs = rows
    .map((item) => normalizeRelativeDir(item.fileUrl || ''))
    .filter(Boolean);

  if (dirs.length === 0) return '.';
  if (dirs.length === 1) return dirs[0] || '.';

  const commonParts = dirs[0].split('/').filter(Boolean);
  dirs.slice(1).forEach((dir) => {
    const parts = dir.split('/').filter(Boolean);
    let index = 0;
    while (index < commonParts.length && index < parts.length && commonParts[index] === parts[index]) {
      index += 1;
    }
    commonParts.length = index;
  });

  return commonParts.join('/') || '.';
};

const getPathParentSuffix = (pathObj: any, basePath: string) => {
  const dir = normalizeRelativeDir(pathObj.fileUrl || '');
  const prefix = normalizeRelativeDir(basePath || '');
  if (!dir) return '/';
  if (!prefix || prefix === '.') return dir;
  if (dir === prefix) return '/';
  if (dir.startsWith(`${prefix}/`)) {
    return dir.slice(prefix.length + 1) || '/';
  }
  return dir;
};

const buildPathParentWithSuffix = (basePath: string, rawSuffix: string) => {
  const base = normalizeRelativeDir(basePath || '');
  const suffixText = String(rawSuffix || '').trim();
  const suffix = suffixText === '/' ? '' : normalizeRelativeDir(suffixText);

  if (!base || base === '.') return suffix;
  return suffix ? `${base}/${suffix}`.replace(/\/+/g, '/') : base;
};

const serviceSegmentPattern = /(^|[-_])(api|biz|service|server|ui|web|admin|gateway|job|task|client|frontend|backend)$/i;

const inferServiceBasePath = (pathObj: any) => {
  const dir = normalizeRelativeDir(pathObj.fileUrl || '');
  const fileName = normalizeFileName(pathObj.fileName || '');
  const combined = `${dir}/${fileName}`.replace(/\/+/g, '/').replace(/\/+$/, '');
  const parts = combined.split('/').filter(Boolean);

  if (parts.length === 0) return '.';
  if (parts[0] === '..' || parts[0] === '.') return parts[0];

  if (parts.length >= 3 && /(^|[-_])service$/i.test(parts[0]) && serviceSegmentPattern.test(parts[1]) && parts[2] === 'src') {
    return parts.slice(0, 3).join('/');
  }

  if (parts.length >= 2 && /(^|[-_])service$/i.test(parts[0]) && serviceSegmentPattern.test(parts[1])) {
    return parts.slice(0, 2).join('/');
  }

  const moduleIndex = parts.findIndex((part, index) => index > 0 && index <= 3 && serviceSegmentPattern.test(part));
  if (moduleIndex > 0) {
    return parts.slice(0, moduleIndex + 1).join('/');
  }

  if (parts[0] === 'src' && parts.length >= 3 && ['api', 'views', 'pages', 'components'].includes(parts[1])) {
    return parts.slice(0, 3).join('/');
  }

  if (parts[0] === 'src' && parts.length >= 2) {
    return parts.slice(0, 2).join('/');
  }

  return parts[0];
};

const getServiceName = (basePath: string) => {
  if (!basePath || basePath === '.') return '项目根目录';
  if (basePath === '..') return '上级目录';
  const parts = basePath.split('/').filter(Boolean);
  return parts[parts.length - 1] || basePath;
};

const getPathSet = (pathObj: any) => Number(pathObj?.pathSet ?? 0);

const makePathSetSectionKey = (pathSet: number, copyIndex = 0) => {
  return copyIndex > 0 ? `path-set-${pathSet}-copy-${copyIndex}` : `path-set-${pathSet}`;
};

const makePathGroupEditKey = (pathSet: number, groupKey: string) => `${pathSet}::${groupKey}`;

const getPathCopySignature = (pathObj: any) => {
  return [
    normalizeRelativeDir(pathObj.fileUrl || ''),
    normalizeFileName(pathObj.fileName || ''),
    Number(pathObj.enabled ?? 1),
    Number(pathObj.incremented ?? 0),
  ].join('\u001f');
};

const splitLegacyCopiedRows = (rows: any[]) => {
  const sortedRows = [...rows].sort((a, b) => Number(a.ID || 0) - Number(b.ID || 0));
  const signatureMap = new Map<string, any[]>();
  sortedRows.forEach((pathObj) => {
    const signature = getPathCopySignature(pathObj);
    const current = signatureMap.get(signature) || [];
    current.push(pathObj);
    signatureMap.set(signature, current);
  });

  const groups = Array.from(signatureMap.values());
  const maxCopies = groups.reduce((max, group) => Math.max(max, group.length), 1);
  const duplicatedRowsCount = groups.reduce((total, group) => total + (group.length > 1 ? group.length : 0), 0);
  if (maxCopies <= 1 || duplicatedRowsCount < sortedRows.length * 0.5) {
    return [sortedRows];
  }

  const splitRows = Array.from({ length: maxCopies }, () => [] as any[]);
  groups.forEach((group) => {
    group.sort((a, b) => Number(a.ID || 0) - Number(b.ID || 0));
    group.forEach((pathObj, index) => {
      splitRows[index].push(pathObj);
    });
  });

  return splitRows
    .map((group) => group.sort((a, b) => Number(a.ID || 0) - Number(b.ID || 0)))
    .filter((group) => group.length > 0);
};

const getPathGroupId = (pathObj: any) => Number(pathObj?.pathGroupId || pathObj?.path_group_id || 0);

const makeStoredPathGroupKey = (group: any) => `group-${Number(group?.ID || 0)}`;

const makeLegacyPathGroupKey = (pathSet: number, basePath: string) => `legacy-${pathSet}-${basePath || '.'}`;

const buildPathGroupStats = (group: any, rows: any[]) => {
  const pathsInGroup = Array.isArray(rows) ? rows : [];
  return {
    ...group,
    paths: pathsInGroup,
    firstId: pathsInGroup.reduce((min, pathObj) => Math.min(min, Number(pathObj.ID || min)), Number.MAX_SAFE_INTEGER),
    enabledCount: pathsInGroup.filter((pathObj) => Number(pathObj.enabled || 0) === 1).length,
    incrementedCount: pathsInGroup.filter((pathObj) => Number(pathObj.incremented || 0) === 1).length,
  };
};

const choosePathGroupBasePath = (inferredKey: string, rows: any[]) => {
  const commonPrefix = getCommonPathPrefix(rows);
  const key = normalizeRelativeDir(inferredKey || '');
  if (key && key.includes('/src') && (commonPrefix === key || commonPrefix.startsWith(`${key}/`))) {
    return key;
  }
  return commonPrefix;
};

const buildPathGroups = (rows: any[], storedGroups: any[] = [], pathSet = 0) => {
  const rowMap = new Map<string, any[]>();
  rows.forEach((pathObj) => {
    const groupId = getPathGroupId(pathObj);
    const key = groupId > 0 ? `group-${groupId}` : makeLegacyPathGroupKey(pathSet, inferServiceBasePath(pathObj));
    const current = rowMap.get(key) || [];
    current.push(pathObj);
    rowMap.set(key, current);
  });

  const groups = storedGroups.map((group) => {
    const basePath = normalizeRelativeDir(group.basePath || '') || '.';
    const key = makeStoredPathGroupKey(group);
    const groupRows = rowMap.get(key) || [];
    rowMap.delete(key);
    return buildPathGroupStats({
      ...group,
      key,
      basePath,
      serviceName: getServiceName(basePath),
      pathSet: Number(group.pathSet || pathSet || 0),
      sort: Number(group.sort || 0),
    }, groupRows);
  });

  rowMap.forEach((legacyRows, key) => {
    const inferredKey = legacyRows.length > 0 ? inferServiceBasePath(legacyRows[0]) : key.replace(/^legacy-\d+-/, '');
    const basePath = choosePathGroupBasePath(inferredKey, legacyRows);
    groups.push(buildPathGroupStats({
      ID: 0,
      key,
      basePath,
      serviceName: getServiceName(basePath),
      pathSet,
      sort: 0,
      isLegacy: true,
    }, legacyRows));
  });

  return groups.sort((a, b) => {
    if (a.basePath === '..') return -1;
    if (b.basePath === '..') return 1;
    const sortDiff = Number(a.sort || 0) - Number(b.sort || 0);
    if (sortDiff !== 0) return sortDiff;
    return Number(a.firstId || Number.MAX_SAFE_INTEGER) - Number(b.firstId || Number.MAX_SAFE_INTEGER);
  });
};

const getStoredPathSetName = (rows: any[]) => {
  const namedRow = rows.find((pathObj) => String(pathObj?.pathSetName || '').trim());
  return String(namedRow?.pathSetName || '').trim();
};

const getStoredGroupPathSetName = (groups: any[]) => {
  const namedGroup = groups.find((group) => String(group?.pathSetName || '').trim());
  return String(namedGroup?.pathSetName || '').trim();
};

const buildPathSetSections = (rows: any[], storedGroups: any[] = []) => {
  const setMap = new Map<number, any>();

  rows.forEach((pathObj) => {
    const pathSet = getPathSet(pathObj);
    const current = setMap.get(pathSet) || {
      key: makePathSetSectionKey(pathSet),
      pathSet,
      paths: [],
    };
    current.paths.push(pathObj);
    setMap.set(pathSet, current);
  });

  storedGroups.forEach((group) => {
    const pathSet = Number(group?.pathSet ?? 0);
    const current = setMap.get(pathSet) || {
      key: makePathSetSectionKey(pathSet),
      pathSet,
      paths: [],
    };
    current.groups = [...(current.groups || []), group];
    setMap.set(pathSet, current);
  });

  if (!setMap.has(0)) {
    setMap.set(0, {
      key: makePathSetSectionKey(0),
      pathSet: 0,
      paths: [],
    });
  }

  return Array.from(setMap.values()).sort((a, b) => {
    if (a.pathSet === 0) return -1;
    if (b.pathSet === 0) return 1;
    return Number(a.pathSet || 0) - Number(b.pathSet || 0);
  }).flatMap((section) => {
    const splitRows = Number(section.pathSet || 0) === 0 && !(section.groups || []).length
      ? splitLegacyCopiedRows(section.paths)
      : [section.paths];
    return splitRows.map((pathsInSection, copyIndex) => ({
      ...section,
      key: makePathSetSectionKey(Number(section.pathSet || 0), copyIndex),
      copyIndex,
      paths: pathsInSection,
      pathSetName: getStoredPathSetName(pathsInSection),
      storedPathGroups: section.groups || [],
      isPrimary: Number(section.pathSet || 0) === 0 && copyIndex === 0,
      pathGroups: buildPathGroups(pathsInSection, section.groups || [], Number(section.pathSet || 0)),
    }));
  }).map((section, index) => ({
    ...section,
    pathSetName: section.pathSetName || getStoredGroupPathSetName(section.storedPathGroups || []),
    displayIndex: index,
  }));
};

const getPathSetIdentityKey = (section: any) => {
  if (section?.isPrimary) return 'path-set-primary';
  const pathSet = Number(section?.pathSet || 0);
  if (pathSet > 0) return `path-set-${pathSet}`;
  const ids = (section?.paths || [])
    .map((pathObj: any) => Number(pathObj.ID || 0))
    .filter(Boolean)
    .sort((a: number, b: number) => a - b)
    .join(',');
  return `path-set-0-copy-${ids || Number(section?.copyIndex || 0)}`;
};

const applyStablePathSetDisplayNumbers = (sections: any[], displayNumberMap: Map<string, number>) => {
  const realPathSetNumbers = new Set(
    sections
      .map((section) => Number(section?.pathSet || 0))
      .filter((pathSet) => pathSet > 0),
  );
  const usedNumbers = new Set(realPathSetNumbers);
  const activeLegacyKeys = new Set<string>();

  const nextSections = sections.map((section) => {
    const identityKey = getPathSetIdentityKey(section);
    if (section?.isPrimary) {
      return { ...section, identityKey, displayNumber: 0 };
    }

    const pathSet = Number(section?.pathSet || 0);
    if (pathSet > 0) {
      return { ...section, identityKey, displayNumber: pathSet };
    }

    activeLegacyKeys.add(identityKey);
    let displayNumber = displayNumberMap.get(identityKey);
    if (!displayNumber) {
      displayNumber = Math.max(Number(section?.copyIndex || section?.displayIndex || 1), 1);
      while (usedNumbers.has(displayNumber)) displayNumber += 1;
      displayNumberMap.set(identityKey, displayNumber);
    }
    usedNumbers.add(displayNumber);

    return { ...section, identityKey, displayNumber };
  });

  displayNumberMap.forEach((_, key) => {
    if (key.startsWith('path-set-0-copy-') && !activeLegacyKeys.has(key)) {
      displayNumberMap.delete(key);
    }
  });

  return nextSections;
};

const getPathSetTitle = (section: any) => {
  const customName = String(section?.pathSetName || '').trim();
  if (customName) return customName;
  if (section?.isPrimary) return '相对路径';
  const displayNumber = Number(section?.displayNumber || 0);
  if (displayNumber > 0) return `相对路径副本 ${displayNumber}`;
  const pathSet = Number(section?.pathSet || 0);
  if (pathSet > 0) return `相对路径副本 ${pathSet}`;
  return `相对路径副本 ${Number(section?.copyIndex || section?.displayIndex || 0)}`;
};

type ProjectConfigDialogProps = {
  project: any;
  initialProjectInstanceId?: number | null;
  initialPathSetKey?: string | null;
  initialPathSet?: number | null;
  initialPathGroupKey?: string | null;
  onClose: () => void;
  onProjectSaved?: () => Promise<void> | void;
};

export default function ProjectConfigDialog({
  project: templateProject,
  initialProjectInstanceId,
  initialPathSetKey,
  initialPathSet,
  initialPathGroupKey,
  onClose,
  onProjectSaved,
}: ProjectConfigDialogProps) {
  const navigate = useNavigate();
  const templateProjectId = Number(templateProject?.ID || 0);
  const [projectInstances, setProjectInstances] = useState<any[]>([]);
  const [selectedProject, setSelectedProject] = useState<any>(null);
  const projectId = Number(selectedProject?.ID || 0);
  const [projectDraft, setProjectDraft] = useState<any>(emptyProjectDraft(templateProject));
  const [projectEditorOpen, setProjectEditorOpen] = useState(false);
  const [paths, setPaths] = useState<any[]>([]);
  const [pathGroups, setPathGroups] = useState<any[]>([]);
  const [pathDraft, setPathDraft] = useState<any>(emptyPathDraft(0));
  const [pathEditorOpen, setPathEditorOpen] = useState(false);
  const [pathGroupDraft, setPathGroupDraft] = useState<any>(emptyPathGroupDraft(0));
  const [pathGroupEditorOpen, setPathGroupEditorOpen] = useState(false);
  const [contentEditorOpen, setContentEditorOpen] = useState(false);
  const [contentPath, setContentPath] = useState<any>(null);
  const [contentModel, setContentModel] = useState<any>(null);
  const [contentDraft, setContentDraft] = useState('');
  const [pathGroupEdits, setPathGroupEdits] = useState<Record<string, string>>({});
  const [pathSetNameEdits, setPathSetNameEdits] = useState<Record<string, string>>({});
  const [editingPathSetNameKey, setEditingPathSetNameKey] = useState<string | null>(null);
  const [pathSetActionConfirm, setPathSetActionConfirm] = useState<{ action: 'copy' | 'delete'; section: any } | null>(null);
  const [selectedPathSetKey, setSelectedPathSetKey] = useState<string | null>(null);
  const [activePathSetKey, setActivePathSetKey] = useState<string | null>(null);
  const [activePathGroupKey, setActivePathGroupKey] = useState<string | null>(null);
  const [pathSearch, setPathSearch] = useState('');
  const [promptSummaryOpen, setPromptSummaryOpen] = useState(false);
  const [promptSummaryPath, setPromptSummaryPath] = useState<any | null>(null);
  const [promptModel, setPromptModel] = useState<any | null>(null);
  const [promptDraft, setPromptDraft] = useState('');
  const [placeholderPath, setPlaceholderPath] = useState<any | null>(null);
  const [placeholderRows, setPlaceholderRows] = useState<DbTemplatePlaceholder[]>([]);
  const [loadingPlaceholders, setLoadingPlaceholders] = useState(false);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [loadingPaths, setLoadingPaths] = useState(false);
  const [savingProject, setSavingProject] = useState(false);
  const [savingPath, setSavingPath] = useState(false);
  const [savingPathGroup, setSavingPathGroup] = useState(false);
  const [loadingContent, setLoadingContent] = useState(false);
  const [savingContent, setSavingContent] = useState(false);
  const [loadingPrompt, setLoadingPrompt] = useState(false);
  const [savingPrompt, setSavingPrompt] = useState(false);
  const [savingPlaceholders, setSavingPlaceholders] = useState(false);
  const [savingPathGroupKey, setSavingPathGroupKey] = useState<string | null>(null);
  const [savingPathSetNameKey, setSavingPathSetNameKey] = useState<string | null>(null);
  const [copyingPathSet, setCopyingPathSet] = useState<number | null>(null);
  const [deletingProjectId, setDeletingProjectId] = useState<number | null>(null);
  const [deletingPathId, setDeletingPathId] = useState<number | null>(null);
  const [deletingPathSet, setDeletingPathSet] = useState<number | null>(null);
  const pathSetDisplayNumberRef = useRef<Map<string, number>>(new Map());
  const cancelPathSetNameSaveRef = useRef<Set<string>>(new Set());
  const initialPathDetailAppliedRef = useRef('');

  const pathPlaceholderOptions = useMemo(() => (
    mergePathPlaceholders([
      ...paths.flatMap((pathObj) => parseDbTemplatePlaceholders(pathObj?.dynamicPlaceholders)),
      ...placeholderRows,
    ]).filter((item) => !isPathFieldSnippetPlaceholderKey(item.key))
  ), [paths, placeholderRows]);

  useEffect(() => {
    const previousBodyOverflow = document.body.style.overflow;
    const previousHtmlOverflow = document.documentElement.style.overflow;
    document.body.style.overflow = 'hidden';
    document.documentElement.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousBodyOverflow;
      document.documentElement.style.overflow = previousHtmlOverflow;
    };
  }, []);

  const persistSelectedProjectInstance = async (nextProjectId: number) => {
    if (!templateProjectId || !nextProjectId) return;
    try {
      await updateProjectSelectedInstance({
        templateProjectId,
        projectInstanceId: nextProjectId,
      });
    } catch (e) {
      toast.error('保存项目选中状态失败');
    }
  };

  const fetchProjectInstances = async (preferredProjectId?: number, ensureDefault = true) => {
    if (!templateProjectId) {
      setProjectInstances([]);
      setSelectedProject(null);
      setProjectDraft(emptyProjectDraft(templateProject));
      setPaths([]);
      setPathGroups([]);
      return;
    }

    setLoadingProjects(true);
    try {
      const res: any = await getProjectInstanceList(templateProjectId, ensureDefault);
      let rows = unwrapResponseData(res);
      if (!Array.isArray(rows)) rows = [];
      rows = rows.filter((item: any) => Number(item.templateProjectId || 0) === templateProjectId);
      setProjectInstances(rows);

      let persistedProjectId = Number(templateProject?.selectedProjectInstanceId || 0);
      if (!preferredProjectId) {
        try {
          const projectRes: any = await getProject(templateProjectId);
          const latestTemplateProject = unwrapResponseData(projectRes);
          persistedProjectId = Number(latestTemplateProject?.selectedProjectInstanceId || persistedProjectId || 0);
        } catch (e) {
          // Keep the project list usable even if the selection lookup fails.
        }
      }

      const nextProject =
        rows.find((item: any) => Number(item.ID || 0) === Number(preferredProjectId || 0)) ||
        rows.find((item: any) => Number(item.ID || 0) === persistedProjectId) ||
        rows.find((item: any) => Number(item.ID || 0) === projectId) ||
        rows[0] ||
        null;
      if (preferredProjectId && nextProject?.ID) {
        persistSelectedProjectInstance(Number(nextProject.ID));
      }
      setSelectedProject(nextProject);
      setProjectDraft(nextProject || emptyProjectDraft(templateProject));
      setPathDraft(emptyPathDraft(Number(nextProject?.ID || 0)));
      setPathGroupDraft(emptyPathGroupDraft(Number(nextProject?.ID || 0)));
      setPathEditorOpen(false);
      setPathGroupEditorOpen(false);
      setContentEditorOpen(false);
      setContentPath(null);
      setContentModel(null);
      setContentDraft('');
      setSelectedPathSetKey(null);
      setActivePathSetKey(null);
      setActivePathGroupKey(null);
      setPathGroupEdits({});
      setPathSetNameEdits({});
      setEditingPathSetNameKey(null);
      setPromptSummaryOpen(false);
      setPromptSummaryPath(null);
      setPromptModel(null);
      setPromptDraft('');
    } catch (e) {
      toast.error('加载项目列表失败');
    } finally {
      setLoadingProjects(false);
    }
  };

  useEffect(() => {
    initialPathDetailAppliedRef.current = '';
    setProjectInstances([]);
    setSelectedProject(null);
    setProjectDraft(emptyProjectDraft(templateProject));
    setPaths([]);
    setPathGroups([]);
    setPathDraft(emptyPathDraft(0));
    setPathGroupDraft(emptyPathGroupDraft(0));
    setSelectedPathSetKey(null);
    setActivePathSetKey(null);
    setActivePathGroupKey(null);
    setProjectEditorOpen(false);
    setPathEditorOpen(false);
    setPathGroupEditorOpen(false);
    setContentEditorOpen(false);
    setContentPath(null);
    setContentModel(null);
    setContentDraft('');
    setPathGroupEdits({});
    setPathSetNameEdits({});
    setEditingPathSetNameKey(null);
    setPromptSummaryOpen(false);
    setPromptSummaryPath(null);
    setPromptModel(null);
    setPromptDraft('');
    fetchProjectInstances(Number(initialProjectInstanceId || 0) || undefined, true);
  }, [templateProjectId, initialProjectInstanceId, initialPathSetKey, initialPathSet, initialPathGroupKey]);

  const fetchPathGroups = async () => {
    if (!projectId) {
      setPathGroups([]);
      return;
    }
    const res: any = await getPathGroupList(projectId);
    let rows = unwrapResponseData(res);
    if (!Array.isArray(rows)) rows = [];
    setPathGroups(rows.filter((item: any) => Number(item.projectInstanceId || 0) === projectId));
  };

  const fetchPaths = async () => {
    if (!projectId) {
      setPaths([]);
      return;
    }
    const res: any = await getPathList(projectId);
    let rows = unwrapResponseData(res);
    if (!Array.isArray(rows)) rows = [];
    setPaths(rows.filter((item: any) => Number(item.projectInstanceId || 0) === projectId));
  };

  const fetchPathConfig = async () => {
    if (!projectId) {
      setPathGroups([]);
      setPaths([]);
      return;
    }
    setLoadingPaths(true);
    try {
      await fetchPathGroups();
      await fetchPaths();
    } catch (e) {
      toast.error('加载相对路径失败');
    } finally {
      setLoadingPaths(false);
    }
  };

  useEffect(() => {
    fetchPathConfig();
  }, [projectId]);

  useEffect(() => {
    pathSetDisplayNumberRef.current.clear();
  }, [projectId]);

  const projectList = useMemo(() => {
    return [...projectInstances].sort((a, b) => Number(a.ID || 0) - Number(b.ID || 0));
  }, [projectInstances]);

  const pathSetSections = useMemo(() => {
    return applyStablePathSetDisplayNumbers(buildPathSetSections(paths, pathGroups), pathSetDisplayNumberRef.current);
  }, [paths, pathGroups]);

  const primaryPathSetSection = useMemo(() => {
    return pathSetSections.find((item) => item.isPrimary) || pathSetSections[0] || null;
  }, [pathSetSections]);

  const selectedPathSetSection = useMemo(() => {
    if (!selectedPathSetKey) return primaryPathSetSection;
    return pathSetSections.find((item) => item.key === selectedPathSetKey) || primaryPathSetSection;
  }, [pathSetSections, primaryPathSetSection, selectedPathSetKey]);

  const totalPathGroupCount = useMemo(() => {
    return pathSetSections.reduce((total, section) => total + section.pathGroups.length, 0);
  }, [pathSetSections]);

  const activePathSetSection = useMemo(() => {
    if (!activePathSetKey) return null;
    return pathSetSections.find((item) => item.key === activePathSetKey) || null;
  }, [pathSetSections, activePathSetKey]);

  const activePathGroup = useMemo(() => {
    if (!activePathSetSection || !activePathGroupKey) return null;
    return activePathSetSection.pathGroups.find((item: any) => item.key === activePathGroupKey) || null;
  }, [activePathSetSection, activePathGroupKey]);

  const pathDetailOpen = Boolean(activePathSetKey) && Boolean(activePathGroupKey);

  const filteredPaths = useMemo(() => {
    const keyword = pathSearch.trim().toLowerCase();
    const activeGroupRows = pathDetailOpen
      ? (activePathGroup?.paths || [])
      : [];
    const rows = [...activeGroupRows].sort((a, b) => Number(a.ID || 0) - Number(b.ID || 0));
    if (!keyword) return rows;
    return rows.filter((item) => {
      return (
        String(item.fileUrl || '').toLowerCase().includes(keyword) ||
        String(item.fileName || '').toLowerCase().includes(keyword) ||
        formatPathFileLabel(item).toLowerCase().includes(keyword) ||
        formatPathLabel(item).toLowerCase().includes(keyword)
      );
    });
  }, [activePathGroup, pathSearch, pathDetailOpen]);

  const pathGroupSwitchOptions = useMemo(() => {
    return getPathGroupSwitchOptions(activePathSetSection?.pathGroups || []);
  }, [activePathSetSection]);

  const filteredPathSetSections = useMemo(() => {
    const keyword = pathSearch.trim().toLowerCase();
    if (!keyword) return pathSetSections;
    return pathSetSections.map((section) => {
      const sectionMatches = getPathSetTitle(section).toLowerCase().includes(keyword);
      const pathGroups = section.pathGroups.filter((group: any) => (
        group.basePath.toLowerCase().includes(keyword) ||
        group.serviceName.toLowerCase().includes(keyword) ||
        group.paths.some((item: any) => (
          formatPathFileLabel(item).toLowerCase().includes(keyword) ||
          formatPathLabel(item).toLowerCase().includes(keyword)
        ))
      ));

      if (sectionMatches) return section;
      if (pathGroups.length > 0) return { ...section, pathGroups };
      return null;
    }).filter(Boolean);
  }, [pathSetSections, pathSearch]);

  useEffect(() => {
    if (!activePathSetKey) return;
    const section = pathSetSections.find((item) => item.key === activePathSetKey);
    if (!section || (activePathGroupKey && !section.pathGroups.some((item: any) => item.key === activePathGroupKey))) {
      setActivePathSetKey(null);
      setActivePathGroupKey(null);
    }
  }, [activePathSetKey, activePathGroupKey, pathSetSections]);

  useEffect(() => {
    const targetGroupKey = String(initialPathGroupKey || '').trim();
    if (!projectId || !targetGroupKey || pathSetSections.length === 0) return;

    const targetPathSetKey = String(initialPathSetKey || '').trim();
    const targetPathSet = Number(initialPathSet || 0);
    const applyKey = `${projectId}::${targetPathSetKey}::${targetPathSet}::${targetGroupKey}`;
    if (initialPathDetailAppliedRef.current === applyKey) return;

    const nextSection =
      (targetPathSetKey
        ? pathSetSections.find((section) => (
          section.key === targetPathSetKey &&
          section.pathGroups.some((group: any) => group.key === targetGroupKey)
        ))
        : null) ||
      pathSetSections.find((section) => (
        Number(section.pathSet || 0) === targetPathSet &&
        section.pathGroups.some((group: any) => group.key === targetGroupKey)
      )) ||
      pathSetSections.find((section) => section.pathGroups.some((group: any) => group.key === targetGroupKey));

    const nextGroup = nextSection?.pathGroups.find((group: any) => group.key === targetGroupKey);
    if (!nextSection || !nextGroup) return;

    setSelectedPathSetKey(nextSection.key);
    setActivePathSetKey(nextSection.key);
    setActivePathGroupKey(nextGroup.key);
    initialPathDetailAppliedRef.current = applyKey;
  }, [projectId, initialPathSetKey, initialPathSet, initialPathGroupKey, pathSetSections]);

  const persistSelectedPathSetIdentity = async (identity: string) => {
    if (!projectId || !identity) return;
    try {
      await updateProjectInstanceSelectedPathSet({
        projectInstanceId: projectId,
        selectedPathSetIdentity: identity,
      });
      setSelectedProject((prev: any) => (
        Number(prev?.ID || 0) === projectId ? { ...prev, selectedPathSetIdentity: identity } : prev
      ));
      setProjectInstances((prev) => prev.map((item) => (
        Number(item.ID || 0) === projectId ? { ...item, selectedPathSetIdentity: identity } : item
      )));
    } catch (e) {
      toast.error('保存相对路径选中状态失败');
    }
  };

  useEffect(() => {
    if (!projectId || pathSetSections.length === 0) {
      setSelectedPathSetKey(null);
      return;
    }

    const selectedPathSetIdentity = String(selectedProject?.selectedPathSetIdentity || '').trim();
    const storedSection = selectedPathSetIdentity
      ? pathSetSections.find((item) => item.identityKey === selectedPathSetIdentity || item.key === selectedPathSetIdentity)
      : null;
    if (storedSection && selectedPathSetKey !== storedSection.key) {
      setSelectedPathSetKey(storedSection.key);
      return;
    }

    if (selectedPathSetKey && pathSetSections.some((item) => item.key === selectedPathSetKey)) {
      return;
    }

    setSelectedPathSetKey((storedSection || pathSetSections[0]).key);
  }, [projectId, selectedProject?.selectedPathSetIdentity, pathSetSections, selectedPathSetKey]);

  const selectPathSetSection = (section: any) => {
    if (!section?.key) return;
    const identity = section.identityKey || section.key;
    setSelectedPathSetKey(section.key);
    if (String(selectedProject?.selectedPathSetIdentity || '').trim() !== identity) {
      persistSelectedPathSetIdentity(identity);
    }
  };

  const selectProject = (nextProject: any) => {
    const nextProjectId = Number(nextProject?.ID || 0);
    if (nextProjectId) {
      persistSelectedProjectInstance(nextProjectId);
    }
    setSelectedProject(nextProject);
    setProjectDraft(nextProject);
    setPathDraft(emptyPathDraft(nextProjectId));
    setPathGroupDraft(emptyPathGroupDraft(nextProjectId));
    setSelectedPathSetKey(null);
    setActivePathSetKey(null);
    setActivePathGroupKey(null);
    setProjectEditorOpen(false);
    setPathEditorOpen(false);
    setPathGroupEditorOpen(false);
    setContentEditorOpen(false);
    setContentPath(null);
    setContentModel(null);
    setContentDraft('');
    setPathGroupEdits({});
    setPathSetNameEdits({});
    setEditingPathSetNameKey(null);
    setPromptSummaryOpen(false);
    setPromptSummaryPath(null);
    setPromptModel(null);
    setPromptDraft('');
  };

  const openCreateProject = () => {
    setProjectDraft(emptyProjectDraft(templateProject));
    setProjectEditorOpen(true);
  };

  const openProjectEditor = (nextProject = selectedProject) => {
    setSelectedProject(nextProject);
    setProjectDraft(nextProject || emptyProjectDraft(templateProject));
    setProjectEditorOpen(true);
  };

  const openCreatePathGroup = (section = selectedPathSetSection) => {
    if (!projectId) {
      toast.error('请先新增项目');
      return;
    }
    setPathGroupDraft({
      ...emptyPathGroupDraft(projectId),
      pathSet: Number(section?.pathSet || 0),
      pathSetName: String(section?.pathSetName || '').trim(),
    });
    setPathGroupEditorOpen(true);
  };

  const openCreatePath = (pathSet = 0, basePath?: string) => {
    if (!projectId) {
      toast.error('请先新增项目');
      return;
    }
    const targetSection = pathSetSections.find((section) => Number(section.pathSet || 0) === Number(pathSet || 0));
    const resolvedGroup =
      (activePathGroup && Number(activePathSetSection?.pathSet || 0) === Number(pathSet || 0) ? activePathGroup : null) ||
      (basePath ? targetSection?.pathGroups?.find((group: any) => normalizeRelativeDir(group.basePath || '') === normalizeRelativeDir(basePath)) : null) ||
      targetSection?.pathGroups?.[0] ||
      null;
    const firstBasePath = resolvedGroup?.basePath || '';
    setPathDraft({
      ...emptyPathDraft(projectId),
      pathSet: Number(pathSet || 0),
      pathSetName: targetSection?.pathSetName || '',
      pathGroupId: Number(resolvedGroup?.ID || 0),
      fileUrl: basePath ?? firstBasePath,
    });
    setPathEditorOpen(true);
  };

  const openEditPath = (pathObj: any) => {
    setPathDraft({ ...pathObj });
    setPathEditorOpen(true);
  };

  const switchActivePathGroup = (nextGroupKey: string) => {
    if (!nextGroupKey || nextGroupKey === activePathGroupKey) return;
    const nextGroup = activePathSetSection?.pathGroups.find((group: any) => group.key === nextGroupKey);
    if (!nextGroup) {
      toast.error('未找到可切换的公共相对路径');
      return;
    }
    const currentSuffix = getPathParentSuffix(pathDraft, activePathGroup?.basePath || '');
    setActivePathGroupKey(nextGroup.key);
    setPathDraft({
      ...pathDraft,
      pathGroupId: Number(nextGroup.ID || 0),
      fileUrl: buildPathParentWithSuffix(nextGroup.basePath || '', currentSuffix),
    });
  };

  const openPathPromptDialog = async (pathObj: any) => {
    if (Number(pathObj?.enabled || 0) !== 1) {
      toast.error('停用文件不能编辑提示词');
      return;
    }
    setPromptSummaryPath(pathObj);
    setPromptModel(null);
    setPromptDraft('');
    setPromptSummaryOpen(true);
    setLoadingPrompt(true);
    try {
      const res: any = await getModelListByPathId(Number(pathObj.ID || 0));
      let models = unwrapResponseData(res);
      if (!Array.isArray(models)) models = [];
      models = models.filter((item: any) => Number(item.pathId || 0) === Number(pathObj.ID || 0));
      const model = models[0] || null;
      setPromptModel(model);
      setPromptDraft(model?.prompt || '');
    } catch (e) {
      toast.error('加载提示词失败');
    } finally {
      setLoadingPrompt(false);
    }
  };

  const savePathPrompt = async () => {
    if (!promptSummaryPath?.ID) return;
    setSavingPrompt(true);
    try {
      if (promptModel?.ID) {
        const payload = { ...promptModel, prompt: promptDraft };
        await updateModel(payload);
        setPromptModel(payload);
        toast.success('提示词已保存');
      } else {
        await createModel({ pathId: Number(promptSummaryPath.ID), content: '', prompt: promptDraft });
        const res: any = await getModelListByPathId(Number(promptSummaryPath.ID));
        let models = unwrapResponseData(res);
        if (!Array.isArray(models)) models = [];
        models = models.filter((item: any) => Number(item.pathId || 0) === Number(promptSummaryPath.ID));
        setPromptModel(models[0] || null);
        toast.success('提示词已初始化');
      }
    } catch (e) {
      toast.error('保存提示词失败');
    } finally {
      setSavingPrompt(false);
    }
  };

  const openPathContentEditor = async (pathObj: any) => {
    setContentPath(pathObj);
    setContentModel(null);
    setContentDraft('');
    setContentEditorOpen(true);
    setLoadingContent(true);
    try {
      const res: any = await getModelListByPathId(Number(pathObj.ID || 0));
      let models = unwrapResponseData(res);
      if (!Array.isArray(models)) models = [];
      models = models.filter((item: any) => Number(item.pathId || 0) === Number(pathObj.ID || 0));
      const model = models[0] || null;
      setContentModel(model);
      setContentDraft(model?.content || '');
    } catch (e) {
      toast.error('加载文件内容失败');
    } finally {
      setLoadingContent(false);
    }
  };

  const saveProject = async () => {
    const projectName = String(projectDraft.projectName || '').trim();
    if (!projectName) {
      toast.error('项目名称不能为空');
      return;
    }

    setSavingProject(true);
    try {
      const payload = {
        ...projectDraft,
        templateProjectId,
        projectName,
        diskPath: String(projectDraft.diskPath || '').trim(),
        remark: String(projectDraft.remark || '').trim(),
      };
      const res: any = payload.ID
        ? await updateProjectInstance(payload)
        : await createProjectInstance(payload);
      const savedProject = unwrapResponseData(res);
      const nextProjectId = Number(savedProject?.ID || payload.ID || 0);
      toast.success(payload.ID ? '项目已保存' : '项目已新增');
      setProjectEditorOpen(false);
      await fetchProjectInstances(nextProjectId, true);
      await onProjectSaved?.();
    } catch (e) {
      toast.error('保存项目失败');
    } finally {
      setSavingProject(false);
    }
  };

  const removeProject = async (projectObj: any) => {
    if (!projectObj?.ID) return;
    if (!confirm(`确定删除项目「${projectObj.projectName || '未命名项目'}」及其所有相对路径吗？`)) return;

    setDeletingProjectId(Number(projectObj.ID));
    try {
      await deleteProjectInstance(projectObj);
      toast.success('项目已删除');
      setProjectEditorOpen(false);
      await fetchProjectInstances(undefined, false);
    } catch (e) {
      toast.error('删除项目失败');
    } finally {
      setDeletingProjectId(null);
    }
  };

  const savePath = async () => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }

    const nextFileUrl = normalizeRelativeDir(pathDraft.fileUrl || '');
    const nextFileName = normalizeFileName(pathDraft.fileName || '');

    if (!nextFileName) {
      toast.error('文件路径不能为空');
      return;
    }

    setSavingPath(true);
    try {
      const payload = {
        ...pathDraft,
        projectId,
        projectInstanceId: projectId,
        pathSet: Number(pathDraft.pathSet ?? 0),
        pathGroupId: Number(pathDraft.pathGroupId || activePathGroup?.ID || 0),
        fileUrl: nextFileUrl,
        fileName: nextFileName,
        enabled: Number(pathDraft.enabled ?? 1),
        incremented: Number(pathDraft.incremented ?? 0),
      };
      if (payload.ID) {
        await updatePath(payload);
        toast.success('相对路径已更新');
      } else {
        await createPath(payload);
        toast.success('相对路径已新增');
      }
      setPathDraft(emptyPathDraft(projectId));
      setPathEditorOpen(false);
      await fetchPathConfig();
    } catch (e) {
      toast.error('保存相对路径失败');
    } finally {
      setSavingPath(false);
    }
  };

  const openPathPlaceholderDialog = async (pathObj: any) => {
    setPlaceholderPath(pathObj);
    const savedPlaceholders = parseDbTemplatePlaceholders(pathObj?.dynamicPlaceholders);
    setPlaceholderRows([]);
    setLoadingPlaceholders(true);
    try {
      const inferredPlaceholders: DbTemplatePlaceholder[] = [
        ...extractPathPlaceholdersFromText(pathObj?.fileUrl),
        ...extractPathPlaceholdersFromText(pathObj?.fileName),
      ];
      const res: any = await getModelListByPathId(Number(pathObj.ID || 0));
      let models = unwrapResponseData(res);
      if (!Array.isArray(models)) models = [];
      models
        .filter((item: any) => Number(item.pathId || 0) === Number(pathObj.ID || 0))
        .forEach((model: any) => {
          inferredPlaceholders.push(...extractPathPlaceholdersFromText(model?.content));
          inferredPlaceholders.push(...extractPathPlaceholdersFromText(model?.prompt));
        });
      const savedMap = new Map(savedPlaceholders.map((item) => [String(item.key || '').trim(), item]));
      const nextRows = mergePathPlaceholders(inferredPlaceholders).map((item) => {
        const saved = savedMap.get(item.key);
        if (!saved) return item;
        return {
          ...item,
          description: saved.description || item.description,
          value: saved.value || item.value,
        };
      });
      setPlaceholderRows(nextRows);
    } catch (e) {
      toast.error('读取文件占位符失败');
    } finally {
      setLoadingPlaceholders(false);
    }
  };

  const addPathPlaceholderRow = () => {
    setPlaceholderRows((rows) => [...rows, { key: '', description: '', value: '' }]);
  };

  const updatePathPlaceholderRow = (index: number, patch: Partial<DbTemplatePlaceholder>) => {
    setPlaceholderRows((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, ...patch } : row));
  };

  const updatePathPlaceholderKey = (index: number, key: string) => {
    const option = pathPlaceholderOptions.find((item) => item.key === key);
    setPlaceholderRows((rows) => rows.map((row, rowIndex) => {
      if (rowIndex !== index) return row;
      return {
        ...row,
        key,
        description: row.description || option?.description || PATH_PLACEHOLDER_DESCRIPTIONS[key] || '',
        value: row.value || option?.value || PATH_PLACEHOLDER_DEFAULTS[key] || '',
      };
    }));
  };

  const removePathPlaceholderRow = (index: number) => {
    setPlaceholderRows((rows) => rows.filter((_, rowIndex) => rowIndex !== index));
  };

  const closePathPlaceholderDialog = () => {
    if (savingPlaceholders) return;
    setPlaceholderPath(null);
    setPlaceholderRows([]);
    setLoadingPlaceholders(false);
  };

  const savePathPlaceholders = async () => {
    if (!placeholderPath?.ID) return;
    setSavingPlaceholders(true);
    try {
      const dynamicPlaceholders = stringifyDbTemplatePlaceholders(
        placeholderRows.filter((row) => !isPathFieldSnippetPlaceholderKey(row.key)),
      );
      await updatePath({
        ...placeholderPath,
        dynamicPlaceholders,
      });
      setPaths((rows) => rows.map((pathObj) => (
        Number(pathObj.ID || 0) === Number(placeholderPath.ID || 0)
          ? { ...pathObj, dynamicPlaceholders }
          : pathObj
      )));
      setPlaceholderPath(null);
      setPlaceholderRows([]);
      toast.success('动态占位符已保存');
    } catch (e) {
      toast.error('保存动态占位符失败');
    } finally {
      setSavingPlaceholders(false);
    }
  };

  const savePathGroupDraft = async () => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    const nextBasePath = normalizeRelativeDir(pathGroupDraft.basePath || '');
    if (!nextBasePath) {
      toast.error('相对路径不能为空');
      return;
    }

    setSavingPathGroup(true);
    try {
      await createPathGroup({
        ...pathGroupDraft,
        projectId,
        projectInstanceId: projectId,
        pathSet: Number(pathGroupDraft.pathSet || 0),
        pathSetName: String(pathGroupDraft.pathSetName || '').trim(),
        basePath: nextBasePath,
      });
      toast.success('相对路径已新增');
      setPathGroupDraft(emptyPathGroupDraft(projectId));
      setPathGroupEditorOpen(false);
      await fetchPathConfig();
    } catch (e) {
      toast.error('新增相对路径失败');
    } finally {
      setSavingPathGroup(false);
    }
  };

  const savePathContent = async () => {
    if (!contentPath?.ID) return;
    setSavingContent(true);
    try {
      if (contentModel?.ID) {
        const payload = { ...contentModel, content: contentDraft };
        await updateModel(payload);
        setContentModel(payload);
        toast.success('文件内容已保存');
      } else {
        await createModel({ pathId: Number(contentPath.ID), content: contentDraft });
        const res: any = await getModelListByPathId(Number(contentPath.ID));
        let models = unwrapResponseData(res);
        if (!Array.isArray(models)) models = [];
        models = models.filter((item: any) => Number(item.pathId || 0) === Number(contentPath.ID));
        setContentModel(models[0] || null);
        toast.success('文件内容已初始化');
      }
    } catch (e) {
      toast.error('保存文件内容失败');
    } finally {
      setSavingContent(false);
    }
  };

  const savePathSetName = async (section: any, rawName: string) => {
    if (cancelPathSetNameSaveRef.current.has(section.key)) {
      cancelPathSetNameSaveRef.current.delete(section.key);
      setEditingPathSetNameKey(null);
      return;
    }
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }

    const nextName = String(rawName || '').trim();
    const currentStoredName = String(section?.pathSetName || '').trim();
    const currentDisplayName = getPathSetTitle(section);
    const edited = Object.prototype.hasOwnProperty.call(pathSetNameEdits, section.key);

    if (!edited && nextName === currentDisplayName) {
      setEditingPathSetNameKey(null);
      return;
    }
    if (nextName === currentStoredName) {
      setPathSetNameEdits((prev) => {
        const next = { ...prev };
        delete next[section.key];
        return next;
      });
      setEditingPathSetNameKey(null);
      return;
    }

    const pathIds = (section.paths || []).map((pathObj: any) => Number(pathObj.ID || 0)).filter(Boolean);
    const groupIds = (section.pathGroups || []).map((group: any) => Number(group.ID || 0)).filter(Boolean);
    if (pathIds.length === 0 && groupIds.length === 0) {
      toast.error('暂无路径，无法保存名称');
      return;
    }

    setSavingPathSetNameKey(section.key);
    const pathIdSet = new Set(pathIds);
    const groupIdSet = new Set(groupIds);
    try {
      const renameRes: any = await renamePathSetApi({
        projectId,
        projectInstanceId: projectId,
        pathSet: Number(section.pathSet || 0),
        pathIds,
        groupIds,
        pathSetName: nextName,
      });
      if (typeof renameRes?.code !== 'undefined' && Number(renameRes.code) !== 0) {
        throw new Error(renameRes.msg || 'rename failed');
      }

      setPaths((prev) => prev.map((pathObj) => (
        pathIdSet.has(Number(pathObj.ID || 0)) ? { ...pathObj, pathSetName: nextName } : pathObj
      )));
      setPathGroups((prev) => prev.map((group) => (
        groupIdSet.has(Number(group.ID || 0)) ? { ...group, pathSetName: nextName } : group
      )));
      setPathSetNameEdits((prev) => {
        const next = { ...prev };
        delete next[section.key];
        return next;
      });
      setEditingPathSetNameKey(null);
      toast.success('相对路径名称已更新');
      await fetchPathConfig();
    } catch (e) {
      toast.error('重命名相对路径失败');
      setPathSetNameEdits((prev) => ({ ...prev, [section.key]: currentDisplayName }));
    } finally {
      setSavingPathSetNameKey(null);
    }
  };

  const savePathGroup = async (section: any, group: any, rawNextBasePath: string) => {
    const editKey = makePathGroupEditKey(Number(section.pathSet || 0), group.key);
    const oldBasePath = normalizeRelativeDir(group?.basePath || '');
    const nextBasePath = normalizeRelativeDir(rawNextBasePath || '');
    if (!nextBasePath) {
      toast.error('相对路径不能为空');
      setPathGroupEdits((prev) => ({ ...prev, [editKey]: oldBasePath }));
      return;
    }
    if (oldBasePath === nextBasePath) {
      setPathGroupEdits((prev) => {
        const next = { ...prev };
        delete next[editKey];
        return next;
      });
      return;
    }

    setSavingPathGroupKey(editKey);
    try {
      if (Number(group.ID || 0) > 0) {
        await updatePathGroup({
          ...group,
          projectId,
          projectInstanceId: projectId,
          pathSet: Number(section.pathSet || 0),
          basePath: nextBasePath,
        });
      } else {
        const groupPaths = Array.isArray(group?.paths) ? group.paths : [];
        await Promise.all(groupPaths.map((pathObj: any) => {
          const nextFileUrl = replacePathPrefix(pathObj.fileUrl || '', oldBasePath, nextBasePath);
          const nextFileName = replacePathPrefix(pathObj.fileName || '', oldBasePath, nextBasePath);
          return updatePath({
            ...pathObj,
            projectId,
            projectInstanceId: projectId,
            pathSet: getPathSet(pathObj),
            fileUrl: nextFileUrl,
            fileName: nextFileName || normalizeFileName(pathObj.fileName || ''),
          });
        }));
      }
      toast.success('相对路径已更新');
      setPathGroupEdits((prev) => {
        const next = { ...prev };
        delete next[editKey];
        return next;
      });
      await fetchPathConfig();
    } catch (e) {
      toast.error('保存相对路径失败');
      setPathGroupEdits((prev) => ({ ...prev, [editKey]: oldBasePath }));
    } finally {
      setSavingPathGroupKey(null);
    }
  };

  const removePathGroup = async (section: any, group: any) => {
    const editKey = makePathGroupEditKey(Number(section.pathSet || 0), group.key);
    const deleteState = getPathGroupDeleteState(group);
    if (!deleteState.canDelete) {
      toast.error(deleteState.reason);
      return;
    }

    setSavingPathGroupKey(editKey);
    try {
      if (Number(group.ID || 0) > 0) {
        await deletePathGroup(group);
      }
      setPathGroupEdits((prev) => {
        const next = { ...prev };
        delete next[editKey];
        return next;
      });
      if (activePathSetKey === section.key && activePathGroupKey === group.key) {
        setActivePathSetKey(null);
        setActivePathGroupKey(null);
      }
      toast.success('空子目录已删除');
      await fetchPathConfig();
    } catch (e) {
      toast.error('删除子目录失败');
    } finally {
      setSavingPathGroupKey(null);
    }
  };

  const openCopyPathSetConfirm = (section: any) => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    const sourcePaths = Array.isArray(section?.paths) ? section.paths : [];
    const sourceGroups = Array.isArray(section?.pathGroups) ? section.pathGroups : [];
    if (sourcePaths.length === 0 && sourceGroups.length === 0) {
      toast.error('请先在当前相对路径中新增路径');
      return;
    }
    selectPathSetSection(section);
    setPathSetActionConfirm({ action: 'copy', section });
  };

  const copyPathSet = async (section: any) => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    const sourcePaths = Array.isArray(section?.paths) ? section.paths : [];
    const sourceGroups = Array.isArray(section?.pathGroups) ? section.pathGroups : [];
    if (sourcePaths.length === 0 && sourceGroups.length === 0) {
      toast.error('请先在当前相对路径中新增路径');
      return;
    }

    const nextPathSet = Math.max(
      paths.reduce((max, item) => Math.max(max, getPathSet(item)), 0),
      pathGroups.reduce((max, group) => Math.max(max, Number(group.pathSet || 0)), 0),
    ) + 1;
    setCopyingPathSet(Number(section.pathSet || 0));
    try {
      let copiedPathSet = nextPathSet;
      const copyRes: any = await copyPathSetApi({
        projectId,
        projectInstanceId: projectId,
        pathSet: Number(section.pathSet || 0),
        pathIds: sourcePaths.map((pathObj: any) => Number(pathObj.ID || 0)).filter(Boolean),
        groupIds: sourceGroups.map((group: any) => Number(group.ID || 0)).filter(Boolean),
      });
      if (typeof copyRes?.code !== 'undefined' && Number(copyRes.code) !== 0) {
        throw new Error(copyRes.msg || 'copy failed');
      }
      const copyData = unwrapResponseData(copyRes);
      copiedPathSet = Number(copyData?.pathSet || copiedPathSet);
      toast.success('相对路径配置已复制');
      await fetchPathConfig();
      setSelectedPathSetKey(makePathSetSectionKey(copiedPathSet));
      persistSelectedPathSetIdentity(`path-set-${copiedPathSet}`);
    } catch (e) {
      toast.error('复制相对路径配置失败');
    } finally {
      setCopyingPathSet(null);
    }
  };

  const openRemovePathSetConfirm = (section: any) => {
    if (section?.isPrimary) return;
    setPathSetActionConfirm({ action: 'delete', section });
  };

  const confirmPathSetAction = () => {
    if (!pathSetActionConfirm) return;
    const { action, section } = pathSetActionConfirm;
    setPathSetActionConfirm(null);
    if (action === 'copy') {
      selectPathSetSection(section);
      copyPathSet(section);
      return;
    }
    removePathSet(section);
  };

  const removePathSet = async (section: any) => {
    if (section?.isPrimary) return;

    setDeletingPathSet(Number(section.pathSet || 0));
    try {
      await deletePathSetApi({
        projectId,
        projectInstanceId: projectId,
        pathSet: Number(section.pathSet || 0),
        groupIds: (section.pathGroups || []).map((group: any) => Number(group.ID || 0)).filter(Boolean),
      });
      if (activePathSetKey === section.key) {
        setActivePathSetKey(null);
        setActivePathGroupKey(null);
      }
      if (selectedPathSetKey === section.key) {
        setSelectedPathSetKey(makePathSetSectionKey(0));
        persistSelectedPathSetIdentity('path-set-primary');
      }
      toast.success('相对路径配置已删除');
      await fetchPathConfig();
    } catch (e) {
      toast.error('删除相对路径配置失败');
    } finally {
      setDeletingPathSet(null);
    }
  };

  const removePath = async (pathObj: any) => {
    if (!confirm(`确定删除路径「${formatPathLabel(pathObj)}」吗？`)) return;

    setDeletingPathId(Number(pathObj.ID));
    try {
      await deletePath(pathObj);
      toast.success('相对路径已删除');
      if (Number(pathDraft.ID || 0) === Number(pathObj.ID || 0)) {
        setPathDraft(emptyPathDraft(projectId));
        setPathEditorOpen(false);
      }
      await fetchPaths();
    } catch (e) {
      toast.error('删除相对路径失败');
    } finally {
      setDeletingPathId(null);
    }
  };

  const togglePathEnabled = async (pathObj: any) => {
    const nextEnabled = Number(pathObj.enabled || 0) === 1 ? 0 : 1;
    try {
      await updatePathEnabled({ ...pathObj, enabled: nextEnabled });
      setPaths((prev) => prev.map((item) => (
        Number(item.ID) === Number(pathObj.ID) ? { ...item, enabled: nextEnabled } : item
      )));
      setPathDraft((prev: any) => (
        Number(prev.ID || 0) === Number(pathObj.ID || 0) ? { ...prev, enabled: nextEnabled } : prev
      ));
    } catch (e) {
      toast.error('更新启用状态失败');
    }
  };

  const openEditor = (section = selectedPathSetSection) => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    const params = new URLSearchParams();
    params.set('templateId', String(templateProjectId || ''));
    params.set('returnTemplateId', String(templateProjectId || ''));
    params.set('returnProjectInstanceId', String(projectId || ''));
    if (pathDetailOpen && activePathGroupKey) {
      const returnSection = activePathSetSection || section;
      params.set('returnView', 'pathDetail');
      params.set('returnPathSetKey', String(returnSection?.key || ''));
      params.set('returnPathSet', String(Number(returnSection?.pathSet || section?.pathSet || 0)));
      params.set('returnPathGroupKey', String(activePathGroupKey || ''));
    }
    if (section) {
      params.set('pathSet', String(Number(section.pathSet || 0)));
      params.set('pathSetName', getPathSetTitle(section));
      const pathIds = (section.paths || []).map((pathObj: any) => Number(pathObj.ID || 0)).filter(Boolean);
      if (pathIds.length > 0) params.set('pathIds', pathIds.join(','));
    }
    navigate(`/code-generate/${projectId}/templates?${params.toString()}`);
  };

  return (
    <div data-testid="project-config-dialog" className="fixed inset-0 z-50 flex overflow-hidden bg-white text-gray-900">
      <aside className="flex w-[380px] shrink-0 flex-col border-r border-gray-200 bg-gray-50">
        <div className="flex h-16 items-center justify-between gap-3 border-b border-gray-200 bg-white px-5">
          <div className="min-w-0">
            <div className="truncate text-xs font-bold uppercase text-gray-400" title={templateProject?.projectName || ''}>
              模板配置 · {templateProject?.projectName || '未命名模板'}
            </div>
            <h2 className="truncate text-lg font-extrabold text-gray-950">项目列表</h2>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <button
              type="button"
              onClick={openCreateProject}
              className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-900"
              title="新增项目"
            >
              <Plus size={18} />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-3">
          {loadingProjects ? (
            <div className="mt-10 text-center text-sm text-gray-400">加载项目中...</div>
          ) : projectList.length === 0 ? (
            <div className="flex min-h-[300px] flex-col items-center justify-center text-center">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-white text-gray-300 shadow-sm ring-1 ring-gray-200">
                <FileCode size={24} />
              </div>
              <div className="text-sm font-bold text-gray-600">暂无项目</div>
              <button
                type="button"
                onClick={openCreateProject}
                className="mt-4 inline-flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-bold text-white transition-colors hover:bg-gray-800"
              >
                <Plus size={15} />
                新增项目
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-1">
              {projectList.map((item) => {
                const active = Number(item.ID || 0) === projectId;
                return (
                  <div
                    key={item.ID}
                    data-testid="project-list-row"
                    onClick={() => selectProject(item)}
                    className={`group flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2.5 transition-colors ${active ? 'border-gray-900 bg-gray-900 text-white' : 'border-transparent bg-white text-gray-700 hover:border-gray-200 hover:bg-gray-100'}`}
                  >
                    <FileCode size={16} className={active ? 'shrink-0 text-white' : 'shrink-0 text-gray-400'} />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-extrabold" title={item.projectName || '未命名项目'}>
                        {item.projectName || '未命名项目'}
                      </div>
                      <div className={`mt-0.5 truncate text-xs ${active ? 'text-white/55' : 'text-gray-400'}`} title={item.diskPath || ''}>
                        {item.diskPath || `ID: ${item.ID}`}
                      </div>
                    </div>
                    <button
                      type="button"
                      data-testid="project-edit-button"
                      onClick={(event) => {
                        event.stopPropagation();
                        openProjectEditor(item);
                      }}
                      className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 opacity-0 hover:bg-white hover:text-gray-900 group-hover:opacity-100'}`}
                      title="编辑项目"
                    >
                      <Edit2 size={14} />
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </aside>

      <main className="flex min-h-0 min-w-0 flex-1 flex-col bg-white">
        <div className="flex h-16 items-center justify-between gap-4 border-b border-gray-200 px-6">
          <div className="flex min-w-0 items-center gap-3">
            <div className="min-w-0">
              <h3 className="truncate text-lg font-extrabold text-gray-950">
                相对路径配置
              </h3>
              <p className="mt-0.5 truncate text-xs text-gray-400" title={selectedProject?.projectName || ''}>
                {`${selectedProject?.projectName || '请先选择项目'} · 共 ${projectId ? pathSetSections.length : 0} 个配置 / ${totalPathGroupCount} 个子服务 / ${paths.length} 条路径`}
              </p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <div className="relative w-72">
              <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                value={pathSearch}
                onChange={(event) => setPathSearch(event.target.value)}
                placeholder="搜索相对路径..."
                className="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-3 text-sm outline-none transition focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5"
              />
            </div>
            <button
              type="button"
              onClick={() => openEditor(selectedPathSetSection)}
              disabled={!projectId}
              className="inline-flex h-9 items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 text-sm font-bold text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <FileCode size={15} />
              编辑代码模版
            </button>
            <button
              type="button"
              onClick={onClose}
              className="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-400 transition-colors hover:bg-gray-50 hover:text-gray-900"
              title="关闭"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          <section className="h-full min-w-0 overflow-y-auto p-6">
            {loadingPaths ? (
              <div className="mt-20 text-center text-sm text-gray-400">加载中...</div>
            ) : !projectId ? (
              <div className="flex h-full min-h-[360px] flex-col items-center justify-center text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-lg bg-gray-100 text-gray-300">
                  <FileCode size={28} />
                </div>
                <div className="text-base font-bold text-gray-700">请选择或新增项目</div>
              </div>
            ) : filteredPathSetSections.length === 0 ? (
              <div className="flex h-full min-h-[360px] flex-col items-center justify-center text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-lg bg-gray-100 text-gray-300">
                  <Folder size={28} />
                </div>
                <div className="text-base font-bold text-gray-700">没有匹配相对路径</div>
              </div>
            ) : (
              <div className="space-y-2">
                {filteredPathSetSections.map((section: any) => {
                  const sectionTitle = getPathSetTitle(section);
                  const sectionPathSet = Number(section.pathSet || 0);
                  const sectionSelected = selectedPathSetKey === section.key;
                  const copyingThisSet = copyingPathSet === sectionPathSet;
                  const deletingThisSet = deletingPathSet === Number(section.pathSet || 0);
                  const savingThisSetName = savingPathSetNameKey === section.key;
                  const sectionNameEditValue = pathSetNameEdits[section.key] ?? sectionTitle;
                  const editingThisSetName = editingPathSetNameKey === section.key;

                  return (
                    <div
                      key={section.key}
                      data-testid="path-set-section"
                      onClick={() => selectPathSetSection(section)}
                      className={`cursor-pointer overflow-hidden rounded-lg border bg-white transition-all ${sectionSelected ? 'border-gray-900 shadow-sm ring-2 ring-gray-900/10' : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'}`}
                    >
                      <div className={`flex flex-wrap items-center justify-between gap-4 border-b px-4 py-2 transition-colors ${sectionSelected ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-200 bg-gray-50 text-gray-900'}`}>
                        <div className="min-w-0 flex-1 basis-56">
                          <div className="flex min-w-0 items-center gap-1.5 text-sm font-extrabold">
                            <Folder size={15} className={`shrink-0 ${sectionSelected ? 'text-white/70' : 'text-gray-400'}`} />
                            {editingThisSetName ? (
                              <input
                                type="text"
                                value={sectionNameEditValue}
                                disabled={savingThisSetName}
                                autoFocus
                                onClick={(event) => event.stopPropagation()}
                                onFocus={() => selectPathSetSection(section)}
                                onChange={(event) => setPathSetNameEdits((prev) => ({ ...prev, [section.key]: event.target.value }))}
                                onBlur={() => {
                                  setEditingPathSetNameKey(null);
                                  setPathSetNameEdits((prev) => {
                                    const next = { ...prev };
                                    delete next[section.key];
                                    return next;
                                  });
                                }}
                                onKeyDown={(event) => {
                                  if (event.key === 'Enter') {
                                    event.preventDefault();
                                    void savePathSetName(section, event.currentTarget.value);
                                  }
                                  if (event.key === 'Escape') {
                                    event.preventDefault();
                                    setEditingPathSetNameKey(null);
                                    setPathSetNameEdits((prev) => {
                                      const next = { ...prev };
                                      delete next[section.key];
                                      return next;
                                    });
                                  }
                                }}
                                className={`min-w-0 flex-1 truncate rounded-md border border-gray-300 bg-white px-2 py-0.5 text-sm font-extrabold text-gray-900 outline-none transition focus:ring-2 focus:ring-black/5 disabled:opacity-60`}
                                title="按 Enter 保存"
                              />
                            ) : (
                              <span
                                className="min-w-0 flex-1 truncate px-1 py-0.5"
                                title="双击重命名"
                                onDoubleClick={(event) => {
                                  event.stopPropagation();
                                  selectPathSetSection(section);
                                  setPathSetNameEdits((prev) => ({ ...prev, [section.key]: sectionTitle }));
                                  setEditingPathSetNameKey(section.key);
                                }}
                              >
                                {sectionTitle}
                              </span>
                            )}
                            {savingThisSetName ? (
                              <RefreshCw size={13} className="shrink-0 animate-spin opacity-70" />
                            ) : null}
                          </div>
                          <div className={`mt-0.5 truncate text-xs font-bold ${sectionSelected ? 'text-white/60' : 'text-gray-400'}`}>
                            {section.paths.length} 条路径 / {section.pathGroups.length} 个子服务
                          </div>
                        </div>

                        <div className="flex shrink-0 items-center gap-2">
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              selectPathSetSection(section);
                              openCreatePathGroup(section);
                            }}
                            disabled={!projectId}
                            className={`inline-flex h-8 items-center gap-1.5 rounded-lg px-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${sectionSelected ? 'bg-white text-gray-900 hover:bg-white/90' : 'bg-gray-900 text-white hover:bg-gray-800'}`}
                            title={`在 ${sectionTitle} 中新增公共相对路径`}
                          >
                            <Plus size={15} />
                            新增路径
                          </button>
                          {!section.isPrimary && (
                            <button
                              type="button"
                              onClick={(event) => {
                                event.stopPropagation();
                                openRemovePathSetConfirm(section);
                              }}
                              disabled={deletingThisSet}
                              className={`inline-flex h-8 items-center gap-1.5 rounded-lg border px-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${sectionSelected ? 'border-white/20 bg-white/10 text-white hover:bg-red-500/20 hover:text-white' : 'border-gray-200 bg-white text-gray-700 hover:bg-red-50 hover:text-red-600'}`}
                              title={`删除 ${sectionTitle}`}
                            >
                              {deletingThisSet ? <RefreshCw size={15} className="animate-spin" /> : <Trash2 size={15} />}
                              删除
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              openCopyPathSetConfirm(section);
                            }}
                            disabled={!projectId || copyingThisSet || section.paths.length === 0}
                            className={`inline-flex h-8 items-center gap-1.5 rounded-lg border px-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${sectionSelected ? 'border-white/20 bg-white/10 text-white hover:bg-white/15' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50'}`}
                            title={`复制 ${sectionTitle} 的全部内容`}
                          >
                            {copyingThisSet ? <RefreshCw size={15} className="animate-spin" /> : <Copy size={15} />}
                            复制
                          </button>
                        </div>
                      </div>

                      {section.pathGroups.length === 0 ? (
                        <div className="flex min-h-[96px] flex-col items-center justify-center px-4 py-6 text-center">
                          <Folder size={26} className="mb-3 text-gray-300" />
                          <div className="text-sm font-bold text-gray-500">{pathSearch ? '没有匹配相对路径' : '暂无相对路径配置'}</div>
                        </div>
                      ) : (
                        <>
                          <div className="grid grid-cols-[minmax(360px,1fr)_minmax(280px,360px)] border-b border-gray-200 bg-white px-4 py-1.5 text-xs font-bold uppercase text-gray-400">
                            <div>相对路径</div>
                            <div className="text-right">操作</div>
                          </div>

                          <div className="divide-y divide-gray-100">
                            {section.pathGroups.map((group: any) => {
                              const editKey = makePathGroupEditKey(Number(section.pathSet || 0), group.key);
                              const editingValue = pathGroupEdits[editKey] ?? group.basePath;
                              const savingThisGroup = savingPathGroupKey === editKey;

                              return (
                                <div
                                  key={group.key}
                                  data-testid="path-group-row"
                                  onClick={() => {
                                    setActivePathSetKey(section.key);
                                    setActivePathGroupKey(group.key);
                                  }}
                                  className="grid cursor-pointer grid-cols-[minmax(360px,1fr)_minmax(280px,360px)] items-center gap-3 bg-white px-4 py-2 text-gray-700 transition-colors hover:bg-gray-50"
                                  title={`进入 ${group.basePath} 子目录`}
                                >
                                  <div className="min-w-0">
                                    <div className="mb-0.5 flex items-center gap-1.5 text-xs font-bold text-gray-400">
                                      <Folder size={13} />
                                      相对路径
                                    </div>
                                    <div className="flex items-center gap-2">
                                      <input
                                        type="text"
                                        value={editingValue}
                                        disabled={savingThisGroup}
                                        onClick={(event) => event.stopPropagation()}
                                        onChange={(event) => setPathGroupEdits((prev) => ({ ...prev, [editKey]: event.target.value }))}
                                        onBlur={(event) => savePathGroup(section, group, event.target.value)}
                                        onKeyDown={(event) => {
                                          if (event.key === 'Enter') {
                                            event.currentTarget.blur();
                                          }
                                          if (event.key === 'Escape') {
                                            event.preventDefault();
                                            setPathGroupEdits((prev) => {
                                              const next = { ...prev };
                                              delete next[editKey];
                                              return next;
                                            });
                                          }
                                        }}
                                        className="min-w-0 flex-1 rounded-md border border-transparent bg-transparent px-0 py-0.5 font-mono text-sm font-extrabold text-gray-900 outline-none transition focus:border-gray-200 focus:bg-white focus:px-2 focus:ring-2 focus:ring-black/5 disabled:opacity-60"
                                        title={editingValue}
                                      />
                                      {savingThisGroup && <RefreshCw size={14} className="shrink-0 animate-spin text-gray-400" />}
                                    </div>
                                  </div>

                                  <div className="flex min-w-0 items-center justify-end gap-2">
                                    <button
                                      type="button"
                                      onClick={(event) => {
                                        event.stopPropagation();
                                        removePathGroup(section, group);
                                      }}
                                      className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 text-sm font-bold text-gray-700 transition-colors hover:bg-red-50 hover:text-red-600"
                                      title="删除空子目录"
                                    >
                                      <Trash2 size={14} />
                                      删除
                                    </button>
                                    <button
                                      type="button"
                                      onClick={(event) => {
                                        event.stopPropagation();
                                        setActivePathSetKey(section.key);
                                        setActivePathGroupKey(group.key);
                                      }}
                                      className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 text-sm font-bold text-gray-700 transition-colors hover:bg-gray-50 hover:text-gray-950"
                                      title={`进入 ${group.basePath} 子目录`}
                                    >
                                      子目录
                                      <ArrowRight size={14} />
                                    </button>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        </>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </section>
        </div>
      </main>

      {pathDetailOpen && (
        <div data-testid="path-detail-dialog" className="fixed inset-0 z-[60] flex h-screen w-screen flex-col overflow-hidden bg-white text-gray-900">
          <div className="flex h-16 items-center justify-between gap-4 border-b border-gray-200 px-6">
            <div className="flex min-w-0 items-center gap-3">
              <div className="min-w-0">
                <h3 className="truncate text-lg font-extrabold text-gray-950">相对路径明细</h3>
                <p className="mt-0.5 truncate text-xs text-gray-400" title={activePathGroup?.basePath || selectedProject?.projectName || ''}>
                  {activePathSetSection ? `${getPathSetTitle(activePathSetSection)} · ` : ''}{activePathGroup?.basePath || selectedProject?.projectName || '请先选择项目'} · 共 {filteredPaths.length} 条
                </p>
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-2">
              <div className="relative w-72">
                <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  value={pathSearch}
                  onChange={(event) => setPathSearch(event.target.value)}
                  placeholder="搜索文件路径..."
                  className="w-full rounded-lg border border-gray-200 bg-gray-50 py-2 pl-9 pr-3 text-sm outline-none transition focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5"
                />
              </div>
              <button
                type="button"
                onClick={() => openCreatePath(Number(activePathSetSection?.pathSet || 0), activePathGroup?.basePath || '')}
                disabled={!projectId || !activePathGroup}
                className="inline-flex h-9 items-center gap-2 rounded-lg bg-gray-900 px-3 text-sm font-bold text-white transition-colors hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Plus size={15} />
                新增文件
              </button>
              <button
                type="button"
                onClick={() => openEditor(activePathSetSection || selectedPathSetSection)}
                disabled={!projectId}
                className="inline-flex h-9 items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 text-sm font-bold text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <FileCode size={15} />
                编辑代码模版
              </button>
              <button
                type="button"
                onClick={() => {
                  setActivePathSetKey(null);
                  setActivePathGroupKey(null);
                }}
                className="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-400 transition-colors hover:bg-gray-50 hover:text-gray-900"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>
          </div>

          <section className="min-h-0 flex-1 overflow-y-auto p-6">
            {filteredPaths.length === 0 ? (
              <div className="flex h-full min-h-[360px] flex-col items-center justify-center text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-lg bg-gray-100 text-gray-300">
                  <FileCode size={28} />
                </div>
                <div className="text-base font-bold text-gray-700">{pathSearch ? '没有匹配路径' : '该相对路径暂无文件'}</div>
              </div>
            ) : (
              <div className="overflow-hidden rounded-lg border border-gray-200">
                <div className="grid grid-cols-[minmax(520px,3fr)_minmax(220px,1fr)_120px_212px] border-b border-gray-200 bg-gray-50 px-4 py-2 text-xs font-bold uppercase text-gray-400">
                  <div>剩余父级</div>
                  <div>文件路径</div>
                  <div>状态</div>
                  <div className="text-right">操作</div>
                </div>

                <div className="divide-y divide-gray-100">
                  {filteredPaths.map((pathObj) => {
                    const active = Number(pathDraft.ID || 0) === Number(pathObj.ID || 0);
                    const enabled = Number(pathObj.enabled || 0) === 1;
                    return (
                      <div
                        key={pathObj.ID}
                        data-testid="path-row"
                        onClick={() => openPathContentEditor(pathObj)}
                        className={`grid w-full cursor-pointer grid-cols-[minmax(520px,3fr)_minmax(220px,1fr)_120px_212px] items-center gap-3 px-4 py-3 text-left transition-colors ${active ? 'bg-gray-900 text-white' : 'bg-white text-gray-700 hover:bg-gray-50'}`}
                      >
                        <div className="min-w-0">
                          <div className={`mb-1 flex items-center gap-1.5 text-xs font-bold ${active ? 'text-white/60' : 'text-gray-400'}`}>
                            <Folder size={13} />
                            剩余父级
                          </div>
                          <div className="truncate font-mono text-sm" title={getPathParentSuffix(pathObj, activePathGroup?.basePath || '')}>
                            {getPathParentSuffix(pathObj, activePathGroup?.basePath || '')}
                          </div>
                        </div>

                        <div className="min-w-0">
                          <div className={`mb-1 flex items-center gap-1.5 text-xs font-bold ${active ? 'text-white/60' : 'text-gray-400'}`}>
                            <FileCode size={13} />
                            编辑代码模版文件
                          </div>
                          <div className="truncate font-mono text-sm font-semibold" title={formatPathFileLabel(pathObj)}>
                            {formatPathFileLabel(pathObj)}
                          </div>
                        </div>

                        <div className="flex items-center gap-2">
                          <span className={`rounded-full px-2 py-1 text-xs font-bold ${enabled ? (active ? 'bg-emerald-400/20 text-emerald-100' : 'bg-emerald-50 text-emerald-700') : (active ? 'bg-white/10 text-white/50' : 'bg-gray-100 text-gray-400')}`}>
                            {enabled ? '启用' : '停用'}
                          </span>
                          {Number(pathObj.incremented || 0) === 1 && (
                            <span className={`rounded-full px-2 py-1 text-xs font-bold ${active ? 'bg-cyan-400/20 text-cyan-100' : 'bg-cyan-50 text-cyan-700'}`}>
                              增量
                            </span>
                          )}
                        </div>

                        <div className="flex justify-end gap-1">
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              openPathPromptDialog(pathObj);
                            }}
                            disabled={!enabled}
                            className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 hover:bg-teal-50 hover:text-teal-700'}`}
                            title="编辑提示词"
                          >
                            <Wand2 size={14} />
                          </button>
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              openPathContentEditor(pathObj);
                            }}
                            className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 hover:bg-cyan-50 hover:text-cyan-700'}`}
                            title="编辑文件内容"
                          >
                            <FileCode size={14} />
                          </button>
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              openEditPath(pathObj);
                            }}
                            className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 hover:bg-gray-100 hover:text-gray-700'}`}
                            title="编辑路径"
                          >
                            <Edit2 size={14} />
                          </button>
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              openPathPlaceholderDialog(pathObj);
                            }}
                            className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 hover:bg-violet-50 hover:text-violet-700'}`}
                            title="动态占位符"
                          >
                            <Braces size={14} />
                          </button>
                          <button
                            type="button"
                            onClick={(event) => {
                              event.stopPropagation();
                              removePath(pathObj);
                            }}
                            className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${active ? 'text-white/70 hover:bg-white/10 hover:text-white' : 'text-gray-300 hover:bg-red-50 hover:text-red-600'}`}
                            title="删除"
                          >
                            {deletingPathId === Number(pathObj.ID) ? <RefreshCw size={14} className="animate-spin" /> : <Trash2 size={14} />}
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </section>
        </div>
      )}

      {pathSetActionConfirm && (
        <div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div className="w-full max-w-[460px] overflow-hidden rounded-xl border border-gray-200 bg-white shadow-2xl">
            <div className="flex items-start justify-between gap-4 border-b border-gray-100 px-6 py-5">
              <div className="min-w-0">
                <h4 className="text-lg font-extrabold text-gray-950">
                  {pathSetActionConfirm.action === 'copy' ? '确认复制配置' : '确认删除配置'}
                </h4>
                <p className="mt-1 truncate text-sm font-bold text-gray-400" title={getPathSetTitle(pathSetActionConfirm.section)}>
                  {getPathSetTitle(pathSetActionConfirm.section)}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setPathSetActionConfirm(null)}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-900"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="px-6 py-5 text-sm font-medium leading-6 text-gray-600">
              {pathSetActionConfirm.action === 'copy' ? (
                <p>
                  确认复制「{getPathSetTitle(pathSetActionConfirm.section)}」吗？系统会复制该配置下的子目录、文件路径、文件内容和提示词。
                </p>
              ) : (
                <p>
                  确认删除「{getPathSetTitle(pathSetActionConfirm.section)}」吗？系统会删除该配置下的子目录、文件路径、文件内容和提示词，删除后不可恢复。
                </p>
              )}
            </div>

            <div className="flex justify-end gap-3 border-t border-gray-100 bg-gray-50 px-6 py-4">
              <button
                type="button"
                onClick={() => setPathSetActionConfirm(null)}
                className="rounded-lg px-4 py-2 text-sm font-bold text-gray-600 transition-colors hover:bg-white hover:text-gray-900"
              >
                取消
              </button>
              <button
                type="button"
                onClick={confirmPathSetAction}
                className={`rounded-lg px-4 py-2 text-sm font-extrabold text-white transition-colors ${
                  pathSetActionConfirm.action === 'delete'
                    ? 'bg-red-600 hover:bg-red-700'
                    : 'bg-gray-900 hover:bg-gray-800'
                }`}
              >
                {pathSetActionConfirm.action === 'copy' ? '确认复制' : '确认删除'}
              </button>
            </div>
          </div>
        </div>
      )}

      {promptSummaryOpen && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div
            data-testid="prompt-summary-dialog"
            className="flex max-h-[92vh] w-full max-w-[980px] flex-col overflow-hidden rounded-xl border border-gray-200 bg-white text-gray-900 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4 border-b border-gray-200 bg-gray-50 px-6 py-4">
              <div className="min-w-0">
                <div className="text-xs font-bold uppercase tracking-wider text-teal-600">文件提示词</div>
                <h4 className="mt-1 truncate text-lg font-extrabold text-gray-950" title={formatPathLabel(promptSummaryPath || {})}>
                  {formatPathFileLabel(promptSummaryPath || {})}
                </h4>
                <p className="mt-1 truncate font-mono text-xs font-bold text-gray-400" title={formatPathLabel(promptSummaryPath || {})}>
                  {formatPathLabel(promptSummaryPath || {})}
                </p>
              </div>
              <button
                type="button"
                onClick={() => {
                  if (!savingPrompt) {
                    setPromptSummaryOpen(false);
                    setPromptSummaryPath(null);
                    setPromptModel(null);
                    setPromptDraft('');
                  }
                }}
                disabled={savingPrompt}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="min-h-0 flex-1 px-6 py-5">
              <textarea
                value={promptDraft}
                onChange={(event) => setPromptDraft(event.target.value)}
                disabled={loadingPrompt || savingPrompt}
                placeholder="输入当前文件的提示词..."
                autoFocus
                className="h-[54vh] min-h-[360px] w-full resize-none rounded-lg border border-gray-200 bg-gray-50 p-4 font-mono text-sm font-semibold leading-6 text-gray-800 outline-none transition focus:border-teal-300 focus:bg-white focus:ring-2 focus:ring-teal-500/15 disabled:cursor-not-allowed disabled:opacity-60"
              />
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-gray-200 bg-gray-50 px-6 py-4">
              {loadingPrompt && (
                <div className="mr-auto inline-flex items-center gap-2 text-sm font-bold text-gray-400">
                  <RefreshCw size={15} className="animate-spin" />
                  正在加载提示词
                </div>
              )}
              <button
                type="button"
                onClick={() => {
                  if (!savingPrompt) {
                    setPromptSummaryOpen(false);
                    setPromptSummaryPath(null);
                    setPromptModel(null);
                    setPromptDraft('');
                  }
                }}
                disabled={savingPrompt}
                className="inline-flex h-10 items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 text-sm font-bold text-gray-700 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-60"
              >
                取消
              </button>
              <button
                type="button"
                onClick={savePathPrompt}
                disabled={loadingPrompt || savingPrompt}
                className="inline-flex h-10 items-center gap-2 rounded-lg bg-gray-950 px-4 text-sm font-extrabold text-white transition-colors hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {savingPrompt ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
                保存提示词
              </button>
            </div>
          </div>
        </div>
      )}

      {projectEditorOpen && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div
            data-testid="project-editor-dialog"
            className="flex max-h-[92vh] w-full max-w-[560px] flex-col overflow-hidden rounded-xl border border-white/10 bg-[#1f1f1f] text-gray-100 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4 border-b border-white/10 bg-[#111111] px-7 py-5">
              <div className="min-w-0">
                <div className="text-xs font-bold uppercase text-gray-400">{projectDraft.ID ? '编辑项目' : '新增项目'}</div>
                <h4 className="mt-1 truncate text-xl font-extrabold text-white" title={projectDraft.projectName || '未命名项目'}>
                  {projectDraft.projectName || '未命名项目'}
                </h4>
              </div>
              <button
                type="button"
                onClick={() => setProjectEditorOpen(false)}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
                title="关闭"
              >
                <X size={20} />
              </button>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto bg-[#242424] px-7 py-6">
              <div className="space-y-6">
                <div>
                  <label className="mb-2.5 block text-sm font-extrabold text-gray-400">项目名称</label>
                  <input
                    type="text"
                    value={projectDraft.projectName || ''}
                    onChange={(event) => setProjectDraft({ ...projectDraft, projectName: event.target.value })}
                    className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 text-lg font-extrabold text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                    placeholder="输入项目名称"
                    autoFocus
                  />
                </div>

                <div>
                  <label className="mb-2.5 block text-sm font-extrabold text-gray-400">磁盘输出路径</label>
                  <textarea
                    value={projectDraft.diskPath || ''}
                    onChange={(event) => setProjectDraft({ ...projectDraft, diskPath: event.target.value })}
                    className="min-h-[132px] w-full resize-y rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold leading-relaxed text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                    placeholder="/Users/workspace/project"
                  />
                </div>

                <div>
                  <label className="mb-2.5 block text-sm font-extrabold text-gray-400">备注</label>
                  <textarea
                    value={projectDraft.remark || ''}
                    onChange={(event) => setProjectDraft({ ...projectDraft, remark: event.target.value })}
                    className="min-h-[152px] w-full resize-y rounded-lg border border-white/15 bg-[#111111] px-4 py-3 text-lg font-bold leading-relaxed text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                    placeholder="记录项目说明"
                  />
                </div>
              </div>
            </div>

            <div className="space-y-3 border-t border-white/10 bg-[#171717] p-5">
              {projectDraft.ID && (
                <button
                  type="button"
                  onClick={() => removeProject(projectDraft)}
                  disabled={deletingProjectId === Number(projectDraft.ID)}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-white/15 px-4 py-3 text-base font-extrabold text-gray-100 transition-colors hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {deletingProjectId === Number(projectDraft.ID) ? <RefreshCw size={18} className="animate-spin" /> : <Trash2 size={18} />}
                  删除项目
                </button>
              )}
              <button
                type="button"
                onClick={saveProject}
                disabled={savingProject}
                className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#0f172a] px-4 py-3 text-base font-extrabold text-white transition-colors hover:bg-[#111c34] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {savingProject ? <RefreshCw size={18} className="animate-spin" /> : <Save size={18} />}
                保存项目
              </button>
            </div>
          </div>
        </div>
      )}

      {contentEditorOpen && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div
            data-testid="path-content-editor-dialog"
            className="flex h-[88vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-xl border border-white/10 bg-[#111827] text-gray-100 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4 border-b border-white/10 bg-[#0f172a] px-6 py-4">
              <div className="min-w-0">
                <div className="text-xs font-bold uppercase text-gray-400">编辑代码模版文件内容</div>
                <h4 className="mt-1 truncate text-lg font-extrabold text-white" title={formatPathLabel(contentPath || {})}>
                  {formatPathFileLabel(contentPath || {})}
                </h4>
                <p className="mt-1 truncate font-mono text-xs font-bold text-gray-400" title={formatPathLabel(contentPath || {})}>
                  {formatPathLabel(contentPath || {})}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  onClick={savePathContent}
                  disabled={savingContent || loadingContent}
                  className="inline-flex h-9 items-center gap-2 rounded-lg bg-teal-500 px-4 text-sm font-extrabold text-slate-950 transition-colors hover:bg-teal-400 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {savingContent ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
                  保存内容
                </button>
                <button
                  type="button"
                  onClick={() => setContentEditorOpen(false)}
                  className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
                  title="关闭"
                >
                  <X size={18} />
                </button>
              </div>
            </div>

            <div className="relative min-h-0 flex-1">
              {loadingContent && (
                <div className="absolute inset-0 z-10 flex items-center justify-center bg-slate-950/75">
                  <span className="rounded-lg border border-teal-500/30 bg-teal-500/10 px-4 py-2 text-sm font-bold text-teal-300">
                    加载文件内容中...
                  </span>
                </div>
              )}
              <Editor
                height="100%"
                theme="vs-dark"
                language={getLanguageType(contentPath?.fileName || '')}
                value={contentDraft}
                onChange={(value) => setContentDraft(value || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  fontFamily: '"Fira Code", Monaco, "Courier New", monospace',
                  wordWrap: 'on',
                  padding: { top: 18, bottom: 18 },
                  smoothScrolling: true,
                  lineHeight: 1.55,
                }}
              />
            </div>
          </div>
        </div>
      )}

      {pathGroupEditorOpen && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div
            data-testid="path-group-editor-dialog"
            className="flex max-h-[92vh] w-full max-w-[560px] flex-col overflow-hidden rounded-xl border border-white/10 bg-[#1f1f1f] text-gray-100 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4 border-b border-white/10 bg-[#111111] px-7 py-5">
              <div className="min-w-0">
                <h4 className="text-xl font-extrabold text-white">新增路径</h4>
                <p className="mt-1 truncate text-sm font-bold text-gray-400">
                  {getPathSetTitle(selectedPathSetSection || primaryPathSetSection || {})}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setPathGroupEditorOpen(false)}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="bg-[#242424] px-7 py-6">
              <label className="mb-2.5 block text-sm font-extrabold text-gray-400">公共相对路径</label>
              <input
                type="text"
                value={pathGroupDraft.basePath || ''}
                onChange={(event) => setPathGroupDraft({ ...pathGroupDraft, basePath: event.target.value })}
                className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                placeholder="c12-mtp-web-service/src/main/java"
                autoFocus
              />
            </div>

            <div className="border-t border-white/10 bg-[#171717] p-5">
              <button
                type="button"
                onClick={savePathGroupDraft}
                disabled={savingPathGroup}
                className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#0f172a] px-4 py-3 text-base font-extrabold text-white transition-colors hover:bg-[#111c34] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {savingPathGroup ? <RefreshCw size={18} className="animate-spin" /> : <Save size={18} />}
                保存路径
              </button>
            </div>
          </div>
        </div>
      )}

      {placeholderPath && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div className="flex max-h-[92vh] w-full max-w-5xl flex-col overflow-hidden rounded-xl border border-white/10 bg-[#1f1f1f] text-gray-100 shadow-2xl">
            <div className="flex items-start justify-between gap-4 border-b border-white/10 bg-[#111111] px-7 py-5">
              <div className="min-w-0">
                <h4 className="text-xl font-extrabold text-white">动态占位符</h4>
                <p className="mt-1 truncate font-mono text-sm font-bold text-gray-400" title={formatPathLabel(placeholderPath || {})}>
                  {formatPathLabel(placeholderPath || {})}
                </p>
              </div>
              <button
                type="button"
                onClick={closePathPlaceholderDialog}
                disabled={savingPlaceholders}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto bg-[#242424] px-7 py-6">
              <datalist id="path-placeholder-key-options">
                {pathPlaceholderOptions.map((item) => (
                  <option key={item.key} value={item.key}>
                    {item.description || item.value || item.key}
                  </option>
                ))}
              </datalist>
              <div className="overflow-hidden rounded-lg border border-white/15">
                <div className="grid grid-cols-[minmax(160px,0.9fr)_minmax(180px,1.2fr)_minmax(180px,1.2fr)_48px] border-b border-white/15 bg-[#171717] text-sm font-extrabold text-gray-400">
                  <div className="px-4 py-3">占位符 key</div>
                  <div className="px-4 py-3">描述</div>
                  <div className="px-4 py-3">默认 value</div>
                  <div />
                </div>
                {loadingPlaceholders ? (
                  <div className="px-4 py-8 text-center text-sm font-bold text-gray-500">正在读取占位符...</div>
                ) : placeholderRows.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm font-bold text-gray-500">暂无占位符</div>
                ) : placeholderRows.map((row, index) => (
                  <div
                    key={`${row.key}-${index}`}
                    className="grid grid-cols-[minmax(160px,0.9fr)_minmax(180px,1.2fr)_minmax(180px,1.2fr)_48px] items-center border-b border-white/10 last:border-b-0"
                  >
                    <div className="p-2">
                      <input
                        value={row.key || ''}
                        onChange={(event) => updatePathPlaceholderKey(index, event.target.value)}
                        list="path-placeholder-key-options"
                        className="w-full rounded-lg border border-white/15 bg-[#111111] px-3 py-2 font-mono text-sm font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-teal-400 focus:ring-2 focus:ring-teal-500/20"
                        placeholder="menu_parent_id"
                        autoFocus={index === 0}
                      />
                    </div>
                    <div className="p-2">
                      <input
                        value={row.description || ''}
                        onChange={(event) => updatePathPlaceholderRow(index, { description: event.target.value })}
                        className="w-full rounded-lg border border-white/15 bg-[#111111] px-3 py-2 text-sm font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-teal-400 focus:ring-2 focus:ring-teal-500/20"
                        placeholder="菜单父级 id"
                      />
                    </div>
                    <div className="p-2">
                      <input
                        value={row.value || ''}
                        onChange={(event) => updatePathPlaceholderRow(index, { value: event.target.value })}
                        className="w-full rounded-lg border border-white/15 bg-[#111111] px-3 py-2 font-mono text-sm font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-teal-400 focus:ring-2 focus:ring-teal-500/20"
                        placeholder="2063000000000000004"
                      />
                    </div>
                    <div className="flex justify-center p-2">
                      <button
                        type="button"
                        onClick={() => removePathPlaceholderRow(index)}
                        className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-500/10 hover:text-red-300"
                        title="删除"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              <button
                type="button"
                onClick={addPathPlaceholderRow}
                className="mt-4 inline-flex items-center gap-2 rounded-lg border border-white/15 px-4 py-2 text-sm font-extrabold text-gray-100 transition-colors hover:bg-white/10"
              >
                <Plus size={15} />
                新增占位符
              </button>
            </div>

            <div className="flex justify-end gap-2 border-t border-white/10 bg-[#171717] px-7 py-5">
              <button
                type="button"
                onClick={closePathPlaceholderDialog}
                disabled={savingPlaceholders}
                className="rounded-lg px-5 py-2.5 text-sm font-extrabold text-gray-300 transition-colors hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
              >
                取消
              </button>
              <button
                type="button"
                onClick={savePathPlaceholders}
                disabled={savingPlaceholders || loadingPlaceholders}
                className="inline-flex items-center gap-2 rounded-lg bg-[#0f172a] px-5 py-2.5 text-sm font-extrabold text-white transition-colors hover:bg-[#111c34] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {savingPlaceholders ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {pathEditorOpen && (
        <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm">
          <div
            data-testid="path-editor-dialog"
            className="flex max-h-[92vh] w-full max-w-[560px] flex-col overflow-hidden rounded-xl border border-white/10 bg-[#1f1f1f] text-gray-100 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4 border-b border-white/10 bg-[#111111] px-7 py-5">
              <div className="min-w-0">
                <h4 className="text-xl font-extrabold text-white">{pathDraft.ID ? '编辑路径' : '新增路径'}</h4>
                <p className="mt-1 truncate font-mono text-sm font-bold text-gray-400" title={formatPathLabel(pathDraft)}>
                  {pathDraft.ID ? formatPathLabel(pathDraft) : '编辑代码模版文件路径'}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-1">
                <button
                  type="button"
                  onClick={() => setPathEditorOpen(false)}
                  className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
                  title="关闭"
                >
                  <X size={18} />
                </button>
              </div>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto bg-[#242424] px-7 py-6">
              <div className="space-y-6">
                {activePathGroup ? (
                  <>
                    <div>
                      <label className="mb-2.5 block text-sm font-extrabold text-gray-400">公共相对路径</label>
                      {pathGroupSwitchOptions.length > 1 ? (
                        <select
                          value={activePathGroup.key}
                          onChange={(event) => switchActivePathGroup(event.target.value)}
                          className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-white outline-none transition focus:border-white/30 focus:ring-2 focus:ring-white/10"
                          title={activePathGroup.basePath || '/'}
                        >
                          {pathGroupSwitchOptions.map((option) => (
                            <option key={option.key} value={option.key}>
                              {option.label}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <div
                          className="break-all rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-gray-300"
                          title={activePathGroup.basePath || '/'}
                        >
                          {activePathGroup.basePath || '/'}
                        </div>
                      )}
                    </div>

                    <div>
                      <label className="mb-2.5 block text-sm font-extrabold text-gray-400">剩余父级</label>
                      <input
                        type="text"
                        value={getPathParentSuffix(pathDraft, activePathGroup.basePath || '')}
                        onChange={(event) => setPathDraft({
                          ...pathDraft,
                          fileUrl: buildPathParentWithSuffix(activePathGroup.basePath || '', event.target.value),
                        })}
                        className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                        placeholder="/"
                        autoFocus
                      />
                    </div>
                  </>
                ) : (
                  <div>
                    <label className="mb-2.5 block text-sm font-extrabold text-gray-400">父级目录</label>
                    <input
                      type="text"
                      value={pathDraft.fileUrl || ''}
                      onChange={(event) => setPathDraft({ ...pathDraft, fileUrl: event.target.value })}
                      className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                      placeholder="../"
                      autoFocus
                    />
                  </div>
                )}

                <div>
                  <label className="mb-2.5 block text-sm font-extrabold text-gray-400">文件路径</label>
                  <input
                    type="text"
                    value={pathDraft.fileName || ''}
                    onChange={(event) => setPathDraft({ ...pathDraft, fileName: event.target.value })}
                    className="w-full rounded-lg border border-white/15 bg-[#111111] px-4 py-3 font-mono text-lg font-bold text-white outline-none transition placeholder:text-gray-600 focus:border-white/30 focus:ring-2 focus:ring-white/10"
                    placeholder="create_btStation.sql"
                  />
                </div>

                <div className="rounded-lg border border-white/15 bg-[#111111] p-4">
                  <div className="mb-2 text-sm font-extrabold text-gray-400">完整相对路径</div>
                  <div className="break-all font-mono text-lg font-extrabold text-gray-100">
                    {formatPathLabel(pathDraft)}
                  </div>
                </div>

                <label className="flex cursor-pointer items-center justify-between gap-4 rounded-lg border border-white/15 bg-[#111111] px-4 py-3">
                  <span className="text-lg font-extrabold text-gray-100">启用</span>
                  <input
                    type="checkbox"
                    checked={Number(pathDraft.enabled || 0) === 1}
                    onChange={(event) => setPathDraft({ ...pathDraft, enabled: event.target.checked ? 1 : 0 })}
                    className="h-5 w-5 accent-slate-200"
                  />
                </label>

                <label className="flex cursor-pointer items-center justify-between gap-4 rounded-lg border border-white/15 bg-[#111111] px-4 py-3">
                  <span className="text-lg font-extrabold text-gray-100">增量</span>
                  <input
                    type="checkbox"
                    checked={Number(pathDraft.incremented || 0) === 1}
                    onChange={(event) => setPathDraft({ ...pathDraft, incremented: event.target.checked ? 1 : 0 })}
                    className="h-5 w-5 accent-slate-200"
                  />
                </label>
              </div>
            </div>

            <div className="space-y-3 border-t border-white/10 bg-[#171717] p-5">
              {pathDraft.ID && (
                <button
                  type="button"
                  onClick={() => togglePathEnabled(pathDraft)}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-white/15 px-4 py-3 text-base font-extrabold text-gray-100 transition-colors hover:bg-white/10"
                >
                  <Check size={18} />
                  {Number(pathDraft.enabled || 0) === 1 ? '停用路径' : '启用路径'}
                </button>
              )}
              <button
                type="button"
                onClick={savePath}
                disabled={savingPath}
                className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#0f172a] px-4 py-3 text-base font-extrabold text-white transition-colors hover:bg-[#111c34] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {savingPath ? <RefreshCw size={18} className="animate-spin" /> : <Save size={18} />}
                保存路径
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
