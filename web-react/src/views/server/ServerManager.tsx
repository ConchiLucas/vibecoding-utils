import React, { useEffect, useState } from 'react';
import { getServerPage, deleteServer, saveOrUpdateServer } from '../../api/server';
import toast from 'react-hot-toast';
import { Server, Settings2, Plus, Terminal, Hash, Globe, LogIn, Lock, Network, Bookmark, X, Trash2, Database } from 'lucide-react';
import clsx from 'clsx';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

type ConfigItem = {
  id: string;
  key: string;
  value: string;
};

type ConfigGroup = {
  id: string;
  name: string;
  items: ConfigItem[];
};

const createLocalId = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const parseExtendParams = (extendParams?: string): ConfigGroup[] => {
  if (!extendParams?.trim()) return [];
  try {
    const parsed = JSON.parse(extendParams);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return [];
    return Object.entries(parsed).map(([name, value]) => ({
      id: createLocalId(),
      name,
      items: value && typeof value === 'object' && !Array.isArray(value)
        ? Object.entries(value as Record<string, unknown>).map(([key, itemValue]) => ({
          id: createLocalId(),
          key,
          value: itemValue == null ? '' : String(itemValue)
        }))
        : []
    }));
  } catch {
    return [];
  }
};

const serializeConfigGroups = (groups: ConfigGroup[]) => {
  const payload: Record<string, Record<string, string>> = {};
  const seen = new Set<string>();

  for (const group of groups) {
    const groupName = group.name.trim();
    const filledItems = group.items
      .map(item => ({ key: item.key.trim(), value: item.value }))
      .filter(item => item.key || item.value);

    if (!groupName && filledItems.length === 0) continue;
    if (!groupName) return { error: '请填写配置组名称' };
    if (seen.has(groupName)) return { error: `配置组「${groupName}」重复` };
    seen.add(groupName);

    payload[groupName] = {};
    const seenItemKeys = new Set<string>();
    for (const item of filledItems) {
      if (!item.key) return { error: `请填写「${groupName}」中的配置项名称` };
      if (seenItemKeys.has(item.key)) return { error: `「${groupName}」中的配置项「${item.key}」重复` };
      seenItemKeys.add(item.key);
      payload[groupName][item.key] = item.value;
    }
  }

  return { value: Object.keys(payload).length ? JSON.stringify(payload) : '' };
};

