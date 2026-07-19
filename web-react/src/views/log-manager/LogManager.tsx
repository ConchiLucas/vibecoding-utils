import React, { useCallback, useEffect, useMemo, useState } from 'react';
import clsx from 'clsx';
import toast from 'react-hot-toast';
import {
  Box,
  Database,
  Folder,
  Layers,
  Check,
  Pencil,
  Play,
  RefreshCw,
  ScrollText,
  Search,
  Server,
  Terminal,
  TerminalSquare,
  Trash2,
  X,
} from 'lucide-react';
import {
  DockerServiceSummary,
  LogProject,
  LogProjectGroup,
  LogProjectRoute,
  deleteLogProject,
  getLogManagerDockerServices,
  getLogProjectGroups,
  getLogProjectPage,
  saveOrUpdateLogProject,
} from '../../api/logManager';
import DeployLogPanel from '../../components/DeployLogPanel';
import { useProjectStore } from '../../stores/useProjectStore';
import { useUserStore } from '../../stores/useUserStore';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

type ActiveTab = 'service' | 'docker';

interface LogPanelState {
  open: boolean;
  projectId: number;
  projectName: string;
  envKey: string;
  routeName: string;
  mode: 'deploy' | 'stop' | 'restart' | 'logs';
  streamPath?: string;
  panelTitle?: string;
  introText?: string;
}

const emptyLogPanel: LogPanelState = {
  open: false,
  projectId: 0,
  projectName: '',
  envKey: '',
  routeName: '',
  mode: 'deploy',
};

function getLanguageIcon(lang?: string) {
  const value = String(lang || '').toLowerCase();
  if (value.includes('vue') || value.includes('react')) return <Box size={18} className="text-blue-500" />;
  if (value.includes('go')) return <TerminalSquare size={18} className="text-cyan-600" />;
  if (value.includes('java')) return <Server size={18} className="text-orange-500" />;
  if (value.includes('python')) return <TerminalSquare size={18} className="text-yellow-600" />;
  return <Terminal size={18} className="text-gray-500" />;
}

function isScriptLaunchCommand(command?: string) {
  const normalized = String(command || '').trim().toLowerCase();
  if (!normalized) return false;
  return /^(sh|bash|zsh|fish|source|make|npm|pnpm|yarn|node|python|python3|java|go|mvn|gradle)\b/.test(normalized) ||
    normalized.startsWith('./') ||
    /\.sh(\s|$)/.test(normalized);
}

function isDockerComposeLaunchRoute(route: LogProjectRoute) {
  const backendType = String(route.routeType || '').trim();
  if (backendType) return backendType === 'docker_compose';
  if (isFileLogRoute(route)) return false;
  const launchCommand = route.localExecuteCommand || route.localStartCommand;
  const commandText = `${route.localExecuteCommand || ''} ${route.localStartCommand || ''}`.toLowerCase();
  const directComposeCommand = /\b(docker\s+compose|docker-compose)\b/.test(commandText);
  const composeMarked = Boolean(route.dockerComposeDeploy) || route.buildType === 'docker_compose_deploy';
  if (composeMarked) return true;
  return !isScriptLaunchCommand(launchCommand) && directComposeCommand;
}

function isDockerDeployScriptRoute(route: LogProjectRoute) {
  const launchCommand = route.localExecuteCommand || route.localStartCommand;
  return isDockerComposeLaunchRoute(route) && isScriptLaunchCommand(launchCommand);
}

function isFileLogRoute(route: LogProjectRoute) {
  const backendType = String(route.routeType || '').trim();
  if (backendType) return backendType === 'file_log';
  return route.buildType === 'file_log' || Boolean(String(route.logFilePath || '').trim());
}

function isScriptRoute(route: LogProjectRoute) {
  const backendType = String(route.routeType || '').trim();
  if (backendType) return backendType === 'script';
  return !isDockerComposeLaunchRoute(route) && !isFileLogRoute(route);
}

function isServiceLogRoute(route: LogProjectRoute) {
  return isScriptRoute(route) || isFileLogRoute(route);
}

function sortLogRoutes(routes: LogProjectRoute[]) {
  return [...routes].sort((a, b) => (a.sort || 0) - (b.sort || 0) || a.ID - b.ID);
}

function projectRoutesForTab(project: LogProject, tab: ActiveTab) {
  const routes = sortLogRoutes(project.routes || []);
  if (tab === 'docker') return routes.filter(isDockerComposeLaunchRoute);
  const fileLogRoutes = routes.filter(isFileLogRoute);
  return fileLogRoutes.length > 0 ? fileLogRoutes : routes.filter(isScriptRoute);
}

function projectMatchesActiveTab(project: LogProject, tab: ActiveTab) {
  return projectRoutesForTab(project, tab).length > 0;
}

function ServiceRunningDot({ running }: { running?: boolean }) {
  return (
    <span
      className={clsx(
        'inline-flex h-2.5 w-2.5 shrink-0 rounded-full ring-2 ring-white',
        running ? 'bg-emerald-500 shadow-[0_0_0_4px_rgba(16,185,129,0.14)]' : 'bg-red-500 shadow-[0_0_0_4px_rgba(239,68,68,0.12)]'
      )}
      title={running ? '运行中' : '已停止'}
    />
  );
}

