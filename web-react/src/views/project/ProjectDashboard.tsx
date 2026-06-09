import React, { useEffect, useRef, useState } from 'react';
import { getProjectPage, deleteProject, saveOrUpdateProject } from '../../api/project';
import { saveOrUpdateRoute, deleteRoute } from '../../api/project_route';
import { getGroupList, saveOrUpdateGroup, deleteGroup } from '../../api/project_group';
import { getServerList } from '../../api/server';
import toast from 'react-hot-toast';
import { Play, Settings2, Plus, Search, TerminalSquare, Github, Box, X, Trash2, MonitorPlay, Server, Terminal, Code, Pencil, Check, Folder, Square, ScrollText } from 'lucide-react';
import DeployLogPanel from '../../components/DeployLogPanel';
import ProjectScripts from './ProjectScripts';
import {
  DEPENDENCY_INCREMENTAL_ROUTE_COLOR,
  LOCAL_FULL_ROUTE_COLOR,
  LOCAL_INCREMENTAL_ROUTE_COLOR,
  REMOTE_ROUTE_COLOR,
  getDisplayStopCommand,
  normalizeRouteColor,
  shouldExposeStopButton
} from './ProjectRoutePresentation';

export default function ProjectDashboard() {
  // ── Data ─────────────────────────────────────────────────────────────────
  const [projects, setProjects] = useState<any[]>([]);
  const [groups, setGroups] = useState<any[]>([]);
  const [serverOptions, setServerOptions] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  // ── Filters ───────────────────────────────────────────────────────────────
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null); // null = 全部
  const [searchQuery, setSearchQuery] = useState('');

  // ── Sidebar inline CRUD state ─────────────────────────────────────────────
  const [editingGroupId, setEditingGroupId] = useState<number | null>(null);
  const [editingGroupName, setEditingGroupName] = useState('');
  const editInputRef = useRef<HTMLInputElement>(null);

  // ── Drawer / Form states ──────────────────────────────────────────────────
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [commandDrawerOpen, setCommandDrawerOpen] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);

  const [confirmModal, setConfirmModal] = useState({
    isOpen: false, title: '', message: '', actionText: '', actionColor: '',
    onConfirm: () => { }
  });

  // ── Deploy Log Panel state ───────────────────────────────────────────────
  const [deployLog, setDeployLog] = useState<{
    open: boolean; projectId: number; projectName: string; envKey: string; routeName: string; mode: 'deploy' | 'stop' | 'logs';
  }>({ open: false, projectId: 0, projectName: '', envKey: '', routeName: '', mode: 'deploy' });
  const [scriptDialog, setScriptDialog] = useState<{ project: any; route: any } | null>(null);

  const [formData, setFormData] = useState({
    ID: 0, projectName: '', computerLanguage: 'vue', groupId: 0, groupName: '',
    description: '', accessUrl: '', localProjectPath: '', userId: 0
  });

  const [routeFormData, setRouteFormData] = useState({
    ID: 0, projectId: 0, routeKey: '', routeName: '', serverId: '',
    localProjectPath: '', serverProjectPath: '', localExecuteCommand: '',
    localStopCommand: '', localStartCommand: '', serverExecuteCommand: '', color: '', icon: '',
    buildType: '', fileName: ''
  });

  // ── Fetch ─────────────────────────────────────────────────────────────────
  const fetchData = async () => {
    setLoading(true);
    try {
      const [projRes, groupRes, serverRes] = await Promise.all([
        getProjectPage({ page: 1, pageSize: 200 }),
        getGroupList({}),
        getServerList({}),
      ]) as any[];
      if (projRes.code === 0) setProjects(projRes.data?.list || []);
      if (groupRes.code === 0) setGroups(groupRes.data || []);
      if (serverRes.code === 0) setServerOptions(serverRes.data || []);
    } catch {
      toast.error('数据拉取异常');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const handleGlobalTrigger = () => openDrawer();
    window.addEventListener('OPEN_PROJECT_DRAWER', handleGlobalTrigger);
    return () => {
      window.removeEventListener('OPEN_PROJECT_DRAWER', handleGlobalTrigger);
    };
  }, []);

  useEffect(() => {
    if (editingGroupId && editInputRef.current) editInputRef.current.focus();
  }, [editingGroupId]);

  // ── Sidebar Group CRUD ────────────────────────────────────────────────────


  const handleRenameGroup = async (id: number) => {
    const name = editingGroupName.trim();
    if (!name) return;
    try {
      const res: any = await saveOrUpdateGroup({ ID: id, groupName: name });
      if (res.code === 0) {
        toast.success('已重命名');
        setEditingGroupId(null);
        fetchData();
      } else toast.error(res.msg || '重命名失败');
    } catch { toast.error('操作异常'); }
  };

  const handleDeleteGroup = (id: number, name: string) => {
    setConfirmModal({
      isOpen: true, title: '删除项目组',
      message: `确定删除「${name}」项目组吗？\n该组下不能有任何项目，否则将删除失败。`,
      actionText: '删除', actionColor: 'bg-red-500 hover:bg-red-600',
      onConfirm: async () => {
        setConfirmModal(p => ({ ...p, isOpen: false }));
        try {
          const res: any = await deleteGroup(id);
          if (res.code === 0) { toast.success('已删除'); fetchData(); }
          else toast.error(res.msg || '删除失败');
        } catch { toast.error('删除异常'); }
      }
    });
  };

  // ── Filtered projects ─────────────────────────────────────────────────────
  const filteredProjects = projects.filter(p => {
    const groupOk = selectedGroupId === null || p.groupId === selectedGroupId;
    const searchLow = searchQuery.toLowerCase();
    const searchOk = !searchQuery || (p.projectName?.toLowerCase() || '').includes(searchLow) || (p.description?.toLowerCase() || '').includes(searchLow);
    return groupOk && searchOk;
  });

  // ── Drawer helpers ────────────────────────────────────────────────────────
  const openDrawer = (project?: any) => {
    if (project) {
      setEditingId(project.ID);
      const matchedGroup = groups.find(g => g.ID === project.groupId);
      setFormData({
        ID: project.ID, projectName: project.projectName || '',
        computerLanguage: project.computerLanguage || 'vue',
        groupId: project.groupId || 0,
        groupName: matchedGroup ? matchedGroup.groupName : '',
        description: project.description || '', accessUrl: project.accessUrl || '',
        localProjectPath: project.localProjectPath || '', userId: project.userId || 0
      });
    } else {
      setEditingId(null);
      const matchedGroup = groups.find(g => g.ID === selectedGroupId);
      setFormData({
        ID: 0, projectName: '', computerLanguage: 'vue',
        groupId: selectedGroupId || 0,
        groupName: matchedGroup ? matchedGroup.groupName : '',
        description: '', accessUrl: '', localProjectPath: '', userId: 0
      });
    }
    setDrawerOpen(true);
  };

  const openCommandDrawer = (project: any, route?: any) => {
    setEditingId(project.ID);
    const matchedGroup = groups.find(g => g.ID === project.groupId);
    setFormData({
      ID: project.ID, projectName: project.projectName || '',
      computerLanguage: project.computerLanguage || 'vue',
      groupId: project.groupId || 0,
      groupName: matchedGroup ? matchedGroup.groupName : '',
      description: project.description || '', accessUrl: project.accessUrl || '',
      localProjectPath: project.localProjectPath || '', userId: project.userId || 0
    });
    if (route) {
      setRouteFormData({
        ID: route.ID, projectId: project.ID, routeKey: route.routeKey || '',
        routeName: route.routeName || '',
        serverId: route.serverId ? String(route.serverId) : '',
        localProjectPath: route.localProjectPath || '',
        serverProjectPath: route.serverProjectPath || '',
        localExecuteCommand: route.localExecuteCommand || '',
        localStopCommand: getDisplayStopCommand(route),
        localStartCommand: route.localStartCommand || '',
        serverExecuteCommand: route.serverExecuteCommand || (route.serverId ? 'source start.sh' : ''),
        color: normalizeRouteColor(route.color, route),
        icon: route.icon || 'Play', buildType: route.buildType || 'build_image',
        fileName: route.fileName || ''
      });
    } else {
      const isPythonProject = project.computerLanguage?.toLowerCase() === 'python';
      setRouteFormData({
        ID: 0, projectId: project.ID, routeKey: 'local', routeName: isPythonProject ? '构建项目镜像' : '本地部署',
        serverId: '', localProjectPath: '', serverProjectPath: '',
        localExecuteCommand: isPythonProject ? 'make build-project' : 'docker compose up --build -d',
        localStopCommand: '',
        localStartCommand: '', serverExecuteCommand: '',
        color: LOCAL_FULL_ROUTE_COLOR, icon: 'Play',
        buildType: 'build_image', fileName: ''
      });
    }
    setCommandDrawerOpen(true);
  };

  type RouteDeployMode = 'local_full' | 'local_incremental' | 'remote_incremental' | 'dependency_incremental';

  const getRouteDeployMode = (route: any): RouteDeployMode => {
    const routeKey = String(route?.routeKey || '').toLowerCase();
    const routeName = String(route?.routeName || '');
    const buildType = String(route?.buildType || '').toLowerCase();
    if (route?.serverId || routeKey.includes('remote') || routeName.includes('远程')) return 'remote_incremental';
    if (routeName.includes('依赖增量')) return 'dependency_incremental';
    if (routeKey.includes('incremental') || routeName.includes('增量') || buildType === 'docker_compose_deploy') return 'local_incremental';
    return 'local_full';
  };

  const getRouteModeDefaults = (mode: RouteDeployMode, language: string) => {
    const lang = String(language || '').toLowerCase();
    const isPython = lang === 'python';
    const isJava = lang === 'java';
    const firstServerId = serverOptions[0]?.ID ? String(serverOptions[0].ID) : '';

    const defaults: Record<RouteDeployMode, any> = {
      local_full: {
        routeKey: 'local_full',
        routeName: isPython ? '构建项目镜像' : '本地全量部署',
        serverId: '',
        localExecuteCommand: isPython ? 'make build-project' : 'docker compose up --build -d',
        localStopCommand: isPython ? '' : 'docker compose down',
        serverExecuteCommand: '',
        buildType: 'build_image',
        color: LOCAL_FULL_ROUTE_COLOR,
        icon: 'Play',
      },
      local_incremental: {
        routeKey: isJava ? 'local_incremental_jar' : 'local_incremental',
        routeName: '本地增量部署',
        serverId: '',
        localExecuteCommand: isJava ? 'bash start.sh incremental' : 'make deploy-incremental',
        localStopCommand: isPython ? '' : isJava ? 'docker compose down' : 'make stop',
        serverExecuteCommand: '',
        buildType: 'build_incremental_image',
        color: LOCAL_INCREMENTAL_ROUTE_COLOR,
        icon: 'MonitorPlay',
      },
      remote_incremental: {
        routeKey: 'remote_incremental',
        routeName: '远程增量部署',
        serverId: firstServerId,
        localExecuteCommand: isPython ? 'make package-remote-incremental' : 'make package-remote-incremental',
        localStopCommand: '',
        serverExecuteCommand: isPython ? 'bash remote-incremental.sh <image.tar> <image> <container>' : 'source start.sh',
        buildType: 'build_incremental_image',
        color: REMOTE_ROUTE_COLOR,
        icon: 'MonitorPlay',
      },
      dependency_incremental: {
        routeKey: 'local_incremental',
        routeName: '构建依赖增量镜像',
        serverId: '',
        localExecuteCommand: isPython ? 'make build-deps' : 'make build-deps',
        localStopCommand: '',
        serverExecuteCommand: '',
        buildType: 'build_incremental_image',
        color: DEPENDENCY_INCREMENTAL_ROUTE_COLOR,
        icon: 'Play',
      },
    };

    return defaults[mode];
  };

  const handleRouteDeployModeChange = (mode: RouteDeployMode) => {
    const defaults = getRouteModeDefaults(mode, formData.computerLanguage);
    setRouteFormData(prev => ({
      ...prev,
      ...defaults,
      serverProjectPath: mode === 'remote_incremental' ? prev.serverProjectPath : '',
      fileName: mode === 'remote_incremental' ? prev.fileName : '',
    }));
  };

  // ── Deploy ────────────────────────────────────────────────────────────────
  const handleDeployClick = (project: any, e: React.MouseEvent, envKey: string, routeName: string) => {
    e.stopPropagation();
    setConfirmModal({
      isOpen: true, title: '执行调度',
      message: `确定要针对项目 [${project.projectName}] 调度执行【${routeName}】操作吗？\n请确保相关环境参数及代码已就位。`,
      actionText: '执行任务', actionColor: 'bg-green-600 hover:bg-green-700',
      onConfirm: () => executeDeploy(project.ID, envKey, project.projectName)
    });
  };

  const executeDeploy = async (projectId: number, envKey: string, projectName: string) => {
    setConfirmModal(p => ({ ...p, isOpen: false }));
    // 找到路由名称
    const project = projects.find(p => p.ID === projectId);
    const route = project?.routes?.find((r: any) => String(r.ID) === envKey || r.routeKey === envKey);
    const routeDisplayName = route?.routeName || envKey;
    // 打开日志面板，使用 SSE 流式推送
    setDeployLog({ open: true, projectId, projectName, envKey, routeName: routeDisplayName, mode: 'deploy' });
  };

  const handleStopClick = (project: any, e: React.MouseEvent, envKey: string, routeName: string) => {
    e.stopPropagation();
    setConfirmModal({
      isOpen: true, title: '执行关闭',
      message: `确定要关闭项目 [${project.projectName}] 的【${routeName}】环境吗？\n将执行 docker compose down 关闭服务。`,
      actionText: '执行关闭', actionColor: 'bg-red-500 hover:bg-red-600',
      onConfirm: () => executeStop(project.ID, envKey, project.projectName)
    });
  };

  const executeStop = (projectId: number, envKey: string, projectName: string) => {
    setConfirmModal(p => ({ ...p, isOpen: false }));
    const project = projects.find(p => p.ID === projectId);
    const route = project?.routes?.find((r: any) => String(r.ID) === envKey || r.routeKey === envKey);
    const routeDisplayName = route?.routeName || envKey;
    setDeployLog({ open: true, projectId, projectName, envKey, routeName: routeDisplayName, mode: 'stop' });
  };

  const handleDockerLogClick = (project: any, e: React.MouseEvent, envKey: string, routeName: string) => {
    e.stopPropagation();
    setDeployLog({ open: true, projectId: project.ID, projectName: project.projectName, envKey, routeName, mode: 'logs' });
  };

  // ── Project CRUD ──────────────────────────────────────────────────────────
  const handleDeleteClick = (id: number) => {
    setConfirmModal({
      isOpen: true, title: '永久删除项目',
      message: '删除项目完全不可逆，并会彻底级联擦除所有相关的脚本、服务器配置与部署环境记录。\n请确认是否销毁？',
      actionText: '彻底删除', actionColor: 'bg-red-500 hover:bg-red-600',
      onConfirm: () => executeDelete(id)
    });
  };

  const executeDelete = async (id: number) => {
    setConfirmModal(p => ({ ...p, isOpen: false }));
    try {
      const res: any = await deleteProject(id);
      if (res.code === 0) { toast.success('项目及其附件已销毁'); setDrawerOpen(false); fetchData(); }
      else toast.error(res.msg || '删除失败');
    } catch { toast.error('删除异常'); }
  };

  const handleFormChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(p => ({ ...p, [name]: value }));
  };

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.projectName) { toast.error('请填写项目名称'); return; }
    if (!formData.groupName) { toast.error('请填写或选择所属项目组'); return; }
    
    setSubmitLoading(true);
    try {
      let finalGroupId = formData.groupId;
      const existingGroup = groups.find(g => g.groupName === formData.groupName.trim());
      
      if (existingGroup) {
         finalGroupId = existingGroup.ID;
      } else {
         const groupRes: any = await saveOrUpdateGroup({ groupName: formData.groupName.trim() });
         if (groupRes.code === 0 && groupRes.data?.ID) {
            finalGroupId = groupRes.data.ID;
         } else {
            toast.error(groupRes.msg || '创建项目组失败');
            setSubmitLoading(false);
            return;
         }
      }

      const payload = { ...formData, groupId: finalGroupId };
      const res: any = await saveOrUpdateProject(payload);
      if (res.code === 0) {
        toast.success(editingId ? '项目已更新' : '项目已落库创建');
        setDrawerOpen(false); setCommandDrawerOpen(false); fetchData();
      } else toast.error(res.msg || '操作失败');
    } catch { toast.error('提交异常'); }
    finally { setSubmitLoading(false); }
  };

  const handleRouteFormChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const target = e.target as HTMLInputElement;
    const value = target.type === 'checkbox' ? target.checked : target.value;
    const name = target.name;
    setRouteFormData(p => {
      const next = { ...p, [name]: value } as any;
      return next;
    });
  };

  const handleRouteSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!routeFormData.routeName) { toast.error('部署路线名称为必填'); return; }
    setSubmitLoading(true);
    try {
      const payload = { ...routeFormData, serverId: routeFormData.serverId ? parseInt(routeFormData.serverId, 10) : 0 };
      const res: any = await saveOrUpdateRoute(payload);
      if (res.code === 0) {
        toast.success(routeFormData.ID ? '部署路线已更新' : '新增部署路线成功');
        setCommandDrawerOpen(false); fetchData();
      } else toast.error(res.msg || '路线保存失败');
    } catch { toast.error('提交异常'); }
    finally { setSubmitLoading(false); }
  };

  const handleRouteDeleteClick = (id: number) => {
    setConfirmModal({
      isOpen: true, title: '移除环境路线',
      message: '此环境部署路线将会从该项目中彻底移除，关联的环境专属脚本也会一起删除，触发参数将失效。\n确定继续删除？',
      actionText: '删除路线', actionColor: 'bg-red-500 hover:bg-red-600',
      onConfirm: () => executeRouteDelete(id)
    });
  };

  const executeRouteDelete = async (id: number) => {
    setConfirmModal(p => ({ ...p, isOpen: false }));
    try {
      const res: any = await deleteRoute({ ID: id });
      if (res.code === 0) { toast.success('路线配置已剥离'); setCommandDrawerOpen(false); fetchData(); }
      else toast.error(res.msg || '删除失败');
    } catch { toast.error('删除异常'); }
  };

  const getLanguageIcon = (lang: string) => {
    const lg = (lang || '').toLowerCase();
    if (lg.includes('vue')) return <Box size={20} className="text-[#42b883]" />;
    if (lg.includes('react')) return <Box size={20} className="text-[#61dafb]" />;
    if (lg.includes('python')) return <TerminalSquare size={20} className="text-[#3776AB]" />;
    if (lg.includes('go')) return <TerminalSquare size={20} className="text-[#00ADD8]" />;
    if (lg.includes('java')) return <Server size={20} className="text-[#f89820]" />;
    return <Github size={20} className="text-gray-800" />;
  };

  const currentGroupName = selectedGroupId === null
    ? '全部项目'
    : groups.find(g => g.ID === selectedGroupId)?.groupName || '未知组';

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div className="w-full flex gap-0 relative">

      {/* ── Sidebar ── */}
      <aside className="w-56 shrink-0 bg-white border-r border-gray-100 min-h-[calc(100vh-64px)] flex flex-col">
        {/* 侧边栏顶部 */}
        <div className="px-3 pt-4 pb-2">
          <div className="relative mb-3">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="搜索项目..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full bg-gray-50 border border-gray-200 rounded-md py-1.5 pl-8 pr-2 text-xs focus:outline-none focus:ring-2 focus:ring-black/5 focus:border-gray-300"
            />
          </div>
          <div className="flex items-center justify-between mb-2 px-1">
            <span className="text-[10px] font-bold text-gray-400 uppercase tracking-wider">项目组</span>
            <button
              onClick={() => openDrawer()}
              className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors flex items-center justify-center"
              title="新建配置"
            >
              <Plus size={14} strokeWidth={2.5} />
            </button>
          </div>
        </div>

        {/* 全部 + 子分组 */}
        <div className="px-3 flex flex-col gap-0.5">
          <button
            onClick={() => setSelectedGroupId(null)}
            className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${selectedGroupId === null ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'}`}
          >
            <Folder size={14} className={selectedGroupId === null ? 'text-white' : 'text-gray-400'} />
            <span className="truncate">全部项目</span>
            <span className={`ml-auto text-xs px-1.5 py-0.5 rounded-full font-mono ${selectedGroupId === null ? 'bg-white/20 text-white' : 'bg-gray-100 text-gray-500'}`}>
              {projects.length}
            </span>
          </button>

          {/* 各项目组（子级缩进） */}
          {groups.length > 0 && (
            <div className="ml-4 pl-3 border-l-2 border-gray-200 flex flex-col gap-0.5">
              {groups.map(g => {
                const count = projects.filter(p => p.groupId === g.ID).length;
                const active = selectedGroupId === g.ID;
                return (
                  <div key={g.ID} className="group/item relative">
                    {editingGroupId === g.ID ? (
                      <div className="flex items-center gap-1 px-1 py-1">
                        <input
                          ref={editInputRef}
                          value={editingGroupName}
                          onChange={e => setEditingGroupName(e.target.value)}
                          onKeyDown={e => { if (e.key === 'Enter') handleRenameGroup(g.ID); if (e.key === 'Escape') setEditingGroupId(null); }}
                          className="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-md px-2 py-1.5 outline-none focus:ring-2 focus:ring-black/10"
                        />
                        <button onClick={() => handleRenameGroup(g.ID)} className="p-1 rounded bg-black text-white"><Check size={10} /></button>
                        <button onClick={() => setEditingGroupId(null)} className="p-1 rounded text-gray-400 hover:bg-gray-100"><X size={10} /></button>
                      </div>
                    ) : (
                      <button
                        onClick={() => setSelectedGroupId(g.ID)}
                        className={`w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${active ? 'bg-gray-900 text-white' : 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'}`}
                      >
                        <Folder size={12} className={active ? 'text-white' : 'text-gray-400'} />
                        <span className="truncate flex-1 text-left">{g.groupName}</span>
                        <span className={`text-xs px-1.5 py-0.5 rounded-full font-mono ${active ? 'bg-white/20 text-white' : 'bg-gray-100 text-gray-500'}`}>{count}</span>
                        {/* 悬停编辑/删除按钮 */}
                        {!active && (
                          <span className="hidden group-hover/item:flex items-center gap-0.5 absolute right-2">
                            <span
                              onClick={e => { e.stopPropagation(); setEditingGroupId(g.ID); setEditingGroupName(g.groupName); }}
                              className="p-1 rounded text-gray-400 hover:text-gray-700 hover:bg-gray-100 cursor-pointer"
                            ><Pencil size={11} /></span>
                            <span
                              onClick={e => { e.stopPropagation(); handleDeleteGroup(g.ID, g.groupName); }}
                              className="p-1 rounded text-gray-400 hover:text-red-600 hover:bg-red-50 cursor-pointer"
                            ><Trash2 size={11} /></span>
                          </span>
                        )}
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </aside>
      {/* ── Main Content ── */}
      <div className="flex-1 min-w-0 px-6 py-6 border-l border-transparent">
        {loading ? (
          <div className="h-40 flex items-center justify-center text-gray-400">资源加载中...</div>
        ) : filteredProjects.length === 0 ? (
          <div className="border border-dashed border-gray-300 rounded-xl p-12 text-center bg-gray-50 mt-2">
            <Folder size={32} className="text-gray-300 mx-auto mb-3" />
            <h3 className="text-base font-medium text-gray-900 mb-1">
              {selectedGroupId ? '该项目组暂无服务' : '还没有任何项目'}
            </h3>
            <p className="text-sm text-gray-400 mb-5">
              前往左侧边栏顶部的「+」新建你的配置
            </p>
            <button onClick={() => openDrawer()} className="bg-black hover:bg-gray-800 text-white font-medium py-2 px-5 rounded-lg text-sm transition-colors">
              新建服务
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
            {filteredProjects.map((project) => (
              <div key={project.ID} className="group bg-white rounded-xl shadow-sm border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200 overflow-hidden flex flex-col">
                <div className="p-5 cursor-pointer" onClick={() => openDrawer(project)}>
                  <div className="flex justify-between items-start mb-3">
                    <div className="w-9 h-9 rounded-full bg-gray-100 flex items-center justify-center border border-gray-200">
                      {getLanguageIcon(project.computerLanguage)}
                    </div>
                    <div className="flex items-center gap-1.5 text-xs font-mono bg-gray-100 text-gray-600 px-2 py-1 rounded-md">
                      <Code size={11} /> {project.computerLanguage}
                    </div>
                  </div>
                  <h3 className="text-base font-bold text-gray-900 truncate mb-1">{project.projectName}</h3>
                  <p className="text-xs text-gray-400 line-clamp-2 min-h-[2.2rem]">
                    {project.description || <span className="italic">暂无描述</span>}
                  </p>
                  {project.accessUrl && (
                    <a
                      href={project.accessUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={e => e.stopPropagation()}
                      className="inline-flex items-center gap-1 text-xs text-blue-500 hover:text-blue-700 hover:underline mt-2 truncate max-w-full"
                      title={project.accessUrl}
                    >
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                      {project.accessUrl.replace(/^https?:\/\//, '')}
                    </a>
                  )}
                </div>
                <div className="mt-auto border-t border-gray-100 bg-gray-50/50 p-4">
                  <div className="flex flex-col gap-2">
                    <span className="text-[10px] font-bold text-gray-400 uppercase tracking-wider mb-0.5">部署路线</span>
                    {(project.routes || []).map((route: any) => {
                      const hasStopCmd = shouldExposeStopButton(route);
                      return (
                      <div key={route.ID} className="flex items-center gap-1.5">
                        <button
                          onClick={(e) => { e.stopPropagation(); handleDeployClick(project, e, String(route.ID), route.routeName); }}
                          className={`${hasStopCmd ? 'flex-[2]' : 'flex-1'} min-w-0 flex items-center justify-between px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${normalizeRouteColor(route.color, route)}`}
                        >
                          <div className="min-w-0 flex items-center gap-1.5">
                            {route.icon === 'MonitorPlay' ? <MonitorPlay size={12} /> : <Play size={12} fill="currentColor" />}
                            <span className="truncate">{route.routeName}</span>
                          </div>
                          <Play size={10} className="shrink-0 opacity-30" />
                        </button>
                        {hasStopCmd && (
                          <button
                            onClick={(e) => { e.stopPropagation(); handleStopClick(project, e, String(route.ID), route.routeName); }}
                            className="flex-[1] flex items-center justify-center gap-1 px-2 py-1.5 rounded-md text-xs font-medium bg-red-500 text-white hover:bg-red-600 transition-colors"
                          >
                            <Square size={10} fill="currentColor" />
                            <span>关闭</span>
                          </button>
                        )}
                        <button
                          onClick={(e) => { e.stopPropagation(); setScriptDialog({ project, route }); }}
                          className="flex items-center justify-center p-1.5 rounded-md text-xs border border-blue-100 bg-blue-50 text-blue-600 hover:bg-blue-100 transition-colors"
                          title="管理脚本"
                        ><Code size={13} /></button>
                        <button
                          onClick={(e) => handleDockerLogClick(project, e, String(route.ID), route.routeName)}
                          className="flex items-center justify-center p-1.5 rounded-md text-xs border border-emerald-100 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 transition-colors"
                          title="查看 Docker 实时日志"
                        ><ScrollText size={13} /></button>
                        <button
                          onClick={(e) => { e.stopPropagation(); openCommandDrawer(project, route); }}
                          className="flex items-center justify-center p-1.5 rounded-md text-xs border border-gray-200 bg-white text-gray-700 hover:bg-gray-100 transition-colors"
                          title="路线配置"
                        ><Settings2 size={13} /></button>
                      </div>
                      );
                    })}
                    <button
                      onClick={(e) => { e.stopPropagation(); openCommandDrawer(project, null); }}
                      className="mt-0.5 w-full flex items-center justify-center gap-1.5 py-1.5 border border-dashed border-gray-300 rounded-md text-xs text-gray-400 hover:text-gray-700 hover:bg-gray-50 hover:border-gray-400 transition-colors"
                    >
                      <Plus size={11} /> 添加路线
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* ── Project Drawer ── */}
      {drawerOpen && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-black/20 backdrop-blur-sm" onClick={() => setDrawerOpen(false)} />
          <div className="relative w-full max-w-lg bg-white h-full shadow-2xl flex flex-col animate-in slide-in-from-right-8 duration-300">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <h2 className="text-base font-bold text-gray-900">{editingId ? '编辑服务配置' : '创建新服务'}</h2>
              <button onClick={() => setDrawerOpen(false)} className="p-2 text-gray-400 hover:text-gray-900 rounded-full hover:bg-gray-100 transition-colors"><X size={18} /></button>
            </div>
            <form onSubmit={handleFormSubmit} className="flex-1 overflow-y-auto px-6 py-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">服务名称 <span className="text-red-500">*</span></label>
                <input type="text" name="projectName" required value={formData.projectName} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5" placeholder="例如: web-frontend" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">所属项目组 <span className="text-red-500">*</span></label>
                  <input 
                    type="text" 
                    name="groupName" 
                    required 
                    value={formData.groupName} 
                    onChange={handleFormChange} 
                    list="group-options"
                    className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5" 
                    placeholder="选择或输入新组名" 
                    autoComplete="off"
                  />
                  <datalist id="group-options">
                    {groups.map(g => <option key={g.ID} value={g.groupName} />)}
                  </datalist>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">框架语言</label>
                  <select name="computerLanguage" value={formData.computerLanguage} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 bg-white">
                    <option value="vue">Vue</option>
                    <option value="react">React</option>
                    <option value="python">Python</option>
                    <option value="java">Java</option>
                    <option value="go">Go</option>
                    <option value="docker-compose">前后端docker-compose</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">本地项目绝对路径</label>
                <input type="text" name="localProjectPath" value={formData.localProjectPath} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="/Users/workspace/..." />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">服务描述</label>
                <textarea name="description" value={formData.description} onChange={handleFormChange} rows={3} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 resize-none" placeholder="简要介绍该服务的用途、技术栈或备注..." />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">访问路径</label>
                <input type="text" name="accessUrl" value={formData.accessUrl} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="http://example.com:8080" />
                <p className="text-xs text-gray-400 mt-1">项目的外部访问地址，会显示在卡片上可直接点击跳转</p>
              </div>
            </form>
            <div className="border-t border-gray-100 p-5 flex items-center justify-between bg-gray-50">
              {editingId ? (
                <button type="button" onClick={() => handleDeleteClick(editingId)} className="text-red-500 hover:bg-red-50 p-2 rounded-lg transition-colors flex items-center gap-1 text-sm font-medium">
                  <Trash2 size={15} /> 删除
                </button>
              ) : <div />}
              <div className="flex gap-3">
                <button type="button" onClick={() => setDrawerOpen(false)} className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-100 transition-colors">取消</button>
                <button type="submit" onClick={handleFormSubmit} disabled={submitLoading} className="px-4 py-2 bg-black text-white rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition-colors shadow-sm">
                  {submitLoading ? '保存中...' : '保存配置'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Command / Route Drawer ── */}
      {commandDrawerOpen && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-black/20 backdrop-blur-sm" onClick={() => setCommandDrawerOpen(false)} />
          <div className="relative w-full max-w-2xl bg-gray-50 h-full shadow-2xl flex flex-col animate-in slide-in-from-right-8 duration-300 border-l border-gray-200">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 bg-white">
              <div className="flex items-center gap-2">
                <div className="p-1.5 bg-black text-white rounded-md"><Terminal size={16} /></div>
                <div>
                  <h2 className="text-base font-bold text-gray-900 leading-tight">指令与部署路线</h2>
                  <p className="text-xs text-gray-500">配置 [{formData.projectName}] 的预发布参数</p>
                </div>
              </div>
              <button onClick={() => setCommandDrawerOpen(false)} className="p-2 text-gray-400 hover:text-gray-900 rounded-full hover:bg-gray-100 transition-colors"><X size={18} /></button>
            </div>
            <form className="flex-1 overflow-y-auto p-6 space-y-5">
              {/* Read-only project info */}
              <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm space-y-3">
                <h3 className="text-xs font-bold text-gray-600 flex items-center gap-1.5 uppercase tracking-wide"><Box size={13} /> 项目全局配置 (只读)</h3>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">服务名称</label>
                  <input type="text" disabled value={formData.projectName} className="w-full border border-gray-200 bg-gray-50 text-gray-500 rounded-lg p-2.5 text-sm cursor-not-allowed" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">框架语言</label>
                    <input type="text" disabled value={formData.computerLanguage} className="w-full border border-gray-200 bg-gray-50 text-gray-500 rounded-lg p-2.5 text-sm capitalize cursor-not-allowed" />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">本地项目绝对路径</label>
                  <input type="text" disabled value={formData.localProjectPath} className="w-full border border-gray-200 bg-gray-50 text-gray-500 font-mono rounded-lg p-2.5 text-sm cursor-not-allowed" />
                </div>
              </div>

              {/* Route config */}
              <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm space-y-4">
                <h3 className="text-xs font-bold text-gray-600 flex items-center gap-1.5 uppercase tracking-wide border-b border-gray-100 pb-2"><Settings2 size={13} /> 环境参数配置</h3>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">部署方式</label>
                  <div className="grid grid-cols-2 gap-2">
                    {([
                      { key: 'local_full', title: '本地全量', desc: '构建并启动完整服务', color: LOCAL_FULL_ROUTE_COLOR },
                      { key: 'local_incremental', title: '本地增量', desc: '复用环境增量发布', color: LOCAL_INCREMENTAL_ROUTE_COLOR },
                      { key: 'remote_incremental', title: '远程增量', desc: '打包后推送服务器', color: REMOTE_ROUTE_COLOR },
                      { key: 'dependency_incremental', title: '依赖增量', desc: '构建依赖基础镜像', color: DEPENDENCY_INCREMENTAL_ROUTE_COLOR },
                    ] as const).map(option => {
                      const active = getRouteDeployMode(routeFormData) === option.key;
                      return (
                        <button
                          key={option.key}
                          type="button"
                          onClick={() => handleRouteDeployModeChange(option.key)}
                          className={`text-left rounded-lg border p-3 transition-colors ${
                            active ? 'border-gray-900 bg-gray-50 shadow-sm' : 'border-gray-200 bg-white hover:bg-gray-50'
                          }`}
                        >
                          <span className={`mb-2 inline-flex items-center rounded-md px-2 py-1 text-xs font-bold ${option.color}`}>
                            {option.title}
                          </span>
                          <span className="block text-xs text-gray-500">{option.desc}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">环境名称 <span className="text-red-500">*</span></label>
                    <input type="text" name="routeName" required value={routeFormData.routeName} onChange={handleRouteFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5" placeholder="如: 本地部署" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">目标服务器</label>
                    <select name="serverId" value={routeFormData.serverId} onChange={handleRouteFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 bg-white">
                      <option value="">本机服务器</option>
                      {serverOptions.map(s => <option key={s.ID} value={s.ID}>{s.serverName} ({s.serverIp})</option>)}
                    </select>
                  </div>
                </div>

                {routeFormData.serverId === '' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">本机执行命令</label>
                      <input
                        type="text"
                        name="localExecuteCommand"
                        value={routeFormData.localExecuteCommand}
                        onChange={handleRouteFormChange}
                        className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono text-green-700"
                        placeholder="docker compose up --build -d"
                      />
                      <p className="text-xs text-gray-400 mt-1">部署时在本机执行的启动指令</p>
                    </div>
                    {shouldExposeStopButton(routeFormData) && (
                      <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">关闭命令</label>
                      <input
                        type="text"
                        name="localStopCommand"
                        value={routeFormData.localStopCommand}
                        onChange={handleRouteFormChange}
                        className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono text-red-600"
                        placeholder="docker compose down"
                      />
                      <p className="text-xs text-gray-400 mt-1">关闭服务时执行的指令，配置后卡片上会出现「关闭」按钮</p>
                      </div>
                    )}
                  </>
                )}

                {routeFormData.serverId !== '' && (
                  <div className="space-y-4">
                    <div className="flex gap-4">
                      <div className="flex-1">
                        <label className="block text-sm font-medium text-gray-700 mb-1">压缩文件名</label>
                        <input type="text" name="fileName" value={routeFormData.fileName} onChange={handleRouteFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono text-gray-600" placeholder="例如: dist.zip" />
                      </div>
                      <div className="flex-1">
                        <label className="block text-sm font-medium text-gray-700 mb-1">服务器执行指令</label>
                        <input type="text" name="serverExecuteCommand" value={routeFormData.serverExecuteCommand} onChange={handleRouteFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono text-blue-600" placeholder="例如: source start.sh" />
                      </div>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">远程服务器绝对路径</label>
                      <input type="text" name="serverProjectPath" value={routeFormData.serverProjectPath} onChange={handleRouteFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono text-blue-600" placeholder="/opt/deploy/app" />
                    </div>
                  </div>
                )}
              </div>
            </form>
            <div className="border-t border-gray-200 p-5 flex items-center justify-between bg-white shadow-[0_-4px_10px_rgba(0,0,0,0.02)]">
              {routeFormData.ID ? (
                <button type="button" onClick={() => handleRouteDeleteClick(routeFormData.ID)} className="text-red-500 hover:bg-red-50 p-2 rounded-lg transition-colors flex items-center gap-1 text-sm font-medium">
                  <Trash2 size={15} /> 删除环境
                </button>
              ) : <div />}
              <div className="flex gap-3">
                <button type="button" onClick={() => setCommandDrawerOpen(false)} className="px-4 py-2.5 border border-gray-300 text-gray-700 rounded-lg text-sm font-bold hover:bg-gray-100 transition-colors">取消</button>
                <button type="button" onClick={handleRouteSubmit} disabled={submitLoading} className="px-5 py-2.5 bg-black text-white rounded-lg text-sm font-bold hover:bg-gray-800 disabled:opacity-50 transition-colors shadow-sm flex items-center gap-2">
                  <Code size={14} /> {submitLoading ? '保存中...' : '保存配置'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Confirm Modal ── */}
      {confirmModal.isOpen && (
        <div className="fixed inset-0 z-[999] flex items-center justify-center bg-black/40 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-xl shadow-2xl p-6 w-[420px] border border-gray-100 animate-in zoom-in-95 duration-200">
            <h3 className="text-lg font-bold text-gray-900 mb-2">{confirmModal.title}</h3>
            <p className="text-sm text-gray-600 mb-6 leading-relaxed whitespace-pre-wrap">{confirmModal.message}</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setConfirmModal(p => ({ ...p, isOpen: false }))} className="px-4 py-2 text-sm text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg font-medium cursor-pointer">取消</button>
              <button onClick={confirmModal.onConfirm} className={`px-4 py-2 text-sm text-white rounded-lg font-medium shadow-sm flex items-center gap-1 cursor-pointer ${confirmModal.actionColor}`}>{confirmModal.actionText}</button>
            </div>
          </div>
        </div>
      )}

      {/* ── Deploy Log Panel ── */}
      {deployLog.open && (
        <DeployLogPanel
          projectId={deployLog.projectId}
          projectName={deployLog.projectName}
          envKey={deployLog.envKey}
          routeName={deployLog.routeName}
          mode={deployLog.mode}
          onClose={() => setDeployLog(p => ({ ...p, open: false }))}
        />
      )}

      {scriptDialog && (
        <ProjectScripts
          fullscreenDialog
          projectIdOverride={scriptDialog.project.ID}
          routeIdOverride={scriptDialog.route.ID}
          projectName={scriptDialog.project.projectName}
          routeName={scriptDialog.route.routeName}
          onClose={() => setScriptDialog(null)}
        />
      )}

    </div>
  );
}
