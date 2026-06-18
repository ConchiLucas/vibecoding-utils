import React, { useState, useEffect } from 'react';
import { Package, Database, Server, Trash2, Plus, CheckCircle2, Pencil, Eye, EyeOff, Bot } from 'lucide-react';
import toast from 'react-hot-toast';
import { useProjectStore } from '../../stores/useProjectStore';
import { getTbProjectList, createTbProject, deleteTbProject, updateTbProject, TbProject } from '../../api/sysInterfaceProject';
import { getTbConnectionList, createTbConnection, deleteTbConnection, updateTbConnection, testConnectionPayload, TbConnection } from '../../api/sysConnection';
import { useConfirm } from '../../hooks/useConfirm';
import ConfirmDialog from '../../components/ConfirmDialog';
import ServerManager from '../server/ServerManager';
import AIConfigManager from './AIConfigManager';
import DatabaseBrowser from '../../components/DatabaseBrowser';
import TableDataPreview from '../../components/TableDataPreview';
import { resolveSelectedConnectionId } from './ConfigManagerSelection';

const databaseDefaultPorts: Record<string, number> = {
  mysql: 3306,
  pgsql: 5432,
  mssql: 1433,
  oracle: 1521,
  sqlite: 0,
  clickhouse: 8123,
};

export default function ConfigManager() {
  const [activeTab, setActiveTab] = useState<'project' | 'database' | 'server' | 'ai'>('project');
  
  // Project State
  const {
    activeProject,
    activeProjectId,
    activeConnectionId,
    setActiveProject,
    setActiveConnectionId,
  } = useProjectStore();
  const [projects, setProjects] = useState<TbProject[]>([]);
  const [projectLoading, setProjectLoading] = useState(false);
  const [newProjectName, setNewProjectName] = useState('');
  const [newProjectDesc, setNewProjectDesc] = useState('');

  // Project Edit State
  const [editingProjectId, setEditingProjectId] = useState<number | null>(null);
  const [editProjectName, setEditProjectName] = useState('');
  const [editProjectDesc, setEditProjectDesc] = useState('');

  // Database Connection State
  const [connections, setConnections] = useState<TbConnection[]>([]);
  const [connLoading, setConnLoading] = useState(false);
  const [filterEnv, setFilterEnv] = useState<string>('');
  const [isAddingConn, setIsAddingConn] = useState(false);
  const [editingConnId, setEditingConnId] = useState<number | null>(null);
  const [showDbPassword, setShowDbPassword] = useState(false);
  const [connForm, setConnForm] = useState<Partial<TbConnection>>({
    connectionName: '',
    connectionType: 'mysql',
    connectionUrl: '',
    databaseName: '',
    port: 3306,
    dbLoginName: '',
    dbLoginPassword: '',
    envName: ''
  });

  const { confirm, dialogProps } = useConfirm();

  // Database Browser State
  const [dbBrowserOpen, setDbBrowserOpen] = useState(false);
  const [dbBrowserEnv, setDbBrowserEnv] = useState('');
  const [dbBrowserConnectionId, setDbBrowserConnectionId] = useState<number | undefined>(undefined);

  // Table Data Preview State
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewConnId, setPreviewConnId] = useState(0);
  const [previewDbName, setPreviewDbName] = useState('');
  const [previewTableName, setPreviewTableName] = useState('');

  const handleBrowseTableSelect = (value: string) => {
    // value format: "databaseName:tableName"
    const sepIdx = value.indexOf(':');
    if (sepIdx === -1) return;
    const dbName = value.substring(0, sepIdx);
    const tblName = value.substring(sepIdx + 1);
    // We need to find the connectionId for this database.
    // DatabaseBrowser provides connectionId via the RemoteDatabase object's connectionId.
    // Since we don't get it directly from the table select value, we'll set it from the browser's selectedDb.
    setPreviewDbName(dbName);
    setPreviewTableName(tblName);
    setPreviewOpen(true);
  };

  useEffect(() => {
    if (activeTab === 'project') {
      loadProjects();
    } else if (activeTab === 'database') {
      // Auto-resolve activeProjectId if empty but activeProject exists
      if (activeProject && !activeProjectId) {
         getTbProjectList().then(res => {
            const p = res.data?.find(x => x.projectName === activeProject);
            if (p) setActiveProject(p.projectName, p.ID);
            // Will re-trigger because activeProjectId changes? Actually just call loadConnections directly or let the other useEffect catch it.
            setTimeout(loadConnections, 100); 
         });
      } else {
        loadConnections();
      }
    }
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === 'database' && activeProjectId) {
      loadConnections();
    }
  }, [activeProject, activeProjectId]);

  useEffect(() => {
    if (activeTab !== 'database') return;
    const visibleConnections = connections.filter(conn => !filterEnv || conn.envName === filterEnv);
    if (visibleConnections.length === 0) return;
    const nextConnectionId = resolveSelectedConnectionId(activeConnectionId, visibleConnections);
    if (nextConnectionId !== activeConnectionId) {
      setActiveConnectionId(nextConnectionId);
    }
  }, [activeTab, activeConnectionId, connections, filterEnv, setActiveConnectionId]);

  const loadProjects = async () => {
    setProjectLoading(true);
    try {
      const res = await getTbProjectList();
      setProjects(res.data || []);
      // auto select
      if (!activeProject && (res.data || []).length > 0) {
        setActiveProject(res.data[0].projectName, res.data[0].ID);
      } else if (activeProject && !activeProjectId) {
        const found = (res.data || []).find(p => p.projectName === activeProject);
        if (found) setActiveProject(found.projectName, found.ID);
      }
    } catch (e) {
      toast.error('加载项目失败');
    } finally {
      setProjectLoading(false);
    }
  };

  const handleAddProject = async () => {
    if (!newProjectName.trim()) return toast.error('请输入项目名称');
    try {
      await createTbProject({ projectName: newProjectName.trim(), projectDesc: newProjectDesc.trim() });
      toast.success('项目创建成功');
      setNewProjectName('');
      setNewProjectDesc('');
      loadProjects();
    } catch (e) {
      toast.error('项目创建失败');
    }
  };

  const handleDeleteProject = async (p: TbProject) => {
    const confirmed = await confirm(`确定删除项目「${p.projectName}」吗？`);
    if (!confirmed) return;
    try {
      await deleteTbProject({ ID: p.ID });
      toast.success('项目删除成功');
      if (activeProject === p.projectName) {
        setActiveProject('');
      }
      loadProjects();
    } catch (e) {
      toast.error('项目删除失败');
    }
  };

  const handleEditProject = (p: TbProject, e: React.MouseEvent) => {
    e.stopPropagation();
    setEditingProjectId(p.ID);
    setEditProjectName(p.projectName);
    setEditProjectDesc(p.projectDesc || '');
  };

  const handleSaveProject = async (p: TbProject, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!editProjectName.trim()) return toast.error('项目名称不能为空');
    try {
      await updateTbProject({ ID: p.ID, projectName: editProjectName.trim(), projectDesc: editProjectDesc.trim() });
      toast.success('项目更新成功');
      // Sync activeProject name if it was renamed
      if (activeProject === p.projectName && editProjectName.trim() !== p.projectName) {
        setActiveProject(editProjectName.trim(), p.ID);
      }
      setEditingProjectId(null);
      loadProjects();
    } catch (e) {
      toast.error('项目更新失败');
    }
  };

  const handleCancelEditProject = (e: React.MouseEvent) => {
    e.stopPropagation();
    setEditingProjectId(null);
  };

  const loadConnections = async () => {
    if (!activeProjectId) {
      setConnections([]);
      setConnLoading(false);
      return;
    }
    setConnLoading(true);
    try {
      const res = await getTbConnectionList({ page: 1, pageSize: 999, connectionGroup: String(activeProjectId) });
      const list = res.data?.list || [];
      setConnections(list);
      if (list.length === 0) {
        setActiveConnectionId(null);
      }
    } catch (e) {
      toast.error('加载数据库配置失败');
    } finally {
      setConnLoading(false);
    }
  };

  const handleAddOrEditConnection = async () => {
    if (!activeProject || !activeProjectId) return toast.error('请先选择一个活跃项目');
    if (!connForm.connectionName || !connForm.connectionUrl || !connForm.databaseName || !connForm.dbLoginName) {
      return toast.error('请填写完整的数据库信息');
    }
    try {
      if (editingConnId) {
        await updateTbConnection({
          ...connForm,
          ID: editingConnId,
          connectionGroup: String(activeProjectId)
        });
        toast.success('数据库配置修改成功');
      } else {
        await createTbConnection({
          ...connForm,
          connectionGroup: String(activeProjectId)
        });
        toast.success('数据库配置添加成功');
      }
      setConnForm({
        connectionName: '', connectionType: 'mysql', connectionUrl: '', databaseName: '', port: 3306, dbLoginName: '', dbLoginPassword: '', envName: ''
      });
      setEditingConnId(null);
      setIsAddingConn(false);
      setShowDbPassword(false);
      loadConnections();
    } catch (e) {
      toast.error(editingConnId ? '数据库配置修改失败' : '数据库配置添加失败');
    }
  };

  const handleTestConnectionPayload = async () => {
    if (!connForm.connectionType || !connForm.connectionUrl || !connForm.databaseName || !connForm.dbLoginName) {
      return toast.error('请填写完整的数据库信息再测试');
    }
    const loadingToast = toast.loading('测试连接中...');
    try {
      const res: any = await testConnectionPayload(connForm);
      if (res.code === 0) {
        toast.success('连接成功 ✓', { id: loadingToast });
      } else {
        toast.dismiss(loadingToast); // the interceptor already triggers an error toast
      }
    } catch (e: any) {
      toast.dismiss(loadingToast);
    }
  };

  const handleEditConnection = (conn: TbConnection) => {
    setIsAddingConn(false);
    setEditingConnId(conn.ID);
    setShowDbPassword(false);
    setConnForm({
      connectionName: conn.connectionName,
      connectionType: conn.connectionType || 'mysql',
      connectionUrl: conn.connectionUrl,
      databaseName: conn.databaseName,
      port: conn.port,
      dbLoginName: conn.dbLoginName,
      dbLoginPassword: conn.dbLoginPassword || '',
      envName: conn.envName || ''
    });
  };

  const handleCancelEdit = () => {
    setEditingConnId(null);
    setIsAddingConn(false);
    setShowDbPassword(false);
    setConnForm({
      connectionName: '', connectionType: 'mysql', connectionUrl: '', databaseName: '', port: 3306, dbLoginName: '', dbLoginPassword: '', envName: ''
    });
  };

  const handleDeleteConnection = async (id: number) => {
    const confirmed = await confirm('确定删除该数据库配置吗？');
    if (!confirmed) return;
    try {
      await deleteTbConnection({ ID: id });
      toast.success('删除成功');
      loadConnections();
    } catch (e) {
      toast.error('删除失败');
    }
  };

  const visibleConnections = connections.filter(conn => !filterEnv || conn.envName === filterEnv);

  return (
    <div className="flex h-full bg-slate-50 p-6 gap-6">
      <ConfirmDialog {...dialogProps} />
      
      {/* Sidebar */}
      <div className="w-64 shrink-0 flex flex-col gap-3">
        <h2 className="px-2 text-lg font-bold text-slate-800 mb-2">配置管理</h2>
        
        <button
          onClick={() => setActiveTab('project')}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
            activeTab === 'project' 
              ? 'bg-indigo-600 text-white shadow-md' 
              : 'bg-white text-slate-600 hover:bg-indigo-50 hover:text-indigo-600 border border-slate-200'
          }`}
        >
          <Package size={20} />
          <span className="font-medium">项目配置</span>
        </button>

        <button
          onClick={() => setActiveTab('database')}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
            activeTab === 'database' 
              ? 'bg-indigo-600 text-white shadow-md' 
              : 'bg-white text-slate-600 hover:bg-indigo-50 hover:text-indigo-600 border border-slate-200'
          }`}
        >
          <Database size={20} />
          <span className="font-medium">数据库配置</span>
        </button>

        <button
          onClick={() => setActiveTab('server')}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
            activeTab === 'server' 
              ? 'bg-indigo-600 text-white shadow-md' 
              : 'bg-white text-slate-600 hover:bg-indigo-50 hover:text-indigo-600 border border-slate-200'
          }`}
        >
          <Server size={20} />
          <span className="font-medium">服务器配置</span>
        </button>

        <button
          onClick={() => setActiveTab('ai')}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
            activeTab === 'ai' 
              ? 'bg-indigo-600 text-white shadow-md' 
              : 'bg-white text-slate-600 hover:bg-indigo-50 hover:text-indigo-600 border border-slate-200'
          }`}
        >
          <Bot size={20} />
          <span className="font-medium">AI配置</span>
        </button>
      </div>

      {/* Main Content */}
      <div className="flex-1 min-w-0 bg-white rounded-3xl shadow-sm border border-slate-200 overflow-hidden flex flex-col">
        {/* Project View */}
        {activeTab === 'project' && (
          <div className="p-8 flex flex-col h-full">
            <div className="flex items-center justify-between mb-8">
              <div>
                <h3 className="text-2xl font-bold text-slate-900">核心项目配置</h3>
                <p className="text-slate-500 mt-1">创建并选择当前活跃项目，选中项将作为全局环境隔离标识。</p>
              </div>
              <div className="flex gap-2">
                <input 
                  value={newProjectName}
                  onChange={(e) => setNewProjectName(e.target.value)}
                  placeholder="新项目名称..." 
                  className="px-4 py-2 border rounded-xl text-sm w-48 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
                <input 
                  value={newProjectDesc}
                  onChange={(e) => setNewProjectDesc(e.target.value)}
                  placeholder="项目介绍 (可选)..." 
                  className="px-4 py-2 border rounded-xl text-sm w-48 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
                <button onClick={handleAddProject} className="bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-700 flex items-center gap-1">
                  <Plus size={16} /> 新增
                </button>
              </div>
            </div>

            {projectLoading ? (
              <div className="text-slate-400 text-center py-12">加载中...</div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {projects.map((p) => {
                  const isActive = activeProject === p.projectName;
                  const isEditing = editingProjectId === p.ID;
                  return (
                    <div 
                      key={p.ID}
                      onClick={() => !isEditing && setActiveProject(p.projectName, p.ID)}
                      className={`relative group cursor-pointer p-5 rounded-2xl border-2 transition-all ${
                        isActive 
                          ? 'border-indigo-600 bg-indigo-50 shadow-md' 
                          : 'border-slate-200 hover:border-indigo-300 hover:bg-slate-50'
                      }`}
                    >
                      {isActive && !isEditing && (
                        <div className="absolute top-4 right-4 text-indigo-600">
                          <CheckCircle2 size={24} />
                        </div>
                      )}

                      {isEditing ? (
                        // Inline Edit Form
                        <div onClick={e => e.stopPropagation()} className="space-y-3">
                          <div>
                            <label className="block text-xs font-medium text-slate-500 mb-1">项目名称</label>
                            <input
                              autoFocus
                              value={editProjectName}
                              onChange={e => setEditProjectName(e.target.value)}
                              className="w-full px-3 py-2 border border-indigo-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                              placeholder="项目名称..."
                            />
                          </div>
                          <div>
                            <label className="block text-xs font-medium text-slate-500 mb-1">项目介绍</label>
                            <input
                              value={editProjectDesc}
                              onChange={e => setEditProjectDesc(e.target.value)}
                              className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                              placeholder="项目介绍 (可选)..."
                            />
                          </div>
                          <div className="flex gap-2 pt-1">
                            <button
                              onClick={e => handleSaveProject(p, e)}
                              className="flex-1 py-1.5 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition"
                            >保存</button>
                            <button
                              onClick={handleCancelEditProject}
                              className="flex-1 py-1.5 border border-slate-200 text-slate-600 text-sm font-medium rounded-lg hover:bg-slate-50 transition"
                            >取消</button>
                          </div>
                        </div>
                      ) : (
                        // Normal Display
                        <>
                          <div className="flex items-center gap-3 mb-2">
                            <div className={`p-2 rounded-lg ${isActive ? 'bg-indigo-200 text-indigo-700' : 'bg-slate-200 text-slate-600'}`}>
                              <Package size={20} />
                            </div>
                            <h4 className={`text-lg font-bold truncate ${isActive ? 'text-indigo-900' : 'text-slate-800'}`}>
                              {p.projectName}
                            </h4>
                          </div>
                          <p className="text-sm text-slate-500 mt-3 mb-4 truncate" title={p.projectDesc || '暂无项目介绍'}>{p.projectDesc || '暂无项目介绍'}</p>
                          <div className="flex gap-2">
                            <button 
                              onClick={(e) => handleEditProject(p, e)}
                              className="text-xs text-indigo-500 hover:text-indigo-700 font-medium px-3 py-1.5 bg-indigo-50 hover:bg-indigo-100 rounded-lg transition opacity-0 group-hover:opacity-100"
                            >
                              编辑
                            </button>
                            <button 
                              onClick={(e) => { e.stopPropagation(); handleDeleteProject(p); }}
                              className="text-xs text-rose-500 hover:text-rose-700 font-medium px-3 py-1.5 bg-rose-50 hover:bg-rose-100 rounded-lg transition opacity-0 group-hover:opacity-100"
                            >
                              删除项目
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Database View */}
        {activeTab === 'database' && (
          <div className="p-8 flex flex-col h-full overflow-hidden">
            <div className="flex items-center justify-between mb-8 shrink-0">
              <div>
                <h3 className="text-2xl font-bold text-slate-900">环境数据库配置</h3>
              </div>
              <div className="flex items-center gap-3">
                <select 
                  value={filterEnv} 
                  onChange={(e) => setFilterEnv(e.target.value)}
                  className="px-4 py-2 border rounded-xl text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500 min-w-[200px]"
                >
                  <option value="">全部环境</option>
                  {Array.from(new Set(connections.map(c => c.envName).filter(Boolean))).map(env => (
                    <option key={env} value={env}>{env}</option>
                  ))}
                </select>
                <button 
                  onClick={() => {
                    setIsAddingConn(true);
                    setEditingConnId(null);
                    setShowDbPassword(false);
                    setConnForm({
                      connectionName: '', connectionType: 'mysql', connectionUrl: '', databaseName: '', port: 3306, dbLoginName: '', dbLoginPassword: '', envName: ''
                    });
                  }}
                  className="bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-700 flex items-center gap-1"
                >
                  <Plus size={16} /> 新增数据源
                </button>
              </div>
            </div>

            {!activeProject ? (
              <div className="flex-1 flex flex-col items-center justify-center text-slate-400">
                <Package size={48} className="mb-4 opacity-20" />
                <p>请先在「项目配置」中选择一个活跃项目</p>
              </div>
            ) : (
              <div className="flex-1 overflow-y-auto pr-4 space-y-8">
                {/* Add/Edit Form */}
                {(isAddingConn || editingConnId) && (
                  <div className="bg-slate-50 p-6 rounded-2xl border border-slate-200">
                     <h4 className="font-semibold text-slate-800 flex items-center gap-2 mb-4">
                     {editingConnId ? <Pencil size={18} /> : <Plus size={18} />} 
                     {editingConnId ? '编辑数据源连接' : '新增数据源连接'}
                   </h4>
                   <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                     <input placeholder="连接名称 (如: 测试环境主库)" value={connForm.connectionName} onChange={e => setConnForm({...connForm, connectionName: e.target.value})} className="px-3 py-2 border rounded-lg text-sm" />
                     <div className="relative">
                       <input list="env-list" placeholder="部署环境 (选择或新建)" value={connForm.envName || ''} onChange={e => setConnForm({...connForm, envName: e.target.value})} className="px-3 py-2 border rounded-lg text-sm w-full" />
                       <datalist id="env-list">
                         {Array.from(new Set(connections.map(c => c.envName).filter(Boolean))).map(env => (
                           <option key={env} value={env} />
                         ))}
                       </datalist>
                     </div>
                     <select
                       value={connForm.connectionType}
                       onChange={e => {
                         const connectionType = e.target.value;
                         setConnForm({
                           ...connForm,
                           connectionType,
                           port: databaseDefaultPorts[connectionType] ?? connForm.port,
                         });
                       }}
                       className="px-3 py-2 border rounded-lg text-sm bg-white"
                     >
                        <option value="mysql">mysql</option>
                        <option value="pgsql">pgsql</option>
                        <option value="mssql">mssql</option>
                        <option value="oracle">oracle</option>
                        <option value="sqlite">sqlite</option>
                        <option value="clickhouse">clickhouse</option>
                     </select>
                     <input placeholder="Host地址 (如: 127.0.0.1)" value={connForm.connectionUrl} onChange={e => setConnForm({...connForm, connectionUrl: e.target.value})} className="px-3 py-2 border rounded-lg text-sm" />
                     <input type="number" placeholder="端口 (如: 3306)" value={connForm.port || ''} onChange={e => setConnForm({...connForm, port: Number(e.target.value)})} className="px-3 py-2 border rounded-lg text-sm" />
                     <input placeholder="数据库名" value={connForm.databaseName} onChange={e => setConnForm({...connForm, databaseName: e.target.value})} className="px-3 py-2 border rounded-lg text-sm" />
                     <input placeholder="用户名" value={connForm.dbLoginName} onChange={e => setConnForm({...connForm, dbLoginName: e.target.value})} className="px-3 py-2 border rounded-lg text-sm" />
                     <div className="relative">
                       <input
                         placeholder="密码"
                         type={showDbPassword ? 'text' : 'password'}
                         value={connForm.dbLoginPassword}
                         onChange={e => setConnForm({...connForm, dbLoginPassword: e.target.value})}
                         className="w-full px-3 py-2 pr-10 border rounded-lg text-sm"
                         autoComplete="off"
                       />
                       <button
                         type="button"
                         onClick={() => setShowDbPassword(prev => !prev)}
                         className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-slate-400 transition hover:text-indigo-500 focus:outline-none focus:text-indigo-500"
                         aria-label={showDbPassword ? '隐藏密码' : '查看密码'}
                         title={showDbPassword ? '隐藏密码' : '查看密码'}
                       >
                         {showDbPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                       </button>
                     </div>
                   </div>
                   <div className="mt-4 flex gap-3">
                     <button onClick={handleAddOrEditConnection} className="bg-indigo-600 text-white px-5 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700">
                       {editingConnId ? '保存修改' : '确认添加'}
                     </button>
                     <button onClick={handleTestConnectionPayload} className="bg-emerald-600 text-white px-5 py-2 rounded-lg text-sm font-medium hover:bg-emerald-700">
                       测试连接
                     </button>
                     <button onClick={handleCancelEdit} className="bg-white border border-slate-300 text-slate-600 px-5 py-2 rounded-lg text-sm font-medium hover:bg-slate-50">
                       取消
                     </button>
                   </div>
                </div>
                )}

                {/* List */}
                <div className="grid gap-4 lg:grid-cols-2">
                  {visibleConnections.map((conn) => {
                    const isSelected = activeConnectionId === conn.ID;
                    return (
                      <article
                        key={conn.ID}
                        role="button"
                        tabIndex={0}
                        aria-pressed={isSelected}
                        onClick={() => setActiveConnectionId(conn.ID)}
                        onKeyDown={(e) => {
                          if (e.currentTarget !== e.target) return;
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            setActiveConnectionId(conn.ID);
                          }
                        }}
                        className={`group rounded-2xl border-2 p-5 shadow-sm transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 ${
                          isSelected
                            ? 'border-indigo-600 bg-indigo-50 shadow-md'
                            : 'border-slate-200 bg-white hover:border-indigo-300 hover:bg-slate-50 hover:shadow-md'
                        }`}
                      >
                        <div className="flex justify-between items-start mb-4">
                          <div className="flex items-center gap-3">
                            <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                              isSelected ? 'bg-indigo-200 text-indigo-700' : 'bg-indigo-50 text-indigo-600'
                            }`}><Database size={20} /></div>
                            <div>
                              <h4 className={`font-bold flex items-center gap-2 ${isSelected ? 'text-indigo-900' : 'text-slate-800'}`}>
                                {conn.connectionName}
                                {conn.envName && <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-100 text-emerald-700">{conn.envName}</span>}
                              </h4>
                              <p className={`text-xs font-mono ${isSelected ? 'text-indigo-500' : 'text-slate-500'}`}>{conn.connectionType}</p>
                            </div>
                          </div>
                          <div className="flex items-center gap-1">
                            {isSelected && <CheckCircle2 size={22} className="mr-1 shrink-0 text-indigo-600" />}
                            <button onClick={(e) => { e.stopPropagation(); setPreviewConnId(conn.ID); setDbBrowserConnectionId(conn.ID); setDbBrowserEnv(conn.envName || ''); setDbBrowserOpen(true); }} className="text-slate-400 hover:text-emerald-500 transition opacity-0 group-hover:opacity-100 focus:opacity-100 p-2" title="浏览数据库"><Eye size={16} /></button>
                            <button onClick={(e) => { e.stopPropagation(); handleEditConnection(conn); }} className="text-slate-400 hover:text-indigo-500 transition opacity-0 group-hover:opacity-100 focus:opacity-100 p-2" title="编辑数据源"><Pencil size={16} /></button>
                            <button onClick={(e) => { e.stopPropagation(); handleDeleteConnection(conn.ID); }} className="text-slate-400 hover:text-rose-500 transition opacity-0 group-hover:opacity-100 focus:opacity-100 p-2" title="删除数据源"><Trash2 size={16} /></button>
                          </div>
                        </div>
                        <div className={`grid grid-cols-2 gap-2 text-xs p-3 rounded-lg font-mono ${
                          isSelected ? 'bg-white/70 text-indigo-950' : 'bg-slate-50 text-slate-600'
                        }`}>
                          <div className="truncate"><span className="text-slate-400">Host: </span>{conn.connectionUrl}</div>
                          <div className="truncate"><span className="text-slate-400">Port: </span>{conn.port}</div>
                          <div className="truncate"><span className="text-slate-400">DB: </span>{conn.databaseName}</div>
                          <div className="truncate"><span className="text-slate-400">User: </span>{conn.dbLoginName}</div>
                        </div>
                      </article>
                    );
                  })}
                  {visibleConnections.length === 0 && !connLoading && (
                    <div className="col-span-1 lg:col-span-2 text-center text-slate-400 py-8 border-2 border-dashed border-slate-200 rounded-2xl">
                      {filterEnv ? '当前环境暂无数据库配置' : '暂无属于该项目的数据库配置'}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Server View */}
        {activeTab === 'server' && (
          <div className="p-8 flex flex-col h-full overflow-hidden">
            <ServerManager />
          </div>
        )}

        {/* AI View */}
        {activeTab === 'ai' && (
          <div className="p-8 flex flex-col h-full overflow-hidden">
            <AIConfigManager />
          </div>
        )}
      </div>

      {/* Database Browser Modal */}
      {activeProjectId && (
        <DatabaseBrowser
          open={dbBrowserOpen}
          onClose={() => { setDbBrowserOpen(false); setDbBrowserConnectionId(undefined); }}
          environment={dbBrowserEnv}
          environments={Array.from(new Set(connections.map(c => c.envName).filter(Boolean))) as string[]}
          onEnvironmentChange={(env) => { setDbBrowserConnectionId(undefined); setDbBrowserEnv(env); }}
          projectId={activeProjectId}
          focusedConnectionId={dbBrowserConnectionId}
          autoClose={false}
          onTableSelect={(value, connectionId) => {
            // Don't close the browser — open preview instead
            const sepIdx = value.indexOf(':');
            if (sepIdx === -1) return;
            const dbName = value.substring(0, sepIdx);
            const tblName = value.substring(sepIdx + 1);
            setPreviewDbName(dbName);
            setPreviewTableName(tblName);
            if (connectionId) setPreviewConnId(connectionId);
            setPreviewOpen(true);
          }}
        />
      )}

      {/* Table Data Preview Modal */}
      <TableDataPreview
        open={previewOpen}
        onClose={() => setPreviewOpen(false)}
        connectionId={previewConnId}
        databaseName={previewDbName}
        tableName={previewTableName}
      />
    </div>
  );
}