function serviceStatusKey(item: Pick<DockerServiceSummary, 'routeId' | 'serviceName'>) {
  return `${item.routeId}:${item.serviceName || ''}`;
}

function mergeServiceStatuses(current: DockerServiceSummary[], next: DockerServiceSummary[]) {
  const statusMap = new Map(next.map(item => [serviceStatusKey(item), item]));
  if (current.length === 0) return next;
  return current.map(item => {
    const status = statusMap.get(serviceStatusKey(item));
    return status ? { ...item, running: status.running } : item;
  });
}

function getSseBaseUrl() {
  const isWails = Boolean((window as any).__wails__) ||
    window.location.protocol === 'wails:' ||
    window.location.hostname === 'wails.localhost';
  return isWails ? 'http://127.0.0.1:48009' : (import.meta.env.VITE_BASE_API || '/api');
}

export default function LogManager() {
  const { activeProject, activeProjectId } = useProjectStore();
  const token = useUserStore(state => state.token);
  const [projects, setProjects] = useState<LogProject[]>([]);
  const [groups, setGroups] = useState<LogProjectGroup[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState<ActiveTab>('service');
  const [loading, setLoading] = useState(false);
  const [dockerLoading, setDockerLoading] = useState(false);
  const [dockerServices, setDockerServices] = useState<DockerServiceSummary[]>([]);
  const [logPanel, setLogPanel] = useState<LogPanelState>(emptyLogPanel);
  const [editingProjectId, setEditingProjectId] = useState<number | null>(null);
  const [editingProjectName, setEditingProjectName] = useState('');
  const [editingProjectPathId, setEditingProjectPathId] = useState<number | null>(null);
  const [editingProjectPath, setEditingProjectPath] = useState('');
  const [projectActionLoadingId, setProjectActionLoadingId] = useState<number | null>(null);
  const { confirm, dialogProps } = useConfirm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const projectRes: any = await getLogProjectPage({
        page: 1,
        pageSize: 500,
        ...(activeProjectId ? { projectConfigId: activeProjectId } : {}),
        ...(!activeProjectId && activeProject ? { projectConfigName: activeProject } : {}),
      });
      const groupRes: any = await getLogProjectGroups();
      if (projectRes.code === 0) setProjects(projectRes.data?.list || []);
      if (groupRes.code === 0) setGroups(groupRes.data || []);
    } catch {
      toast.error('日志项目列表加载失败');
    } finally {
      setLoading(false);
    }
  }, [activeProject, activeProjectId]);

  useEffect(() => {
    setSelectedGroupId(null);
    setSelectedProjectId(null);
    setDockerServices([]);
    void fetchData();
  }, [fetchData]);

  const tabProjects = useMemo(
    () => projects.filter(project => projectMatchesActiveTab(project, activeTab)),
    [projects, activeTab]
  );

  const filteredProjects = useMemo(() => {
    const keyword = searchQuery.trim().toLowerCase();
    return tabProjects.filter(project => {
      const groupOk = selectedGroupId === null || project.groupId === selectedGroupId;
      const searchOk = !keyword ||
        String(project.projectName || '').toLowerCase().includes(keyword) ||
        String(project.description || '').toLowerCase().includes(keyword);
      return groupOk && searchOk;
    });
  }, [tabProjects, searchQuery, selectedGroupId]);

  useEffect(() => {
    if (filteredProjects.length === 0) {
      setSelectedProjectId(null);
      return;
    }
    if (!selectedProjectId || !filteredProjects.some(project => project.ID === selectedProjectId)) {
      setSelectedProjectId(filteredProjects[0].ID);
    }
  }, [filteredProjects, selectedProjectId]);

  const selectedProject = useMemo(
    () => projects.find(project => project.ID === selectedProjectId) || null,
    [projects, selectedProjectId]
  );

  const localRoutes = useMemo(() => {
    if (!selectedProject) return [];
    return sortLogRoutes(selectedProject.routes || []);
  }, [selectedProject]);

  const serviceRoutes = useMemo(
    () => localRoutes.filter(isServiceLogRoute),
    [localRoutes]
  );

  const serviceLogRoutes = useMemo(
    () => localRoutes.filter(isFileLogRoute),
    [localRoutes]
  );

  const executableServiceRoutes = useMemo(
    () => localRoutes.filter(isScriptRoute),
    [localRoutes]
  );

  const dockerComposeRoutes = useMemo(
    () => localRoutes.filter(isDockerComposeLaunchRoute),
    [localRoutes]
  );

  const hasDockerDeployScript = useMemo(
    () => dockerComposeRoutes.some(isDockerDeployScriptRoute),
    [dockerComposeRoutes]
  );

  const activeRoutes = activeTab === 'docker' ? dockerComposeRoutes : (serviceLogRoutes.length > 0 ? serviceLogRoutes : serviceRoutes);
  const activeExecutableRoutes = activeTab === 'docker' ? dockerComposeRoutes : executableServiceRoutes;
  const dockerDisplayCount = dockerServices.length > 0 ? dockerServices.length : dockerComposeRoutes.length;
  const activeDisplayCount = activeTab === 'docker' ? dockerDisplayCount : activeRoutes.length;
  const activeDisplayLabel = activeTab === 'docker'
    ? (dockerServices.length > 0 ? 'Compose 服务' : 'Compose 路线')
    : '服务路线';

  const selectedGroupName = selectedGroupId === null
    ? '全部项目'
    : groups.find(group => group.ID === selectedGroupId)?.groupName || '未知项目组';
  const activeProjectLabel = activeProject || '未选择项目';

  const loadDockerServices = async () => {
    if (!selectedProject) {
      setDockerServices([]);
      return;
    }
    setDockerServices([]);
    setDockerLoading(true);
    try {
      const res: any = await getLogManagerDockerServices(selectedProject.ID, activeTab);
      if (res.code === 0) {
        setDockerServices(res.data || []);
      } else {
        setDockerServices([]);
      }
    } catch {
      setDockerServices([]);
      toast.error(activeTab === 'docker' ? 'Docker 日志入口加载失败' : '服务日志入口加载失败');
    } finally {
      setDockerLoading(false);
    }
  };

  useEffect(() => {
    loadDockerServices();
  }, [activeTab, selectedProjectId, selectedProject?.localProjectPath]);

  useEffect(() => {
    if (!selectedProjectId) return;
    const sseBaseUrl = getSseBaseUrl();
    const url = `${sseBaseUrl}/logManager/serviceStatusStream/${selectedProjectId}?scope=${activeTab}&token=${encodeURIComponent(token || '')}`;
    const es = new EventSource(url);

    es.addEventListener('status', (event: MessageEvent) => {
      try {
        const next = JSON.parse(event.data) as DockerServiceSummary[];
        if (!Array.isArray(next)) return;
        setDockerServices(previous => mergeServiceStatuses(previous, next));
      } catch {
        // Ignore malformed status frames; the next SSE tick will repair the state.
      }
    });

    es.onerror = () => {
      // EventSource will retry automatically while the page is still on the same project/tab.
    };

    return () => {
      es.close();
    };
  }, [activeTab, selectedProjectId, token]);

  const openGroupStream = (action: 'start' | 'stop' | 'restart') => {
    if (!selectedProject) return;
    if (activeExecutableRoutes.length === 0) {
      toast.error(activeTab === 'docker' ? '当前项目没有可执行的 docker-compose 路线' : '当前项目没有可执行的服务启动路线');
      return;
    }
    const isStop = action === 'stop';
    const isRestart = action === 'restart';
    const scope = activeTab === 'docker' ? 'docker' : 'service';
    const scopeLabel = activeTab === 'docker' ? 'Docker Compose 服务' : '服务';
    const isDockerDeploy = activeTab === 'docker' && action === 'start' && hasDockerDeployScript;
    const actionLabel = isDockerDeploy ? '部署' : isRestart ? '重启' : isStop ? '关闭' : '启动';
    const targetCount = activeTab === 'docker' && dockerServices.length > 0 ? dockerServices.length : activeExecutableRoutes.length;
    setLogPanel({
      open: true,
      projectId: selectedProject.ID,
      projectName: selectedProject.projectName,
      envKey: `${scope}-group`,
      routeName: `${targetCount} 个${scopeLabel}`,
      mode: isRestart ? 'restart' : isStop ? 'stop' : 'deploy',
      streamPath: `/logManager/serviceGroupStream/${selectedProject.ID}?action=${action}&scope=${scope}`,
      panelTitle: `${scopeLabel}${actionLabel}日志`,
      introText: `🚀 已连接，开始${actionLabel} [${selectedProject.projectName}] 的 ${targetCount} 个${scopeLabel}...`,
    });
  };

  const openDockerLog = (item: DockerServiceSummary) => {
    if (!selectedProject) return;
    const isDockerTab = activeTab === 'docker';
    const servicePart = item.serviceName ? `&service=${encodeURIComponent(item.serviceName)}` : '';
    const displayName = item.serviceName ? `${item.routeName} / ${item.serviceName}` : item.routeName;
    setLogPanel({
      open: true,
      projectId: selectedProject.ID,
      projectName: selectedProject.projectName,
      envKey: String(item.routeId),
      routeName: displayName,
      mode: 'logs',
      streamPath: `/logManager/dockerLogStream/${selectedProject.ID}?env=${item.routeId}${servicePart}`,
      panelTitle: isDockerTab ? 'Docker 实时日志' : '服务实时日志',
      introText: `🚀 已连接，开始读取 [${selectedProject.projectName}] - ${displayName} 的${isDockerTab ? ' Docker 日志' : '服务日志'}...`,
    });
  };

  const openServiceRestart = (item: DockerServiceSummary) => {
    if (!selectedProject) return;
    const displayName = item.serviceName ? `${item.routeName} / ${item.serviceName}` : item.routeName;
    setLogPanel({
      open: true,
      projectId: selectedProject.ID,
      projectName: selectedProject.projectName,
      envKey: String(item.routeId),
      routeName: displayName,
      mode: 'restart',
      streamPath: `/logManager/restartStream/${selectedProject.ID}?env=${item.routeId}`,
      panelTitle: '服务重启日志',
      introText: `🚀 已连接，开始重启 [${selectedProject.projectName}] - ${displayName}...`,
    });
  };

  const openDockerRestart = (item: DockerServiceSummary) => {
    if (!selectedProject) return;
    const servicePart = item.serviceName ? `&service=${encodeURIComponent(item.serviceName)}` : '';
    const displayName = item.serviceName ? `${item.routeName} / ${item.serviceName}` : item.routeName;
    setLogPanel({
      open: true,
      projectId: selectedProject.ID,
      projectName: selectedProject.projectName,
      envKey: String(item.routeId),
      routeName: displayName,
      mode: 'restart',
      streamPath: `/logManager/restartStream/${selectedProject.ID}?env=${item.routeId}${servicePart}`,
      panelTitle: item.serviceName ? 'Docker 服务重启日志' : 'Docker 路线重启日志',
      introText: `🚀 已连接，开始重启 [${selectedProject.projectName}] - ${displayName}...`,
    });
  };

  const startEditProject = (project: LogProject, event: React.MouseEvent) => {
    event.stopPropagation();
    setEditingProjectId(project.ID);
    setEditingProjectName(project.projectName);
  };

  const cancelEditProject = (event?: React.MouseEvent) => {
    event?.stopPropagation();
    setEditingProjectId(null);
    setEditingProjectName('');
  };

  const saveProjectName = async (project: LogProject, event?: React.MouseEvent) => {
    event?.stopPropagation();
    const nextName = editingProjectName.trim();
    if (!nextName) {
      toast.error('项目名称不能为空');
      return;
    }
    if (nextName === project.projectName) {
      cancelEditProject();
      return;
    }

    setProjectActionLoadingId(project.ID);
    try {
      const { routes: _routes, ...payload } = project;
      const res: any = await saveOrUpdateLogProject({
        ...payload,
        projectName: nextName,
      });
      if (res.code !== 0) {
        toast.error(res.msg || '修改项目名称失败');
        return;
      }
      setProjects(previous => previous.map(item => item.ID === project.ID ? { ...item, projectName: nextName } : item));
      setEditingProjectId(null);
      setEditingProjectName('');
      toast.success('项目名称已修改');
      void fetchData();
    } catch {
      toast.error('修改项目名称失败');
    } finally {
      setProjectActionLoadingId(null);
    }
  };

  const startEditProjectPath = (project: LogProject) => {
    setEditingProjectPathId(project.ID);
    setEditingProjectPath(project.localProjectPath || '');
  };

  const cancelEditProjectPath = () => {
    setEditingProjectPathId(null);
    setEditingProjectPath('');
  };

  const saveProjectPath = async (project: LogProject) => {
    const nextPath = editingProjectPath.trim();
    if (!nextPath) {
      toast.error('项目目录不能为空');
      return;
    }
    if (nextPath === String(project.localProjectPath || '').trim()) {
      cancelEditProjectPath();
      return;
    }

    setProjectActionLoadingId(project.ID);
    try {
      const { routes: _routes, ...payload } = project;
      const res: any = await saveOrUpdateLogProject({
        ...payload,
        localProjectPath: nextPath,
      });
      if (res.code !== 0) {
        toast.error(res.msg || '修改项目目录失败');
        return;
      }
      setProjects(previous => previous.map(item => item.ID === project.ID ? { ...item, localProjectPath: nextPath } : item));
      setDockerServices([]);
      cancelEditProjectPath();
      toast.success('项目目录及旧目录下的路线路径已更新');
      await fetchData();
    } catch {
      toast.error('修改项目目录失败');
    } finally {
      setProjectActionLoadingId(null);
    }
  };

  const deleteProject = async (project: LogProject, event: React.MouseEvent) => {
    event.stopPropagation();
    const routeCount = (project.routes || []).length;
    const ok = await confirm(
      `确定删除日志项目「${project.projectName}」吗？删除后会同时清理该项目下的 ${routeCount} 条服务路线配置。`,
      {
        title: '删除日志项目',
        confirmText: '确定删除',
        cancelText: '取消',
      }
    );
    if (!ok) return;

    setProjectActionLoadingId(project.ID);
    try {
      const res: any = await deleteLogProject(project.ID);
      if (res.code !== 0) {
        toast.error(res.msg || '删除日志项目失败');
        return;
      }
      setProjects(previous => previous.filter(item => item.ID !== project.ID));
      if (selectedProjectId === project.ID) {
        setSelectedProjectId(null);
        setDockerServices([]);
      }
      toast.success('日志项目已删除');
      void fetchData();
    } catch {
      toast.error('删除日志项目失败');
    } finally {
      setProjectActionLoadingId(null);
    }
  };

  return (
    <div className="w-full min-h-[calc(100vh-64px)] flex bg-white text-gray-900">
      <aside className="w-72 shrink-0 border-r border-gray-200 bg-white flex flex-col min-h-[calc(100vh-64px)]">
        <div className="p-4 border-b border-gray-100">
          <div className="flex rounded-lg border border-gray-200 bg-gray-50 p-1">
            <button
              onClick={() => setActiveTab('service')}
              className={clsx(
                'flex-1 h-9 rounded-md text-xs font-bold transition-colors flex items-center justify-center gap-1.5',
                activeTab === 'service' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-800'
              )}
            >
              <ScrollText size={14} /> 服务日志
            </button>
            <button
              onClick={() => setActiveTab('docker')}
              className={clsx(
                'flex-1 h-9 rounded-md text-xs font-bold transition-colors flex items-center justify-center gap-1.5',
                activeTab === 'docker' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-800'
              )}
            >
              <Layers size={14} /> Docker日志
            </button>
          </div>

          <div className="relative mt-4">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              value={searchQuery}
              onChange={event => setSearchQuery(event.target.value)}
              placeholder="搜索项目名称..."
              className="w-full h-10 rounded-lg border border-gray-200 bg-gray-50 pl-9 pr-3 text-sm outline-none focus:border-gray-300 focus:bg-white focus:ring-2 focus:ring-black/5"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-4 pt-4 pb-5 space-y-1 scrollbar-thin">
          {loading ? (
            <div className="py-12 text-center text-sm text-gray-400">项目加载中...</div>
          ) : filteredProjects.length === 0 ? (
            <div className="py-12 text-center text-sm text-gray-400">
              {activeProject
                ? `当前项目「${activeProject}」下暂无${activeTab === 'docker' ? 'Docker' : '服务'}日志项目`
                : `暂无${activeTab === 'docker' ? 'Docker' : '服务'}日志项目`}
            </div>
          ) : (
            filteredProjects.map(project => {
              const active = project.ID === selectedProjectId;
              const routeCount = projectRoutesForTab(project, activeTab).length;
              const editing = editingProjectId === project.ID;
              const actionLoading = projectActionLoadingId === project.ID;
              return (
                <div
                  key={project.ID}
                  onClick={() => setSelectedProjectId(project.ID)}
                  onKeyDown={event => {
                    if (event.target !== event.currentTarget) return;
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      setSelectedProjectId(project.ID);
                    }
                  }}
                  role="button"
                  tabIndex={0}
                  className={clsx(
                    'group w-full rounded-xl border px-3 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-blue-300',
                    active ? 'border-blue-200 bg-blue-50 shadow-sm' : 'border-transparent hover:border-gray-200 hover:bg-gray-50'
                  )}
                >
                  <div className="flex items-start gap-3">
                    <div className={clsx(
                      'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border',
                      active ? 'border-blue-200 bg-white' : 'border-gray-200 bg-gray-50'
                    )}>
                      {getLanguageIcon(project.computerLanguage)}
                    </div>
                    <div className="min-w-0 flex-1">
                      {editing ? (
                        <div className="flex items-center gap-1" onClick={event => event.stopPropagation()}>
                          <input
                            value={editingProjectName}
                            disabled={actionLoading}
                            autoFocus
                            spellCheck={false}
                            onChange={event => setEditingProjectName(event.target.value)}
                            onKeyDown={event => {
                              if (event.key === 'Enter') {
                                void saveProjectName(project);
                              } else if (event.key === 'Escape') {
                                cancelEditProject();
                              }
                            }}
                            className="h-8 min-w-0 flex-1 rounded-lg border border-blue-200 bg-white px-2 text-sm font-bold text-blue-900 outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-200"
                          />
                          <button
                            type="button"
                            disabled={actionLoading}
                            onClick={event => void saveProjectName(project, event)}
                            className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-emerald-200 bg-emerald-50 text-emerald-600 transition-colors hover:bg-emerald-100 disabled:cursor-wait disabled:opacity-50"
                            title="保存项目名称"
                          >
                            <Check size={14} />
                          </button>
                          <button
                            type="button"
                            disabled={actionLoading}
                            onClick={cancelEditProject}
                            className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-500 transition-colors hover:bg-gray-100 disabled:cursor-wait disabled:opacity-50"
                            title="取消修改"
                          >
                            <X size={14} />
                          </button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <div className={clsx('min-w-0 flex-1 truncate text-sm font-bold', active ? 'text-blue-900' : 'text-gray-900')}>
                            {project.projectName}
                          </div>
                          <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                            <button
                              type="button"
                              disabled={actionLoading}
                              onClick={event => startEditProject(project, event)}
                              className="inline-flex h-7 w-7 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-white hover:text-blue-600 disabled:cursor-wait disabled:opacity-50"
                              title="修改项目名称"
                            >
                              <Pencil size={13} />
                            </button>
                            <button
                              type="button"
                              disabled={actionLoading}
                              onClick={event => void deleteProject(project, event)}
                              className="inline-flex h-7 w-7 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 disabled:cursor-wait disabled:opacity-50"
                              title="删除日志项目"
                            >
                              <Trash2 size={13} />
                            </button>
                          </div>
                        </div>
                      )}
                      <div className="mt-1 flex items-center gap-2 text-xs text-gray-400">
                        <span className="truncate">{project.computerLanguage || 'unknown'}</span>
                        <span className="h-1 w-1 rounded-full bg-gray-300" />
                        <span>{routeCount} 路线</span>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </aside>

      <section className="flex-1 min-w-0 bg-gray-50">
        <div className="border-b border-gray-200 bg-white px-8 py-5">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4 min-w-0">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-gray-50">
                {activeTab === 'docker' ? <Layers size={22} className="text-emerald-600" /> : <ScrollText size={22} className="text-blue-600" />}
              </div>
              <div className="min-w-0">
                <h1 className="text-xl font-extrabold tracking-tight text-gray-900">日志管理</h1>
                <p className="mt-1 text-sm text-gray-500 truncate">
                  {selectedProject ? `${selectedGroupName} / ${selectedProject.projectName}` : '选择一个项目查看服务日志'}
                </p>
              </div>
            </div>
            <div className="hidden lg:flex items-center gap-2">
              <div className="inline-flex items-center gap-2 rounded-xl border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-500">
                <Folder size={14} />
                <span>项目</span>
                <span className="font-semibold text-blue-700">{activeProjectLabel}</span>
              </div>
              {selectedProject && (
                <>
                  <button
                    type="button"
                    onClick={() => startEditProjectPath(selectedProject)}
                    className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-600 transition-colors hover:border-blue-200 hover:bg-blue-50 hover:text-blue-700"
                    title="修改项目目录"
                  >
                    <Pencil size={13} />
                    修改目录
                  </button>
                  <div className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-500">
                    <Database size={14} />
                    <span className="font-semibold text-gray-700">{activeDisplayCount}</span>
                    <span>{activeDisplayLabel}</span>
                  </div>
                </>
              )}
            </div>
          </div>
          {selectedProject && editingProjectPathId === selectedProject.ID && (
            <div className="mt-4 flex items-center gap-2 rounded-xl border border-blue-200 bg-blue-50 p-3">
              <Folder size={16} className="shrink-0 text-blue-600" />
              <input
                value={editingProjectPath}
                disabled={projectActionLoadingId === selectedProject.ID}
                autoFocus
                spellCheck={false}
                onChange={event => setEditingProjectPath(event.target.value)}
                onKeyDown={event => {
                  if (event.key === 'Enter') {
                    void saveProjectPath(selectedProject);
                  } else if (event.key === 'Escape') {
                    cancelEditProjectPath();
                  }
                }}
                placeholder="输入新的项目绝对路径"
                className="h-9 min-w-0 flex-1 rounded-lg border border-blue-200 bg-white px-3 font-mono text-xs text-gray-800 outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-200"
              />
              <span className="hidden xl:inline text-xs text-blue-600">旧目录下的路线和日志路径会同步迁移</span>
              <button
                type="button"
                disabled={projectActionLoadingId === selectedProject.ID}
                onClick={() => void saveProjectPath(selectedProject)}
                className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-blue-600 px-3 text-xs font-bold text-white hover:bg-blue-700 disabled:cursor-wait disabled:bg-blue-300"
              >
                <Check size={14} /> 保存
              </button>
              <button
                type="button"
                disabled={projectActionLoadingId === selectedProject.ID}
                onClick={cancelEditProjectPath}
                className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-blue-200 bg-white text-gray-500 hover:bg-gray-50 disabled:cursor-wait disabled:opacity-50"
                title="取消修改"
              >
                <X size={14} />
              </button>
            </div>
          )}
        </div>

        <div className="p-8">
          {!selectedProject ? (
            <div className="flex min-h-[420px] items-center justify-center rounded-2xl border border-dashed border-gray-300 bg-white">
              <div className="text-center">
                <ScrollText size={36} className="mx-auto mb-3 text-gray-300" />
                <h3 className="text-base font-bold text-gray-900">请选择项目</h3>
                <p className="mt-1 text-sm text-gray-400">左侧选择项目后，右侧会显示服务启动器和日志入口</p>
              </div>
            </div>
          ) : activeTab === 'service' ? (
            <div className="grid grid-cols-1 xl:grid-cols-2 2xl:grid-cols-3 gap-5">
              <div className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="inline-flex items-center gap-2 rounded-lg bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700">
                      <Play size={13} fill="currentColor" /> 脚本服务启动器
                    </div>
                    <h2 className="mt-4 text-lg font-extrabold text-gray-900">{selectedProject.projectName}</h2>
                    <p className="mt-1 text-sm text-gray-500 line-clamp-2">{selectedProject.description || '当前项目暂无描述'}</p>
                  </div>
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-gray-200 bg-gray-50">
                    {getLanguageIcon(selectedProject.computerLanguage)}
                  </div>
                </div>

                <div className="mt-5 rounded-xl bg-gray-50 px-4 py-3">
                  <div className="grid grid-cols-2 gap-3 text-xs">
                    <div>
                      <div className="font-bold text-gray-400">项目目录</div>
                      <div className="mt-1 truncate font-mono text-gray-700" title={selectedProject.localProjectPath || ''}>
                        {selectedProject.localProjectPath || '未配置'}
                      </div>
                    </div>
                    <div>
                      <div className="font-bold text-gray-400">启动脚本</div>
                      <div className="mt-1 font-mono text-gray-700">{executableServiceRoutes.length} 个</div>
                    </div>
                  </div>
                </div>

                <div className="mt-5 grid grid-cols-2 gap-3">
                  <button
                    onClick={() => openGroupStream('restart')}
                    disabled={executableServiceRoutes.length === 0}
                    className="col-span-2 inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-amber-500 px-4 text-sm font-bold text-white shadow-sm shadow-amber-500/20 transition-colors hover:bg-amber-600 disabled:cursor-not-allowed disabled:bg-gray-200 disabled:text-gray-400 disabled:shadow-none"
                  >
                    <RefreshCw size={15} /> 重启全部
                  </button>
                  <button
                    onClick={loadDockerServices}
                    disabled={dockerLoading}
                    className="col-span-2 inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-gray-200 bg-gray-900 px-4 text-sm font-bold text-white transition-colors hover:bg-gray-800 disabled:cursor-wait disabled:bg-gray-300"
                  >
                    <RefreshCw size={15} className={dockerLoading ? 'animate-spin' : ''} /> 刷新日志入口
                  </button>
                </div>
              </div>

              {serviceRoutes.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-10 text-center">
                  <Terminal size={32} className="mx-auto mb-3 text-gray-300" />
                  <h3 className="text-base font-bold text-gray-900">暂无服务日志路线</h3>
                  <p className="mt-1 text-sm text-gray-400">脚本、make、npm、java、python 等启动方式会归到服务日志</p>
                </div>
              ) : dockerLoading ? (
                <div className="rounded-2xl border border-gray-200 bg-white p-10 text-center">
                  <RefreshCw size={30} className="mx-auto mb-3 animate-spin text-gray-300" />
                  <h3 className="text-base font-bold text-gray-900">正在解析服务日志</h3>
                  <p className="mt-1 text-sm text-gray-400">会从脚本启动路线的目录里读取可跟踪的服务</p>
                </div>
              ) : dockerServices.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-10 text-center">
                  <ScrollText size={32} className="mx-auto mb-3 text-gray-300" />
                  <h3 className="text-base font-bold text-gray-900">暂无服务日志入口</h3>
                  <p className="mt-1 text-sm text-gray-400">服务日志会跟随脚本启动方案展示在这里</p>
                </div>
              ) : (
                dockerServices.map((item, index) => (
                  <div key={`service-${item.routeId}-${item.serviceName || index}`} className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="inline-flex max-w-full items-center gap-1.5 rounded-lg bg-blue-50 px-2.5 py-1 text-xs font-bold text-blue-700">
                          <ScrollText size={12} />
                          <span className="truncate">service log</span>
                        </div>
                        <div className="mt-4 flex min-w-0 items-center gap-2">
                          <ServiceRunningDot running={item.running} />
                          <h3 className="truncate text-lg font-extrabold text-gray-900">
                            {item.serviceName || '服务日志'}
                          </h3>
                        </div>
                        <p className="mt-1 truncate text-sm font-semibold text-gray-500">{item.routeName}</p>
                      </div>
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-gray-50">
                        <ScrollText size={18} className="text-blue-600" />
                      </div>
                    </div>

                    <div className="mt-5 space-y-2 rounded-xl bg-gray-50 px-4 py-3 text-xs">
                      <div>
                        <span className="font-bold text-gray-400">工作目录</span>
                        <p className="mt-1 truncate font-mono text-gray-700" title={item.workDir}>
                          {item.workDir || '未配置'}
                        </p>
                      </div>
                      <div>
                        <span className="font-bold text-gray-400">服务名</span>
                        <p className="mt-1 truncate font-mono text-blue-700">
                          {item.serviceName || '未指定，读取启动路线日志'}
                        </p>
                      </div>
                    </div>

                    <div className="mt-5 grid grid-cols-2 gap-3">
                      <button
                        onClick={() => openServiceRestart(item)}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-amber-500 px-3 text-sm font-bold text-white transition-colors hover:bg-amber-600"
                      >
                        <RefreshCw size={14} /> 重启服务
                      </button>
                      <button
                        onClick={() => openDockerLog(item)}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-blue-600 px-3 text-sm font-bold text-white transition-colors hover:bg-blue-700"
                      >
                        <ScrollText size={14} /> 查看服务日志
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          ) : dockerComposeRoutes.length === 0 ? (
            <div className="flex min-h-[360px] items-center justify-center rounded-2xl border border-dashed border-gray-300 bg-white">
              <div className="text-center">
                <Layers size={36} className="mx-auto mb-3 text-gray-300" />
                <h3 className="text-base font-bold text-gray-900">暂无 Docker 日志入口</h3>
                <p className="mt-1 text-sm text-gray-400">当前项目没有配置 Docker Compose 路线</p>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 xl:grid-cols-2 2xl:grid-cols-3 gap-5">
              <div className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="inline-flex items-center gap-2 rounded-lg bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700">
                      <Layers size={13} /> Docker Compose 启动器
                    </div>
                    <h2 className="mt-4 text-lg font-extrabold text-gray-900">{selectedProject.projectName}</h2>
                    <p className="mt-1 text-sm text-gray-500">docker-compose 启动方案和它对应的日志卡片都在这里</p>
                  </div>
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-gray-200 bg-gray-50">
                    <Layers size={20} className="text-emerald-600" />
                  </div>
                </div>
                <div className="mt-5 rounded-xl bg-gray-50 px-4 py-3 text-xs">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="font-bold text-gray-400">当前目录</div>
                      <div className="mt-1 truncate font-mono text-gray-700" title={selectedProject.localProjectPath || ''}>
                        {selectedProject.localProjectPath || '未配置'}
                      </div>
                    </div>
                    <div>
                      <div className="font-bold text-gray-400">Compose 服务</div>
                      <div className="mt-1 font-mono text-gray-700">{dockerDisplayCount} 个</div>
                    </div>
                  </div>
                </div>
                <div className="mt-5 grid grid-cols-2 gap-3">
                  <button
                    onClick={() => openGroupStream(hasDockerDeployScript ? 'start' : 'restart')}
                    disabled={dockerComposeRoutes.length === 0}
                    className="col-span-2 inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-amber-500 px-4 text-sm font-bold text-white shadow-sm shadow-amber-500/20 transition-colors hover:bg-amber-600 disabled:cursor-not-allowed disabled:bg-gray-200 disabled:text-gray-400 disabled:shadow-none"
                  >
                    {hasDockerDeployScript ? <Play size={15} /> : <RefreshCw size={15} />}
                    {hasDockerDeployScript ? '执行部署脚本' : '重启全部'}
                  </button>
                  <button
                    onClick={loadDockerServices}
                    disabled={dockerLoading}
                    className="col-span-2 inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-gray-200 bg-gray-900 px-4 text-sm font-bold text-white transition-colors hover:bg-gray-800 disabled:cursor-wait disabled:bg-gray-300"
                  >
                    <RefreshCw size={15} className={dockerLoading ? 'animate-spin' : ''} /> 刷新日志入口
                  </button>
                </div>
              </div>

              {dockerLoading ? (
                <div className="rounded-2xl border border-gray-200 bg-white p-10 text-center">
                  <RefreshCw size={30} className="mx-auto mb-3 animate-spin text-gray-300" />
                  <h3 className="text-base font-bold text-gray-900">正在解析 Docker 服务</h3>
                  <p className="mt-1 text-sm text-gray-400">会从本机部署路线目录里读取 compose 服务</p>
                </div>
              ) : dockerServices.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-10 text-center">
                  <Layers size={32} className="mx-auto mb-3 text-gray-300" />
                  <h3 className="text-base font-bold text-gray-900">暂无 Docker 日志入口</h3>
                  <p className="mt-1 text-sm text-gray-400">直接 docker compose / docker-compose 启动的路线会归到 Docker 日志</p>
                </div>
              ) : (
                dockerServices.map((item, index) => (
                  <div key={`${item.routeId}-${item.serviceName || index}`} className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="inline-flex max-w-full items-center gap-1.5 rounded-lg bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700">
                          <Layers size={12} />
                          <span className="truncate">{item.source === 'docker-compose' ? 'compose service' : 'route logs'}</span>
                        </div>
                        <div className="mt-4 flex min-w-0 items-center gap-2">
                          <ServiceRunningDot running={item.running} />
                          <h3 className="truncate text-lg font-extrabold text-gray-900">
                            {item.serviceName || '全部容器日志'}
                          </h3>
                        </div>
                        <p className="mt-1 truncate text-sm font-semibold text-gray-500">{item.routeName}</p>
                      </div>
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-gray-50">
                        <ScrollText size={18} className="text-emerald-600" />
                      </div>
                    </div>

                    <div className="mt-5 space-y-2 rounded-xl bg-gray-50 px-4 py-3 text-xs">
                      <div>
                        <span className="font-bold text-gray-400">工作目录</span>
                        <p className="mt-1 truncate font-mono text-gray-700" title={item.workDir}>
                          {item.workDir || '未配置'}
                        </p>
                      </div>
                      <div>
                        <span className="font-bold text-gray-400">服务名</span>
                        <p className="mt-1 truncate font-mono text-emerald-700">
                          {item.serviceName || '未指定，读取路线全部日志'}
                        </p>
                      </div>
                    </div>

                    <div className="mt-5 grid grid-cols-2 gap-3">
                      <button
                        onClick={() => openDockerRestart(item)}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-amber-500 px-3 text-sm font-bold text-white transition-colors hover:bg-amber-600"
                      >
                        <RefreshCw size={14} /> {item.serviceName ? '重启服务' : '重启路线'}
                      </button>
                      <button
                        onClick={() => openDockerLog(item)}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-emerald-600 px-3 text-sm font-bold text-white transition-colors hover:bg-emerald-700"
                      >
                        <ScrollText size={14} /> 查看实时日志
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      </section>

      {logPanel.open && (
        <DeployLogPanel
          projectId={logPanel.projectId}
          projectName={logPanel.projectName}
          envKey={logPanel.envKey}
          routeName={logPanel.routeName}
          mode={logPanel.mode}
          streamPath={logPanel.streamPath}
          panelTitle={logPanel.panelTitle}
          introText={logPanel.introText}
          onClose={() => setLogPanel(previous => ({ ...previous, open: false }))}
        />
      )}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
