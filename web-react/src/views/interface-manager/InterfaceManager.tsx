import { useEffect, useState, useRef } from 'react';
import toast from 'react-hot-toast';
import { Route, Trash2, Activity, Package, TreeDeciduous, Upload, ChevronRight, ChevronDown, ChevronLeft, Folder, FileJson, Database, X, Server, Users, Pencil, Check } from 'lucide-react';
import { getTbInterfaceList, deleteTbInterface, TbInterface } from '../../api/sysInterface';
import { uploadSwaggerJson, buildServerTree, TreeNode, renameServer } from '../../api/sysInterfaceServer';
import { getTbServerUserList, createTbServerUser, updateTbServerUser, deleteTbServerUser, TbServerUser } from '../../api/sysInterfaceServerUser';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

// Inline simple drawer/modal wrapper for Params & Logs
import InterfaceLogManager from '../interface-log-manager/InterfaceLogManager';
import InterfaceParamsManager from '../interface-params-manager/InterfaceParamsManager';
import ColumnTreeManager from './ColumnTreeManager';
import InterfaceTestManager from '../interface-test-manager/InterfaceTestManager';
import { deleteTbInterfaceServer } from '../../api/sysInterfaceServer';
import { getTbInterfaceEnvList, createTbInterfaceEnv, updateTbInterfaceEnv, deleteTbInterfaceEnv, TbInterfaceEnv } from '../../api/sysInterfaceEnv';
import { useProjectStore } from '../../stores/useProjectStore';

type ActiveTabType = 'params' | 'logs' | 'inParams' | 'outParams' | 'test';