export default function ServerManager() {
  const [servers, setServers] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const { confirm, dialogProps } = useConfirm();

  // Drawer / Form states
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  
  const [formData, setFormData] = useState({
    ID: 0,
    serverName: '',
    serverIp: '',
    serverInternalIp: '',
    serverLoginName: '',
    serverLoginPassword: '',
    serverLoginPort: 22,
    extendParams: '',
    remark: ''
  });
  const [configGroups, setConfigGroups] = useState<ConfigGroup[]>([]);
  const [activeConfigGroupId, setActiveConfigGroupId] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res: any = await getServerPage({ page: 1, pageSize: 50 });
      if (res.code === 0) {
        setServers(res.data?.list || []);
      }
    } catch (err) {
      toast.error('获取服务器列表异常');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const openDrawer = (server?: any) => {
    if (server) {
      setEditingId(server.ID);
      setFormData({
        ID: server.ID,
        serverName: server.serverName || '',
        serverIp: server.serverIp || '',
        serverInternalIp: server.serverInternalIp || '',
        serverLoginName: server.serverLoginName || '',
        serverLoginPassword: server.serverLoginPassword || '',
        serverLoginPort: server.serverLoginPort || 22,
        extendParams: server.extendParams || '',
        remark: server.remark || ''
      });
      const groups = parseExtendParams(server.extendParams);
      setConfigGroups(groups);
      setActiveConfigGroupId(groups[0]?.id || null);
    } else {
      setEditingId(null);
      setFormData({
        ID: 0, serverName: '', serverIp: '', serverInternalIp: '', serverLoginName: '', serverLoginPassword: '', serverLoginPort: 22, extendParams: '', remark: ''
      });
      setConfigGroups([]);
      setActiveConfigGroupId(null);
    }
    setDrawerOpen(true);
  };

  const closeDrawer = () => {
    setDrawerOpen(false);
  };

  const handleDelete = async (id: number) => {
    const ok = await confirm('确定将该服务器从集群中移除吗？');
    if (!ok) return;
    try {
      const res: any = await deleteServer(id);
      if (res.code === 0) {
        toast.success('节点已移除');
        setDrawerOpen(false);
        fetchData();
      } else {
        toast.error(res.msg || '移除失败');
      }
    } catch (err) {
      toast.error('网络请求异常');
    }
  };

  const handleFormChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(p => ({ ...p, [name]: name === 'serverLoginPort' ? parseInt(value) || 22 : value }));
  };

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.serverName || !formData.serverIp || !formData.serverLoginName) {
      toast.error('请填写完整的必填项信息');
      return;
    }
    
    setSubmitLoading(true);
    try {
      const serializedConfigs = serializeConfigGroups(configGroups);
      if (serializedConfigs.error) {
        toast.error(serializedConfigs.error);
        setSubmitLoading(false);
        return;
      }
      const payload = { ...formData, extendParams: serializedConfigs.value || '' };
      const res: any = await saveOrUpdateServer(payload);
      if (res.code === 0) {
        toast.success(editingId ? '服务器已更新配置' : '集群中已添加此服务器');
        closeDrawer();
        fetchData();
      } else {
        toast.error(res.msg || '节点操作失败');
      }
    } catch (err) {
      toast.error('提交异常');
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleAddConfigGroup = () => {
    const group: ConfigGroup = {
      id: createLocalId(),
      name: `组件配置${configGroups.length + 1}`,
      items: [{ id: createLocalId(), key: '', value: '' }]
    };
    setConfigGroups(prev => [...prev, group]);
    setActiveConfigGroupId(group.id);
  };

  const handleDeleteConfigGroup = (groupId: string) => {
    setConfigGroups(prev => {
      const next = prev.filter(group => group.id !== groupId);
      if (activeConfigGroupId === groupId) {
        setActiveConfigGroupId(next[0]?.id || null);
      }
      return next;
    });
  };

  const handleUpdateConfigGroupName = (groupId: string, name: string) => {
    setConfigGroups(prev => prev.map(group => group.id === groupId ? { ...group, name } : group));
  };

  const handleAddConfigItem = (groupId: string) => {
    setConfigGroups(prev => prev.map(group => group.id === groupId ? {
      ...group,
      items: [...group.items, { id: createLocalId(), key: '', value: '' }]
    } : group));
  };

  const handleUpdateConfigItem = (groupId: string, itemId: string, field: 'key' | 'value', value: string) => {
    setConfigGroups(prev => prev.map(group => group.id === groupId ? {
      ...group,
      items: group.items.map(item => item.id === itemId ? { ...item, [field]: value } : item)
    } : group));
  };

  const handleDeleteConfigItem = (groupId: string, itemId: string) => {
    setConfigGroups(prev => prev.map(group => group.id === groupId ? {
      ...group,
      items: group.items.filter(item => item.id !== itemId)
    } : group));
  };

  const activeConfigGroup = configGroups.find(group => group.id === activeConfigGroupId) || null;

  return (
    <div className="w-full space-y-6 relative">
      
      {/* Top Controls */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
           <h1 className="text-3xl font-bold tracking-tight text-gray-900">服务器节点</h1>
           <p className="text-gray-500 mt-1">统一管理所有远程部署节点环境</p>
        </div>
        <div className="flex items-center gap-3 w-full sm:w-auto">
          <button 
            onClick={() => openDrawer()}
            className="flex-1 sm:flex-none border border-gray-200 bg-white hover:bg-gray-50 text-gray-800 font-medium py-2 px-4 rounded-lg shadow-sm transition-colors flex items-center justify-center gap-2"
          >
            <Plus size={16} /> 添加节点
          </button>
        </div>
      </div>

      {loading ? (
        <div className="h-40 flex items-center justify-center text-gray-400">资源加载中...</div>
      ) : servers.length === 0 ? (
        <div className="border border-dashed border-gray-300 rounded-xl p-12 text-center bg-gray-50">
           <h3 className="text-lg font-medium text-gray-900 mb-2">还没有添加任何节点</h3>
           <p className="text-gray-500 mb-6">添加一个新的服务器节点来部署你的应用。</p>
           <button onClick={() => openDrawer()} className="bg-black hover:bg-gray-800 text-white font-medium py-2 px-6 rounded-lg transition-colors">
              添加节点
           </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {servers.map((server) => (
            <div 
              key={server.ID} 
              className="group bg-white rounded-xl shadow-sm border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200 overflow-hidden flex flex-col"
            >
              <div className="p-6 cursor-pointer" onClick={() => openDrawer(server)}>
                <div className="flex justify-between items-start mb-4">
                  <div className="w-10 h-10 rounded-full bg-blue-50 text-blue-600 flex items-center justify-center border border-blue-100">
                    <Server size={20} />
                  </div>
                  <div className="flex items-center gap-1.5 text-xs font-mono bg-emerald-50 text-emerald-600 px-2 py-1 rounded-md">
                     <div className="w-1.5 h-1.5 rounded-full bg-emerald-500"></div> 已连接
                  </div>
                </div>

                <h3 className="text-lg font-bold text-gray-900 truncate mb-1">{server.serverName}</h3>
                <p className="text-sm font-medium text-gray-500 mb-4">{server.serverIp} <span className="text-gray-300 mx-1">|</span> port {server.serverLoginPort}</p>

                <div className="text-xs text-gray-500 truncate flex items-center gap-2 bg-gray-50 p-2 rounded border border-gray-100 mb-2">
                   <Terminal className="opacity-50" size={14} /> 
                   <span className="font-mono truncate">{server.serverLoginName}@...</span>
                </div>
                {server.remark && (
                   <p className="text-xs text-gray-400 mt-2 truncate">{server.remark}</p>
                )}
              </div>

              <div className="mt-auto border-t border-gray-100 bg-gray-50 p-3 flex justify-end gap-3">
                 <button 
                   onClick={(e) => { e.stopPropagation(); openDrawer(server); }}
                   className="w-full flex items-center justify-center py-2 px-4 border border-gray-200 bg-white hover:bg-gray-100 text-gray-700 rounded-lg transition-colors gap-2 text-sm font-medium shadow-sm"
                 >
                   <Settings2 size={16} /> 节点配置
                 </button>
              </div>

            </div>
          ))}
        </div>
      )}

      {/* Slide-over Form Drawer */}
      {drawerOpen && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-black/20 backdrop-blur-sm" onClick={closeDrawer}></div>
          <div className="relative w-full max-w-3xl bg-white h-full shadow-2xl flex flex-col animate-in slide-in-from-right-8 duration-300">
             
             <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
               <h2 className="text-lg font-bold text-gray-900">{editingId ? '编辑服务器配置' : '关联新服务器'}</h2>
               <button onClick={closeDrawer} className="p-2 text-gray-400 hover:text-gray-900 rounded-full hover:bg-gray-100 transition-colors">
                 <X size={20} />
               </button>
             </div>

             <form onSubmit={handleFormSubmit} className="flex-1 overflow-y-auto px-6 py-6 space-y-5">
               <div>
                 <label className="block text-sm font-medium text-gray-700 mb-1">节点名称 <span className="text-red-500">*</span></label>
                 <input type="text" name="serverName" required value={formData.serverName} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5" placeholder="例如: 华南主节点-01" />
               </div>

               <div className="grid grid-cols-2 gap-4">
                 <div className="col-span-2">
                   <label className="block text-sm justify-between flex font-medium text-gray-700 mb-1">
                     <span className="flex items-center gap-1"><Globe size={14}/> 公网 IP <span className="text-red-500">*</span></span>
                   </label>
                   <input type="text" name="serverIp" required value={formData.serverIp} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="47.110.12.9" />
                 </div>
                 <div className="col-span-2">
                   <label className="block text-sm justify-between flex font-medium text-gray-700 mb-1">
                     <span className="flex items-center gap-1"><Network size={14}/> 内网 IP </span>
                   </label>
                   <input type="text" name="serverInternalIp" value={formData.serverInternalIp} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="172.16.x.x (选填)" />
                 </div>
               </div>

               <div className="grid grid-cols-2 gap-4 pt-2 border-t border-gray-50">
                 <div className="col-span-1">
                   <label className="block text-sm font-medium text-gray-700 mb-1 flex items-center gap-1"><LogIn size={14} /> 登录账户 <span className="text-red-500">*</span></label>
                   <input type="text" name="serverLoginName" required value={formData.serverLoginName} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="root" />
                 </div>
                 <div className="col-span-1">
                   <label className="block text-sm font-medium text-gray-700 mb-1 flex items-center gap-1"><Hash size={14}/> SSH端口 <span className="text-red-500">*</span></label>
                   <input type="number" name="serverLoginPort" required value={formData.serverLoginPort} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 font-mono" placeholder="22" min="1" max="65535" />
                 </div>
               </div>

               <div>
                 <label className="block text-sm font-medium text-gray-700 mb-1 flex items-center gap-1"><Lock size={14} /> 密码口令</label>
                 <input type="password" name="serverLoginPassword" value={formData.serverLoginPassword} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5 bg-gray-50/50" placeholder="SSH认证密码(建议后期改用秘钥)" autoComplete="off" />
               </div>

               <div>
                 <label className="block text-sm font-medium text-gray-700 mb-1 flex items-center gap-1"><Bookmark size={14} /> 节点描述标签</label>
                 <textarea name="remark" rows={3} value={formData.remark} onChange={handleFormChange} className="w-full border border-gray-300 rounded-lg p-2.5 outline-none text-sm focus:ring-2 focus:ring-black/5" placeholder="此节点主要用于处理...业务" />
               </div>

               <div className="border-t border-gray-100 pt-5">
                 <div className="mb-3 flex items-center justify-between gap-3">
                   <label className="flex items-center gap-2 text-sm font-semibold text-gray-800">
                     <Database size={15} /> 组件配置
                   </label>
                   <button
                     type="button"
                     onClick={handleAddConfigGroup}
                     className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-gray-300 bg-white px-3 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50"
                   >
                     <Plus size={13} /> 添加配置组
                   </button>
                 </div>

                 <div className="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)]">
                   <div className="space-y-2 rounded-lg border border-gray-200 bg-gray-50 p-2">
                     {configGroups.length === 0 ? (
                       <div className="px-3 py-8 text-center text-xs text-gray-400">暂无组件配置</div>
                     ) : configGroups.map(group => (
                       <button
                         key={group.id}
                         type="button"
                         onClick={() => setActiveConfigGroupId(group.id)}
                         className={clsx(
                           'group flex w-full items-center justify-between gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors',
                           activeConfigGroupId === group.id
                             ? 'bg-gray-900 text-white shadow-sm'
                             : 'bg-white text-gray-700 hover:bg-gray-100'
                         )}
                       >
                         <span className="truncate font-medium">{group.name || '未命名配置'}</span>
                         <span className={clsx(
                           'rounded-full px-1.5 py-0.5 text-[10px]',
                           activeConfigGroupId === group.id ? 'bg-white/15 text-white' : 'bg-gray-100 text-gray-500'
                         )}>
                           {group.items.length}
                         </span>
                       </button>
                     ))}
                   </div>

                   <div className="min-h-[220px] rounded-lg border border-gray-200 bg-white p-4">
                     {activeConfigGroup ? (
                       <div className="space-y-4">
                         <div className="flex items-end gap-3">
                           <div className="flex-1">
                             <label className="mb-1 block text-xs font-medium text-gray-500">配置组名称</label>
                             <input
                               type="text"
                               value={activeConfigGroup.name}
                               onChange={(e) => handleUpdateConfigGroupName(activeConfigGroup.id, e.target.value)}
                               className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                               placeholder="mysql配置"
                             />
                           </div>
                           <button
                             type="button"
                             onClick={() => handleDeleteConfigGroup(activeConfigGroup.id)}
                             className="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-red-100 text-red-500 transition-colors hover:bg-red-50"
                             title="删除配置组"
                           >
                             <Trash2 size={15} />
                           </button>
                         </div>

                         <div className="space-y-2">
                           <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_36px] gap-2 px-1 text-xs font-medium text-gray-500">
                             <span>名称</span>
                             <span>值</span>
                             <span></span>
                           </div>
                           {activeConfigGroup.items.map(item => (
                             <div key={item.id} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_36px] gap-2">
                               <input
                                 type="text"
                                 value={item.key}
                                 onChange={(e) => handleUpdateConfigItem(activeConfigGroup.id, item.id, 'key', e.target.value)}
                                 className="min-w-0 rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                                 placeholder="redis账号"
                               />
                               <input
                                 type="text"
                                 value={item.value}
                                 onChange={(e) => handleUpdateConfigItem(activeConfigGroup.id, item.id, 'value', e.target.value)}
                                 className="min-w-0 rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-black/5"
                                 placeholder="redis账号内容"
                               />
                               <button
                                 type="button"
                                 onClick={() => handleDeleteConfigItem(activeConfigGroup.id, item.id)}
                                 className="inline-flex h-10 w-9 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500"
                                 title="删除配置项"
                               >
                                 <X size={15} />
                               </button>
                             </div>
                           ))}
                         </div>

                         <button
                           type="button"
                           onClick={() => handleAddConfigItem(activeConfigGroup.id)}
                           className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 text-xs font-medium text-gray-600 transition-colors hover:border-gray-800 hover:text-gray-900"
                         >
                           <Plus size={13} /> 新增配置项
                         </button>
                       </div>
                     ) : (
                       <div className="flex h-full min-h-[188px] items-center justify-center text-sm text-gray-400">
                         点击添加配置组
                       </div>
                     )}
                   </div>
                 </div>
               </div>

             </form>
             
             <div className="border-t border-gray-100 p-6 flex items-center justify-between bg-gray-50">
               {editingId ? (
                 <button type="button" onClick={() => handleDelete(editingId)} className="text-red-500 hover:bg-red-50 p-2 rounded-lg transition-colors flex items-center gap-1 text-sm font-medium">
                   <Trash2 size={16} /> 移除节点
                 </button>
               ) : <div></div>}
               <div className="flex gap-3">
                 <button type="button" onClick={closeDrawer} className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-100 transition-colors">
                   取消
                 </button>
                 <button type="submit" onClick={handleFormSubmit} disabled={submitLoading} className="px-4 py-2 bg-black text-white rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition-colors shadow-sm">
                   {submitLoading ? '保存中...' : (editingId ? '保存配置' : '确认添加')}
                 </button>
               </div>
             </div>

          </div>
        </div>
      )}
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