// Helper: format a date string into relative time like "3分钟前"
function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.max(0, now - then);
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return '刚刚';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}天前`;
  return new Date(dateStr).toLocaleDateString();
}

export default function InterfaceManager() {
  const { activeProject } = useProjectStore();

  const [interfaces, setInterfaces] = useState<TbInterface[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(15);
  const [total, setTotal] = useState(0);
  const { confirm, dialogProps } = useConfirm();

  // Search
  const [searchName, setSearchName] = useState('');
  const [searchPath, setSearchPath] = useState('');

  // Tree State
  const [treeData, setTreeData] = useState<TreeNode[]>([]);
  const [treeSearch, setTreeSearch] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<Set<number>>(new Set());
  const [selectedNodeId, setSelectedNodeId] = useState<number | null>(null);

  // Filtered Interface List based on tree selection
  const [selectedProjectName, setSelectedProjectName] = useState<string>('');
  const [selectedServerName, setSelectedServerName] = useState<string>('');
  const [selectedInterfaceName, setSelectedInterfaceName] = useState<string>('');

  // Project name suggestions for import combo box
  const [projectSuggestions, setProjectSuggestions] = useState<string[]>([]);
  const [showProjectDropdown, setShowProjectDropdown] = useState(false);

  // Dialog states for Sub-managers
  const [activeTab, setActiveTab] = useState<ActiveTabType | null>(null);
  const [activeInterface, setActiveInterface] = useState<string>('');
  const [activeInterfaceId, setActiveInterfaceId] = useState<number>(0);

  // Import Dialog states
  const [importOpen, setImportOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importForm, setImportForm] = useState({
     projectName: activeProject,
     serverName: ''
  });
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Env List Dialog
  const [envListOpen, setEnvListOpen] = useState(false);
  const [envList, setEnvList] = useState<TbInterfaceEnv[]>([]);
  const [envListLoading, setEnvListLoading] = useState(false);
  const [newEnvUrl, setNewEnvUrl] = useState('');
  const [newEnvName, setNewEnvName] = useState('');
  const [isAddingEnv, setIsAddingEnv] = useState(false);
  const [editingEnvId, setEditingEnvId] = useState<number | null>(null);
  const [editEnvUrl, setEditEnvUrl] = useState('');
  const [editEnvName, setEditEnvName] = useState('');

  // User Management Dialog
  const [userListOpen, setUserListOpen] = useState(false);
  const [userList, setUserList] = useState<TbServerUser[]>([]);
  const [userListLoading, setUserListLoading] = useState(false);
  const [isAddingUser, setIsAddingUser] = useState(false);
  const [editingUserId, setEditingUserId] = useState<number | null>(null);
  const [userForm, setUserForm] = useState({ loginAccount: '', loginPassword: '', userNickname: '', roleCode: '', roleName: '', environment: '', requestHeader: '' });
  const [editUserForm, setEditUserForm] = useState({ loginAccount: '', loginPassword: '', userNickname: '', roleCode: '', roleName: '', environment: '', requestHeader: '' });

  // Header Modal State
  const [headerModalOpen, setHeaderModalOpen] = useState(false);
  const [headerUserId, setHeaderUserId] = useState<number | null>(null);
  const [headerValue, setHeaderValue] = useState('');

  // Tree node inline rename state
  const [renamingNodeId, setRenamingNodeId] = useState<number | null>(null);
  const [renameValue, setRenameValue] = useState('');

  const resetInterfaceViewState = (projectName = '') => {
    setSelectedNodeId(null);
    setSelectedProjectName(projectName);
    setSelectedServerName('');
    setSelectedInterfaceName('');
    setInterfaces([]);
    setTotal(0);
    setPage(1);
    setActiveTab(null);
    setActiveInterface('');
    setActiveInterfaceId(0);
  };

  // Re-fetch tree when active project changes
  useEffect(() => {
    resetInterfaceViewState(activeProject || '');
    setTreeData([]);
    setExpandedKeys(new Set());
    void fetchTreeData();
  }, [activeProject]);

  async function fetchTreeData() {
    try {
      const res = await buildServerTree(activeProject || undefined);
      const data = res.data || [];
      setTreeData(data);
      if (data.length === 0) {
        void loadData(1, undefined, undefined, activeProject || undefined);
        return;
      }
      // Auto expand all project nodes
      setExpandedKeys(new Set(data.map(n => n.id)));
      // Auto-select first project node for the current active project.
      if (!selectedNodeId) {
        const firstNode = data[0];
        setSelectedNodeId(firstNode.id);
        const pName = firstNode.projectName || firstNode.interfaceName;
        setSelectedProjectName(pName);
        setSelectedServerName('');
        setSelectedInterfaceName('');
        void loadData(1, undefined, undefined, pName);
      }
    } catch (err) {
      console.error('Tree fetch error:', err);
    }
  }

  async function loadData(targetPage: number, serverName?: string, interfaceNameFilter?: string, projectName?: string) {
    setLoading(true);
    try {
      const res = await getTbInterfaceList({ 
        page: targetPage, 
        pageSize: pageSize,
        interfaceName: interfaceNameFilter || searchName || undefined,
        paths: searchPath || undefined,
        serverName: serverName || undefined,
        projectName: projectName || undefined
      });
      const list = res.data?.list ?? [];
      setInterfaces(list);
      setTotal(res.data?.total ?? 0);
      setPage(targetPage);
    } catch (error) {
      console.error(error);
      toast.error('加载接口数据失败');
    } finally {
      setLoading(false);
    }
  }

  // Tree Handlers
  const toggleExpand = (id: number, e: React.MouseEvent) => {
    e.stopPropagation();
    const newExpanded = new Set(expandedKeys);
    if (newExpanded.has(id)) newExpanded.delete(id);
    else newExpanded.add(id);
    setExpandedKeys(newExpanded);
  };

  const handleNodeClick = (node: TreeNode) => {
    setSelectedNodeId(node.id);
    const hasChildren = node.children && node.children.length > 0;
    if (hasChildren) {
      // Project-level node (L1): filter by project name
      const pName = node.projectName || node.interfaceName;
      setSelectedProjectName(pName);
      setSelectedServerName('');
      setSelectedInterfaceName('');
      void loadData(1, undefined, undefined, pName);
    } else {
      // Service-level node (L2): filter by server name
      const sName = node.serverName || node.interfaceName;
      const pName = node.projectName || '';
      setSelectedProjectName(pName);
      setSelectedServerName(sName);
      setSelectedInterfaceName('');
      void loadData(1, sName, undefined, pName);
    }
  };

  // Rest of handlers
  async function handleDelete(iface: TbInterface) {
    const confirmed = await confirm(`确定要删除接口配置「${iface.interfaceName}」吗？`, {
      title: '删除接口明细',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbInterface({ ID: iface.ID });
      toast.success('接口配置已删除');
      void loadData(page, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined);
      void fetchTreeData();
    } catch (error) {
      console.error(error);
      toast.error('删除失败');
    }
  }

  const handleOpenTesting = (iface: TbInterface) => {
     setActiveInterface(iface.paths || '');
     setActiveInterfaceId(iface.ID);
     setActiveTab('test');
  };

  const handleOpenLogs = (paths: string) => {
     setActiveInterface(paths);
     setActiveTab('logs');
  };

  const handleOpenColumnTree = (iface: TbInterface, type: 'inParams' | 'outParams') => {
    setActiveInterface(iface.paths || '');
    setActiveInterfaceId(iface.ID);
    setActiveTab(type);
  };

  // ---- Tree Node Rename / Delete ----
  const handleStartRename = (node: TreeNode, e: React.MouseEvent) => {
    e.stopPropagation();
    setRenamingNodeId(node.id);
    setRenameValue(node.interfaceName);
  };

  const handleConfirmRename = async (node: TreeNode, e?: React.MouseEvent) => {
    e?.stopPropagation();
    const trimmed = renameValue.trim();
    if (!trimmed || trimmed === node.interfaceName) {
      setRenamingNodeId(null);
      return;
    }
    try {
      await renameServer({ ID: node.id, serverName: trimmed });
      toast.success(`已重命名为「${trimmed}」`);
      // Update local selected state if this node was selected
      if (selectedServerName === node.interfaceName) {
        setSelectedServerName(trimmed);
      }
      setRenamingNodeId(null);
      void fetchTreeData();
    } catch (err: any) {
      toast.error(err?.response?.data?.msg || '重命名失败');
    }
  };

  const handleDeleteNode = async (node: TreeNode, e: React.MouseEvent) => {
    e.stopPropagation();
    const confirmed = await confirm(
      `确定要删除服务节点「${node.interfaceName}」吗？\n这将彻底删除该服务导入的服务、接口、实体、字段数据。`,
      { title: '删除服务节点', confirmText: '确定删除' }
    );
    if (!confirmed) return;
    try {
      await deleteTbInterfaceServer({ ID: node.id });
      toast.success('服务节点已删除');
      if (selectedNodeId === node.id) {
        resetInterfaceViewState(selectedProjectName || activeProject || '');
      }
      void fetchTreeData();
    } catch (err: any) {
      toast.error(err?.response?.data?.msg || '删除失败');
    }
  };

  // ---- Env List Features ----
  const loadEnvList = () => {
     if (!selectedProjectName) return;
     setEnvListLoading(true);
     getTbInterfaceEnvList({ page: 1, pageSize: 999, projectName: selectedProjectName })
       .then(res => setEnvList(res.data?.list || []))
       .catch(() => toast.error('加载环境列表失败'))
       .finally(() => setEnvListLoading(false));
  };

  const handleAddEnv = async () => {
     if (!newEnvUrl || !newEnvName) return;
     try {
       await createTbInterfaceEnv({ projectName: selectedProjectName, envName: newEnvName, baseUrl: newEnvUrl });
       toast.success('环境地址已添加');
       setNewEnvUrl('');
       setNewEnvName('');
       setIsAddingEnv(false);
       loadEnvList();
     } catch (err) {
       toast.error('添加失败');
     }
  };

  const handleUpdateEnv = async (id: number) => {
     if (!editEnvUrl || !editEnvName) return;
     try {
       await updateTbInterfaceEnv({ ID: id, baseUrl: editEnvUrl, envName: editEnvName });
       toast.success('环境地址已更新');
       setEditingEnvId(null);
       setEditEnvUrl('');
       setEditEnvName('');
       loadEnvList();
     } catch (err) {
       toast.error('更新失败');
     }
  };

  const handleDeleteEnv = async (id: number) => {
     const confirmed = await confirm('确定要删除这个环境地址吗？');
     if (!confirmed) return;
     try {
       await deleteTbInterfaceEnv({ ID: id });
       toast.success('环境地址删除成功');
       loadEnvList();
     } catch (err) {
       toast.error('删除失败');
     }
  };

  // ---- User Management Features ----
  const loadUserList = () => {
     if (!selectedProjectName) return;
     setUserListLoading(true);
     getTbServerUserList({ page: 1, pageSize: 999, projectName: selectedProjectName })
       .then(res => setUserList(res.data?.list || []))
       .catch(() => toast.error('加载用户列表失败'))
       .finally(() => setUserListLoading(false));
  };

  const handleAddUser = async () => {
     if (!userForm.loginAccount || !userForm.loginPassword) {
       toast.error('登陆账号和登陆密码为必填项');
       return;
     }
     try {
       await createTbServerUser({ projectName: selectedProjectName, ...userForm });
       toast.success('用户已添加');
       setUserForm({ loginAccount: '', loginPassword: '', userNickname: '', roleCode: '', roleName: '', environment: '' });
       setIsAddingUser(false);
       loadUserList();
     } catch (err) {
       toast.error('添加失败');
     }
  };

  const handleUpdateUser = async (id: number) => {
     if (!editUserForm.loginAccount || !editUserForm.loginPassword) {
       toast.error('登陆账号和登陆密码为必填项');
       return;
     }
     try {
       await updateTbServerUser({ ID: id, ...editUserForm });
       toast.success('用户已更新');
       setEditingUserId(null);
       loadUserList();
     } catch (err) {
       toast.error('更新失败');
     }
  };

  const handleDeleteUser = async (id: number) => {
     const confirmed = await confirm('确定要删除这个用户吗？');
     if (!confirmed) return;
     try {
       await deleteTbServerUser({ ID: id });
       toast.success('用户删除成功');
       loadUserList();
     } catch (err) {
       toast.error('删除失败');
     }
  };

  const handleSaveHeader = async () => {
    if (!headerUserId) return;
    try {
      await updateTbServerUser({ ID: headerUserId, requestHeader: headerValue });
      toast.success('Header已保存');
      setHeaderModalOpen(false);
      loadUserList();
    } catch (err) {
      toast.error('保存失败');
    }
  };

  // ---- Import Feature ----
  const closeImportDialog = () => {
    setImportOpen(false);
    setImportFile(null);
    setImportForm({ projectName: activeProject, serverName: '' });
  };

  const openImportModal = () => {
    setImportForm({ projectName: activeProject, serverName: '' });
    setImportFile(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
    // Load existing project names for suggestions (filtered by active project)
    buildServerTree(activeProject || undefined).then(res => {
      const names = (res.data || []).map((n: TreeNode) => n.interfaceName).filter(Boolean);
      setProjectSuggestions([...new Set(names)] as string[]);
    }).catch(() => {});
    setImportOpen(true);
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) {
      setImportFile(null);
      return;
    }
    // 只校验扩展名，不限制 MIME type（不同OS对JSON的MIME识别不同）
    if (!file.name.endsWith('.json')) {
      toast.error('只能上传 .json 格式文件');
      e.target.value = '';
      setImportFile(null);
      return;
    }
    setImportFile(file);
  };

  const handleImportSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeProject) {
      toast.error('请先在"配置管理"中选择活跃项目');
      return;
    }
    if (!importForm.serverName) {
      toast.error('请填写服务名称');
      return;
    }
    // 直接从 DOM ref 读取文件，避免 React state 异步更新导致的值为 null 问题
    const fileToUpload = fileInputRef.current?.files?.[0] ?? importFile;
    if (!fileToUpload) {
      toast.error('请上传 JSON 文件');
      return;
    }
    setImporting(true);
    try {
      const formData = new FormData();
      formData.append('projectName', activeProject);
      formData.append('serverName', importForm.serverName);
      formData.append('file', fileToUpload);

      await uploadSwaggerJson(formData);
      toast.success('导入成功');
      setImportOpen(false);
      void fetchTreeData();
      void loadData(1, selectedServerName, undefined, activeProject || selectedProjectName || undefined);
    } catch (err: any) {
      toast.error('提交失败: ' + (err?.response?.data?.msg || err.message || '未知错误'));
    } finally {
      setImporting(false);
    }
  };

  // Render tree node recursive helper
  const renderTreeNode = (node: TreeNode, depth = 0) => {
    const isExpanded = expandedKeys.has(node.id);
    const isSelected = selectedNodeId === node.id;
    const hasChildren = node.children && node.children.length > 0;
    const isMatched = !treeSearch || node.interfaceName.includes(treeSearch);
    const isRenaming = renamingNodeId === node.id;

    if (!isMatched && !hasChildren) return null;

    return (
      <div key={node.id}>
        <div
          onClick={() => !isRenaming && handleNodeClick(node)}
          className={`group flex items-center gap-1.5 py-1.5 px-2 rounded-lg cursor-pointer transition ${
            isSelected ? 'bg-indigo-50 text-indigo-700 font-medium' : 'hover:bg-slate-50 text-slate-700'
          }`}
          style={{ paddingLeft: `${depth * 1.5 + 0.5}rem` }}
        >
          {hasChildren ? (
            <button onClick={(e) => toggleExpand(node.id, e)} className="p-0.5 text-slate-400 hover:text-slate-600">
               {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </button>
          ) : (
            <span className="w-[18px]"></span>
          )}
          {hasChildren
            ? <Package size={14} className={isSelected ? 'text-indigo-500' : 'text-amber-500'}/>
            : <Server size={14} className={isSelected ? 'text-indigo-400' : 'text-slate-400'}/>
          }

          {/* Rename inline input (leaf nodes only) */}
          {!hasChildren && isRenaming ? (
            <>
              <input
                autoFocus
                value={renameValue}
                onChange={e => setRenameValue(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') handleConfirmRename(node);
                  if (e.key === 'Escape') setRenamingNodeId(null);
                }}
                onClick={e => e.stopPropagation()}
                className="flex-1 min-w-0 text-sm bg-white border border-indigo-300 rounded px-1.5 py-0.5 outline-none focus:ring-1 focus:ring-indigo-500"
              />
              <button
                onClick={e => handleConfirmRename(node, e)}
                className="ml-1 p-0.5 text-emerald-600 hover:text-emerald-800 shrink-0"
                title="确认"
              >
                <Check size={14} />
              </button>
              <button
                onClick={e => { e.stopPropagation(); setRenamingNodeId(null); }}
                className="p-0.5 text-slate-400 hover:text-slate-600 shrink-0"
                title="取消"
              >
                <X size={14} />
              </button>
            </>
          ) : (
            <>
              <span className="text-sm truncate flex-1" title={node.interfaceName}>{node.interfaceName}</span>
              {/* Hover actions — shown only for leaf (L2) nodes */}
              {!hasChildren && (
                <span className="ml-auto flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                  <button
                    onClick={e => handleStartRename(node, e)}
                    className="p-0.5 rounded text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 transition"
                    title="重命名"
                  >
                    <Pencil size={12} />
                  </button>
                  <button
                    onClick={e => handleDeleteNode(node, e)}
                    className="p-0.5 rounded text-slate-400 hover:text-rose-600 hover:bg-rose-50 transition"
                    title="删除"
                  >
                    <Trash2 size={12} />
                  </button>
                </span>
              )}
            </>
          )}
        </div>
        {hasChildren && isExpanded && (
          <div>{node.children!.map(child => renderTreeNode(child, depth + 1))}</div>
        )}
      </div>
    );
  };


  return (
    <div className="flex gap-6 items-start h-[calc(100vh-8rem)]">
      
      {/* LEFT: Org Tree Server Panel mapped to org-tree.vue */}
      <aside className="w-64 shrink-0 bg-white border border-slate-200 rounded-2xl p-4 flex flex-col h-full shadow-sm">
         <div className="mb-4">
             <input 
               type="text" 
               placeholder="请输入关键字过滤节点" 
               value={treeSearch}
               onChange={e => setTreeSearch(e.target.value)}
               className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:border-indigo-500 outline-none"
             />
         </div>
         <div className="flex-1 overflow-y-auto min-h-0 space-y-0.5 select-none pr-2">
            {treeData.length === 0 ? (
               <div className="text-center text-xs text-slate-400 mt-10">暂无服务节点</div>
            ) : (
               treeData.map(node => renderTreeNode(node, 0))
            )}
         </div>
      </aside>

      {/* RIGHT: Main Interface List Area */}
      <div className="flex-1 space-y-6 overflow-hidden flex flex-col h-full">
         
         <div className="bg-white p-4 rounded-2xl border border-slate-200 flex justify-between items-center shadow-sm shrink-0">
             <div className="flex gap-3">
                 <input type="text" placeholder="接口路径" value={searchPath} onChange={e=>setSearchPath(e.target.value)} className="px-3 py-2 border border-slate-200 rounded-lg text-sm bg-slate-50 outline-none w-56" />
                 <input type="text" placeholder="接口名称" value={searchName} onChange={e=>setSearchName(e.target.value)} className="px-3 py-2 border border-slate-200 rounded-lg text-sm bg-slate-50 outline-none w-48" />
                 <button onClick={() => loadData(1, selectedServerName, selectedInterfaceName, selectedProjectName)} className="px-5 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition">查询</button>
             </div>
             <div className="flex items-center gap-3">
                 {activeProject && (
                   <div
                    className="hidden lg:inline-flex h-10 max-w-[220px] items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-3 text-sm font-semibold text-indigo-700"
                    title={`当前项目：${activeProject}`}
                   >
                    <Folder size={16} className="shrink-0 text-indigo-500" />
                    <span className="shrink-0 text-xs font-bold text-indigo-400">项目</span>
                    <span className="min-w-0 truncate">{activeProject}</span>
                   </div>
                 )}
                 <button 
                  onClick={() => {
                     loadEnvList();
                     setEnvListOpen(true);
                 }} className="px-4 py-2 bg-slate-100 text-slate-700 text-sm font-medium rounded-lg hover:bg-slate-200 transition inline-flex items-center gap-1.5 disabled:opacity-50"
                 >
                    <Database size={16}/> 环境列表
                 </button>
                 <button 
                  onClick={() => {
                     loadUserList();
                     setUserListOpen(true);
                 }} className="px-4 py-2 bg-slate-100 text-slate-700 text-sm font-medium rounded-lg hover:bg-slate-200 transition inline-flex items-center gap-1.5"
                 >
                    <Users size={16}/> 用户管理
                 </button>
                 <button onClick={openImportModal} className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition inline-flex items-center gap-1.5"><Upload size={16}/> 导入服务</button>
             </div>
         </div>

         <div className="flex-1 bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden flex flex-col min-h-0">
            {loading ? (
                <div className="flex-1 flex items-center justify-center text-sm text-slate-500 border-2 border-dashed border-slate-100 m-8 rounded-2xl">
                  正在加载接口明细...
                </div>
            ) : (
                <div className="overflow-y-auto flex-1">
                   {interfaces.length === 0 ? (
                      <div className="flex flex-col items-center justify-center h-48 text-slate-400 text-sm">
                         <div className="mb-2 p-3 bg-slate-50 rounded-full"><Route size={24} /></div>
                         {selectedServerName ? `服务 [${selectedServerName}] 暂无接口` : '暂无数据'}
                      </div>
                   ) : (
                     <table className="w-full text-left text-sm text-slate-600">
                       <thead className="bg-slate-50 text-slate-500 border-b border-slate-200 text-xs uppercase tracking-wider sticky top-0 z-10">
                         <tr>
                            <th className="px-6 py-4 font-bold">接口路径</th>
                            <th className="px-6 py-4 font-bold">接口名称</th>
                            <th className="px-6 py-4 font-bold">所属服务</th>
                            <th className="px-6 py-4 font-bold">最近测试</th>
                            <th className="px-6 py-4 font-bold text-center">操作</th>
                         </tr>
                       </thead>
                       <tbody className="divide-y divide-slate-100">
                         {interfaces.map((iface) => (
                            <tr key={iface.ID} className="hover:bg-slate-50/50 transition">
                               <td className="px-6 py-4 font-mono text-indigo-600 font-medium">[{iface.requestType?.toUpperCase()}] {iface.paths}</td>
                               <td className="px-6 py-4">{iface.interfaceName || '-'}</td>
                               <td className="px-6 py-4"><span className="px-2 py-1 bg-slate-100 text-slate-600 rounded text-xs">{iface.serverName}</span></td>
                               <td className="px-6 py-4 whitespace-nowrap">
                                 {iface.lastTestedAt ? (
                                   <span className="inline-flex items-center gap-1.5 text-xs">
                                     <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0"></span>
                                     <span className="text-slate-600" title={new Date(iface.lastTestedAt).toLocaleString()}>{formatRelativeTime(iface.lastTestedAt)}</span>
                                   </span>
                                 ) : (
                                   <span className="text-xs text-slate-300">未测试</span>
                                 )}
                               </td>
                               <td className="px-6 py-4 text-center space-x-3 whitespace-nowrap">
                                   <button onClick={() => handleOpenTesting(iface)} className="inline-flex items-center gap-1 text-indigo-600 hover:text-indigo-800 transition"><Package size={14}/> 接口测试</button>
                                   <button onClick={() => handleOpenLogs(iface.paths || '')} className="inline-flex items-center gap-1 text-emerald-600 hover:text-emerald-800 transition"><Activity size={14}/> 调用记录</button>
                                   <button onClick={() => handleOpenColumnTree(iface, 'inParams')} className="inline-flex items-center gap-1 text-slate-500 hover:text-slate-800 transition"><TreeDeciduous size={14}/> 入参树</button>
                                   <button onClick={() => handleOpenColumnTree(iface, 'outParams')} className="inline-flex items-center gap-1 text-slate-500 hover:text-slate-800 transition"><TreeDeciduous size={14}/> 出参树</button>
                                   <button onClick={() => handleDelete(iface)} className="ml-2 inline-flex items-center gap-1 text-rose-500 hover:text-rose-700 transition"><Trash2 size={14}/> 移除</button>
                               </td>
                            </tr>
                         ))}
                       </tbody>
                     </table>
                   )}
                </div>
            )}
             {/* Pagination Controls */}
             <div className="shrink-0 px-6 py-3 border-t border-slate-200 bg-slate-50 flex items-center justify-between text-sm text-slate-600 rounded-b-2xl">
               <span>共 <b>{total}</b> 条</span>
               <div className="flex items-center gap-3">
                 <select value={pageSize}
                   onChange={e => { setPageSize(Number(e.target.value)); setTimeout(() => loadData(1, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined), 0); }}
                   className="px-2 py-1 border border-slate-200 rounded text-sm bg-white">
                   {[10, 15, 20, 50, 100].map(s => <option key={s} value={s}>{s}条/页</option>)}
                 </select>
                 <div className="flex items-center gap-1">
                   <button disabled={page <= 1} onClick={() => loadData(page - 1, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined)}
                     className="p-1 rounded hover:bg-slate-200 disabled:opacity-30 disabled:cursor-not-allowed transition">
                     <ChevronLeft size={16} />
                   </button>
                   {(() => {
                     const totalPages = Math.ceil(total / pageSize) || 1;
                     let start = Math.max(1, page - 2);
                     const end = Math.min(totalPages, start + 4);
                     start = Math.max(1, end - 4);
                     const pages: number[] = [];
                     for (let i = start; i <= end; i++) pages.push(i);
                     return pages.map(p => (
                       <button key={p} onClick={() => loadData(p, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined)}
                         className={`w-8 h-8 rounded text-sm font-medium transition ${p === page ? 'bg-indigo-600 text-white' : 'hover:bg-slate-200'}`}>{p}</button>
                     ));
                   })()}
                   <button disabled={page >= Math.ceil(total / pageSize)} onClick={() => loadData(page + 1, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined)}
                     className="p-1 rounded hover:bg-slate-200 disabled:opacity-30 disabled:cursor-not-allowed transition">
                     <ChevronRight size={16} />
                   </button>
                 </div>
                 <div className="flex items-center gap-1">
                   <span className="text-xs text-slate-400">跳往</span>
                   <input type="number" min={1} max={Math.ceil(total / pageSize)}
                     className="w-14 px-2 py-1 border border-slate-200 rounded text-sm text-center bg-white"
                     onKeyDown={e => { if (e.key === 'Enter') { const v = Number((e.target as HTMLInputElement).value); if (v >= 1 && v <= Math.ceil(total / pageSize)) loadData(v, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined); }}}
                   />
                   <span className="text-xs text-slate-400">页</span>
                 </div>
               </div>
             </div>
         </div>
      </div>

      {/* Full Screen Hidden Modal logic restoring Vue's component rendering */}
      {activeTab && (
         <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-in fade-in">
             <div className="bg-slate-50 w-full max-w-6xl h-[85vh] rounded-2xl shadow-2xl overflow-hidden flex flex-col relative animate-in zoom-in-95">
                 <div className="shrink-0 flex items-center justify-between px-6 py-4 bg-white border-b border-slate-200">
                     <h3 className="font-semibold text-lg">
                        {activeTab === 'params' && `环境参数接口覆盖 (Target: ${activeInterface})`}
                        {activeTab === 'logs' && `哨兵审计回放记录 (Target: ${activeInterface})`}
                        {(activeTab === 'inParams' || activeTab === 'outParams') && `接口参数树视图 (Target: ${activeInterface})`}
                        {activeTab === 'test' && `自动化拨测沙盒 (Target: ${activeInterface})`}
                     </h3>
                     <button onClick={() => { setActiveTab(null); void loadData(page, selectedServerName, selectedInterfaceName, selectedProjectName || activeProject || undefined); }} className="p-2 bg-slate-100 hover:bg-rose-100 hover:text-rose-600 rounded-lg transition font-mono font-bold">ESC 关闭</button>
                 </div>
                 <div className="flex-1 overflow-y-auto p-6">
                     {activeTab === 'params' && <InterfaceParamsManager />}
                     {activeTab === 'logs' && <InterfaceLogManager interfacePaths={activeInterface} />}
                     {activeTab === 'inParams' && <ColumnTreeManager interfaceId={activeInterfaceId} type={1} />}
                     {activeTab === 'outParams' && <ColumnTreeManager interfaceId={activeInterfaceId} type={2} />}
                     {activeTab === 'test' && <InterfaceTestManager interfaceId={activeInterfaceId} interfacePaths={activeInterface} projectName={selectedProjectName} />}
                 </div>
             </div>
         </div>
      )}

      {/* Import Swagger JSON Modal */}
      {importOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm animate-in fade-in">
          <div className="bg-white rounded-2xl w-full max-w-md shadow-xl overflow-hidden">
             <div className="p-6 border-b border-slate-100">
                <h3 className="text-lg font-semibold">导入服务结构</h3>
             </div>
             <form onSubmit={handleImportSubmit} className="p-6 space-y-4">
                <div className="relative">
                   <label className="block text-sm font-medium text-slate-700 mb-1">项目名称 <span className="text-rose-500">*</span></label>
                   <input 
                      disabled
                      type="text" 
                      value={activeProject || '未选择配置项目'} 
                      className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm bg-slate-100 text-slate-500 cursor-not-allowed" 
                      placeholder="请先在全局配置管理选择项目" 
                   />
                   {!activeProject && <p className="text-xs text-amber-500 mt-1">上传服务数据需要绑定项目，请前往左侧导航"配置管理"选择活跃项目。</p>}
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">服务名称 <span className="text-rose-500">*</span></label>
                   <input required type="text" value={importForm.serverName} onChange={e => setImportForm({...importForm, serverName: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm" placeholder="如: pay-service" />
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">上传 JSON <span className="text-rose-500">*</span></label>
                   <div className="mt-1">
                      <input 
                        ref={fileInputRef} type="file" accept=".json" onChange={handleFileChange}
                        className="block w-full text-sm text-slate-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100 cursor-pointer"
                      />
                   </div>
                   {importFile ? (
                     <p className="mt-1.5 text-xs text-emerald-600">✓ 已选择: <span className="font-medium">{importFile.name}</span></p>
                   ) : (
                     <p className="mt-1.5 text-xs text-slate-400">未选择文件</p>
                   )}
                </div>
                <div className="pt-4 border-t border-slate-100 flex justify-end gap-2">
                   <button type="button" onClick={() => setImportOpen(false)} className="px-4 py-2 text-sm font-medium text-slate-600 bg-slate-100 rounded-lg">取消</button>
                   <button type="submit" disabled={importing} className="px-6 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg">
                     {importing ? '解析并导入中...' : '提交'}
                   </button>
                </div>
             </form>
          </div>
        </div>
      )}

      {/* Environment/Server List Modal */}
      {envListOpen && (
        <div className="fixed inset-0 z-50 flex flex-col bg-slate-50 animate-in fade-in zoom-in-95 duration-200">
             <div className="px-8 py-5 bg-white border-b border-slate-200 flex items-center justify-between shrink-0 shadow-sm">
                <h3 className="text-xl font-semibold flex items-center gap-2">
                   <Database size={24} className="text-indigo-600" /> 服务环境列表
                </h3>
                <button onClick={() => setEnvListOpen(false)} className="text-slate-400 hover:text-slate-600 p-2 rounded-full hover:bg-slate-100 transition"><X size={24} /></button>
             </div>
             <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-6xl mx-auto mb-4 flex justify-between items-center">
                   <h4 className="text-slate-800 font-medium">当前选中项目: <span className="font-bold text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-md ml-1">{selectedProjectName}</span></h4>
                   <button onClick={() => setIsAddingEnv(true)} disabled={isAddingEnv} className="px-4 py-2 bg-indigo-600 text-white rounded-lg font-medium shadow-sm hover:bg-indigo-700 disabled:opacity-50 transition">
                      + 新增环境配置
                   </button>
                </div>
                {envListLoading ? (
                   <p className="text-slate-500 text-sm py-4 text-center">正在加载数据...</p>
                ) : (
                   <div className="max-w-6xl mx-auto bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden">
                   <table className="w-full text-left text-sm text-slate-600">
                     <thead className="bg-slate-50 text-slate-500 border-b border-slate-200 text-xs uppercase tracking-wider sticky top-0 z-10">
                       <tr>
                          <th className="px-6 py-4 font-bold w-1/3">环境名称</th>
                          <th className="px-6 py-4 font-bold">环境请求地址 (URL)</th>
                          <th className="px-6 py-4 font-bold text-center w-40">操作</th>
                       </tr>
                     </thead>
                     <tbody className="divide-y divide-slate-100">
                        {/* Add New Row */}
                        {isAddingEnv && (
                        <tr className="bg-indigo-50/40">
                           <td className="px-6 py-4">
                              <input 
                                type="text" 
                                placeholder="如: UAT环境" 
                                value={newEnvName} 
                                onChange={e => setNewEnvName(e.target.value)} 
                                className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" 
                              />
                           </td>
                           <td className="px-6 py-4">
                              <input 
                                type="text" 
                                placeholder="如: http://192.168.x.x:9001" 
                                value={newEnvUrl} 
                                onChange={e => setNewEnvUrl(e.target.value)} 
                                className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" 
                              />
                           </td>
                           <td className="px-6 py-4 text-center flex items-center justify-center gap-3">
                              <button onClick={handleAddEnv} disabled={!newEnvUrl || !newEnvName} className="px-3 py-1.5 text-sm font-medium bg-indigo-600 text-white rounded-md shadow-sm hover:bg-indigo-700 disabled:opacity-50 transition w-full">保存</button>
                              <button onClick={() => { setIsAddingEnv(false); setNewEnvUrl(''); setNewEnvName(''); }} className="px-3 py-1.5 text-sm font-medium text-slate-500 hover:bg-slate-100 rounded-md transition whitespace-nowrap">取消</button>
                           </td>
                        </tr>
                        )}

                       {envList.map(env => (
                          <tr key={env.ID} className="hover:bg-slate-50/50 transition">
                             <td className="px-6 py-4">
                                {editingEnvId === env.ID ? (
                                    <input 
                                       type="text" 
                                       value={editEnvName} 
                                       onChange={e => setEditEnvName(e.target.value)} 
                                       className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all" 
                                     />
                                ) : (
                                   <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-slate-100 text-slate-800">
                                      {env.envName || '-'}
                                   </span>
                                )}
                             </td>
                             <td className="px-6 py-4 font-mono text-sm break-all">
                                {editingEnvId === env.ID ? (
                                    <input 
                                       type="text" 
                                       value={editEnvUrl} 
                                       onChange={e => setEditEnvUrl(e.target.value)} 
                                       className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all" 
                                     />
                                ) : (env.baseUrl || '-')}
                             </td>
                             <td className="px-6 py-4 text-center flex items-center justify-center gap-3">
                               {editingEnvId === env.ID ? (
                                  <>
                                     <button onClick={() => handleUpdateEnv(env.ID)} className="px-3 py-1.5 text-sm font-medium bg-indigo-50 text-indigo-700 hover:bg-indigo-100 rounded-md transition disabled:opacity-50" disabled={!editEnvUrl || !editEnvName}>保存</button>
                                     <button onClick={() => setEditingEnvId(null)} className="px-3 py-1.5 text-sm font-medium text-slate-500 hover:bg-slate-100 rounded-md transition">取消</button>
                                  </>
                               ) : (
                                  <>
                                     <button onClick={() => { setEditingEnvId(env.ID); setEditEnvUrl(env.baseUrl); setEditEnvName(env.envName || ''); }} className="px-3 py-1.5 text-sm font-medium text-indigo-600 hover:bg-indigo-50 rounded-md transition">编辑</button>
                                     <button onClick={() => handleDeleteEnv(env.ID)} className="px-3 py-1.5 text-sm font-medium text-rose-600 hover:bg-rose-50 rounded-md transition"><Trash2 size={16}/></button>
                                  </>
                               )}
                             </td>
                          </tr>
                       ))}
                       {envList.length === 0 && (
                          <tr><td colSpan={3} className="px-6 py-12 text-center text-slate-400">目前暂无环境记录，请在上方添加第一个环境配置</td></tr>
                       )}
                     </tbody>
                   </table>
                   </div>
                )}
             </div>
        </div>
      )}

      {/* User Management Modal */}
      {userListOpen && (
        <div className="fixed inset-0 z-50 flex flex-col bg-slate-50 animate-in fade-in zoom-in-95 duration-200">
             <div className="px-8 py-5 bg-white border-b border-slate-200 flex items-center justify-between shrink-0 shadow-sm">
                <h3 className="text-xl font-semibold flex items-center gap-2">
                   <Users size={24} className="text-indigo-600" /> 用户管理
                </h3>
                <button onClick={() => setUserListOpen(false)} className="text-slate-400 hover:text-slate-600 p-2 rounded-full hover:bg-slate-100 transition"><X size={24} /></button>
             </div>
             <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-6xl mx-auto mb-4 flex justify-between items-center">
                   <h4 className="text-slate-800 font-medium">当前选中项目: <span className="font-bold text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-md ml-1">{selectedProjectName}</span></h4>
                   <button onClick={() => { setIsAddingUser(true); setUserForm({ loginAccount: '', loginPassword: '', userNickname: '', roleCode: '', roleName: '', environment: '', requestHeader: '' }); loadEnvList(); }} disabled={isAddingUser} className="px-4 py-2 bg-indigo-600 text-white rounded-lg font-medium shadow-sm hover:bg-indigo-700 disabled:opacity-50 transition">
                      + 新增用户
                   </button>
                </div>
                {userListLoading ? (
                   <p className="text-slate-500 text-sm py-4 text-center">正在加载数据...</p>
                ) : (
                   <div className="max-w-6xl mx-auto bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden">
                   <table className="w-full text-left text-sm text-slate-600">
                     <thead className="bg-slate-50 text-slate-500 border-b border-slate-200 text-xs uppercase tracking-wider sticky top-0 z-10">
                       <tr>
                          <th className="px-6 py-4 font-bold">登陆账号</th>
                          <th className="px-6 py-4 font-bold">用户昵称</th>
                          <th className="px-6 py-4 font-bold">角色名称</th>
                          <th className="px-6 py-4 font-bold">所属环境</th>
                          <th className="px-6 py-4 font-bold text-center w-40">操作</th>
                       </tr>
                     </thead>
                     <tbody className="divide-y divide-slate-100">
                        {/* Add New User Row */}
                        {isAddingUser && (
                        <tr className="bg-indigo-50/40">
                           <td className="px-6 py-4">
                              <input type="text" placeholder="登陆账号 *" value={userForm.loginAccount} onChange={e => setUserForm({...userForm, loginAccount: e.target.value})} className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" />
                           </td>
                           <td className="px-6 py-4">
                              <input type="text" placeholder="用户昵称" value={userForm.userNickname} onChange={e => setUserForm({...userForm, userNickname: e.target.value})} className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" />
                           </td>
                           <td className="px-6 py-4">
                              <input type="text" placeholder="角色名称" value={userForm.roleName} onChange={e => setUserForm({...userForm, roleName: e.target.value})} className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" />
                           </td>
                           <td className="px-6 py-4">
                              <select value={userForm.environment} onChange={e => setUserForm({...userForm, environment: e.target.value})} className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all">
                                 <option value="">请选择环境</option>
                                 {envList.map(env => <option key={env.ID} value={env.envName}>{env.envName}</option>)}
                              </select>
                           </td>
                           <td className="px-6 py-4 text-center">
                              <div className="flex items-center justify-center gap-2 mb-2">
                                 <input type="password" placeholder="登陆密码 *" value={userForm.loginPassword} onChange={e => setUserForm({...userForm, loginPassword: e.target.value})} className="w-full px-3 py-2 border border-indigo-200 rounded-lg text-sm bg-white outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all" />
                              </div>
                              <div className="flex items-center justify-center gap-2">
                                 <button onClick={handleAddUser} disabled={!userForm.loginAccount || !userForm.loginPassword} className="px-3 py-1.5 text-sm font-medium bg-indigo-600 text-white rounded-md shadow-sm hover:bg-indigo-700 disabled:opacity-50 transition flex-1">保存</button>
                                 <button onClick={() => setIsAddingUser(false)} className="px-3 py-1.5 text-sm font-medium text-slate-500 hover:bg-slate-100 rounded-md transition whitespace-nowrap">取消</button>
                              </div>
                           </td>
                        </tr>
                        )}

                       {userList.map(user => (
                          <tr key={user.ID} className="hover:bg-slate-50/50 transition">
                             <td className="px-6 py-4">
                                {editingUserId === user.ID ? (
                                    <input type="text" value={editUserForm.loginAccount} onChange={e => setEditUserForm({...editUserForm, loginAccount: e.target.value})} className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all" />
                                ) : (
                                   <span className="font-medium text-slate-900">{user.loginAccount}</span>
                                )}
                             </td>
                             <td className="px-6 py-4">
                                {editingUserId === user.ID ? (
                                    <input type="text" value={editUserForm.userNickname} onChange={e => setEditUserForm({...editUserForm, userNickname: e.target.value})} className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all" />
                                ) : (user.userNickname || '-')}
                             </td>
                             <td className="px-6 py-4">
                                {editingUserId === user.ID ? (
                                    <input type="text" value={editUserForm.roleName} onChange={e => setEditUserForm({...editUserForm, roleName: e.target.value})} className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all" />
                                ) : (user.roleName || '-')}
                             </td>
                             <td className="px-6 py-4">
                                {editingUserId === user.ID ? (
                                    <select value={editUserForm.environment} onChange={e => setEditUserForm({...editUserForm, environment: e.target.value})} className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none transition-all">
                                       <option value="">请选择环境</option>
                                       {envList.map(env => <option key={env.ID} value={env.envName}>{env.envName}</option>)}
                                    </select>
                                ) : (
                                   <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-emerald-100 text-emerald-800">
                                      {user.environment || '-'}
                                   </span>
                                )}
                             </td>
                             <td className="px-6 py-4 text-center flex items-center justify-center gap-3">
                                {editingUserId === user.ID ? (
                                   <>
                                      <div className="flex items-center gap-1">
                                        <input type="password" placeholder="密码" value={editUserForm.loginPassword} onChange={e => setEditUserForm({...editUserForm, loginPassword: e.target.value})} className="w-24 px-2 py-1.5 border border-slate-300 rounded-lg text-sm focus:border-indigo-500 outline-none" />
                                      </div>
                                      <button onClick={() => handleUpdateUser(user.ID)} className="px-3 py-1.5 text-sm font-medium bg-indigo-600 text-white rounded-md shadow-sm hover:bg-indigo-700 transition">保存</button>
                                      <button onClick={() => setEditingUserId(null)} className="px-3 py-1.5 text-sm font-medium text-slate-500 hover:bg-slate-100 rounded-md transition">取消</button>
                                   </>
                                ) : (
                                   <>
                                      <button onClick={() => { setEditingUserId(user.ID); setEditUserForm({ loginAccount: user.loginAccount, loginPassword: user.loginPassword, userNickname: user.userNickname, roleCode: user.roleCode, roleName: user.roleName, environment: user.environment, requestHeader: user.requestHeader }); loadEnvList(); }} className="px-3 py-1.5 text-sm font-medium text-indigo-600 hover:bg-indigo-50 rounded-md transition">编辑</button>
                                      <button onClick={() => { setHeaderUserId(user.ID); setHeaderValue(user.requestHeader || ''); setHeaderModalOpen(true); }} className="px-3 py-1.5 text-sm font-medium text-emerald-600 hover:bg-emerald-50 rounded-md transition" title="编辑 Header"><FileJson size={16}/></button>
                                      <button onClick={() => handleDeleteUser(user.ID)} className="px-3 py-1.5 text-sm font-medium text-rose-600 hover:bg-rose-50 rounded-md transition"><Trash2 size={16}/></button>
                                   </>
                                )}
                             </td>
                          </tr>
                       ))}
                       {userList.length === 0 && (
                          <tr><td colSpan={5} className="px-6 py-12 text-center text-slate-400">目前暂无用户记录，请在上方添加第一个用户</td></tr>
                       )}
                     </tbody>
                   </table>
                   </div>
                )}
             </div>
        </div>
      )}

      {/* Header Setting Modal */}
      {headerModalOpen && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center bg-slate-900/30 backdrop-blur-sm animate-in fade-in duration-200">
           <div className="bg-white rounded-2xl shadow-xl w-[600px] flex flex-col animate-in zoom-in-95 duration-200">
              <div className="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                 <h3 className="text-lg font-semibold text-slate-800 flex items-center gap-2">
                    <FileJson className="text-indigo-600" size={20} /> 请求 Header (JSON格式)
                 </h3>
                 <button onClick={() => setHeaderModalOpen(false)} className="text-slate-400 hover:text-slate-600"><X size={20}/></button>
              </div>
              <div className="p-6">
                 <p className="text-sm text-slate-500 mb-3">直接将浏览器 F12 中复制的 Request Headers 或者 Raw JSON 粘贴在下方：</p>
                 <textarea 
                    value={headerValue}
                    onChange={e => setHeaderValue(e.target.value)}
                    className="w-full h-64 p-3 border border-slate-300 rounded-xl text-sm font-mono focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 outline-none resize-none"
                    placeholder={`{\n  "Authorization": "Bearer token...",\n  "Cookie": "session_id=123..."\n}`}
                    spellCheck={false}
                 />
              </div>
              <div className="px-6 py-4 border-t border-slate-100 bg-slate-50/50 flex justify-end gap-3 rounded-b-2xl">
                 <button onClick={() => setHeaderModalOpen(false)} className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-lg transition">取消</button>
                 <button onClick={handleSaveHeader} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm transition">保存</button>
              </div>
           </div>
        </div>
      )}

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
