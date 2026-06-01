import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Users, Trash2, PowerOff, Power, Plus, Edit2, Play } from 'lucide-react';
import { getTbClientList, deleteTbClient, updateTbClient, createTbClient, TbClient } from '../../api/sysClient';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function ClientManager() {
  const [clients, setClients] = useState<TbClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const [total, setTotal] = useState(0);
  const { confirm, dialogProps } = useConfirm();

  // Search form
  const [queryEnv, setQueryEnv] = useState('');
  const [queryName, setQueryName] = useState('');

  // Dialog State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingClient, setEditingClient] = useState<Partial<TbClient> | null>(null);

  useEffect(() => {
    void loadData(page);
  }, [page, queryEnv, queryName]);

  async function loadData(targetPage: number) {
    setLoading(true);
    try {
      const res = await getTbClientList({ 
        page: targetPage, 
        pageSize: 20,
        nickName: queryName || undefined,
        // environment: queryEnv || undefined // Assuming API supports this based on vue 
      });
      setClients(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载用户配置失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(client: TbClient) {
    const confirmed = await confirm(`确定要删除此用户吗？`, {
      title: '删除配置',
      confirmText: '确认删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbClient({ ID: client.ID });
      toast.success('删除成功');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('删除失败');
    }
  }

  async function handleBatchDelete() {
     toast('批量删除待集成');
  }

  async function toggleStatus(client: TbClient) {
      const newStatus = client.enableFlag === 1 ? 0 : 1;
      try {
        await updateTbClient({ ID: client.ID, enableFlag: newStatus });
        toast.success(newStatus === 1 ? '已启用' : '已停用');
        void loadData(page);
      } catch (error) {
        toast.error('状态切换失败');
        console.error(error);
      }
  }

  async function handleLogin(client: TbClient) {
      const confirmed = await confirm('确定进行立即执行登录操作?', { title: '提示', confirmText: '确定' });
      if (!confirmed) return;
      toast.success(`正在虚拟登录: ${client.loginName}...`);
      // TODO: Connect to clientLoginApi
  }

  const openAddModal = () => {
    setEditingClient({
       environment: '',
       loginName: '',
       password: '',
       nickName: '',
       identity: '',
       enableFlag: 1,
       remark: ''
    });
    setIsModalOpen(true);
  };

  const openEditModal = (client: TbClient) => {
    setEditingClient({ ...client });
    setIsModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingClient) return;
    
    try {
       if (editingClient.ID) {
           await updateTbClient(editingClient);
           toast.success('修改成功');
       } else {
           await createTbClient(editingClient);
           toast.success('新增成功');
       }
       setIsModalOpen(false);
       loadData(page);
    } catch(err) {
       toast.error('操作失败');
    }
  };

  return (
    <div className="space-y-6">
      
      {/* Search Bar mapped to Vue ElForm */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 flex gap-3 flex-wrap shadow-sm">
         <select 
             value={queryEnv} onChange={(e) => setQueryEnv(e.target.value)}
             className="px-3 py-2 border border-slate-200 rounded-lg text-sm bg-slate-50 outline-none focus:border-indigo-500"
         >
             <option value="">全部环境</option>
             <option value="local">Local</option>
             <option value="test">Test</option>
             <option value="prod">Prod</option>
         </select>
         <input 
             type="text" 
             placeholder="用户名称" 
             value={queryName} 
             onChange={(e) => setQueryName(e.target.value)}
             className="px-3 py-2 border border-slate-200 rounded-lg text-sm bg-slate-50 outline-none focus:border-indigo-500 w-64"
         />
         <button onClick={() => loadData(1)} className="px-5 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition">查询</button>
      </div>

      {/* Button Group mapped to Vue ElSpace */}
      <div className="flex gap-2">
         <button onClick={openAddModal} className="px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition inline-flex items-center gap-1"><Plus size={16}/> 新增</button>
         <button onClick={handleBatchDelete} className="px-4 py-2 bg-white border border-rose-200 text-rose-600 text-sm font-medium rounded-lg hover:bg-rose-50 transition inline-flex items-center gap-1"><Trash2 size={16}/> 批量删除</button>
      </div>

      {/* Grid view replacing ElTable for better modern aesthetic while keeping functional buttons */}
      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-2xl border border-dashed border-slate-300 py-12 text-center text-sm text-slate-500">
            加载中...
          </div>
        ) : clients.map((client) => (
          <article key={client.ID} className="group rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:shadow-md h-full flex flex-col">
            <div className="flex justify-between items-start mb-4">
                <div className="flex gap-3">
                    <div className="w-10 h-10 rounded-xl bg-indigo-50 text-indigo-600 flex items-center justify-center shrink-0"><Users size={20}/></div>
                    <div>
                        <h3 className="font-semibold text-slate-900">{client.nickName || '-'}</h3>
                        <div className="text-xs text-slate-500 mt-1">登录名: {client.loginName}</div>
                    </div>
                </div>
                <label className="flex items-center cursor-pointer">
                  <div className="relative">
                    <input type="checkbox" className="sr-only" checked={client.enableFlag === 1} onChange={() => toggleStatus(client)} />
                    <div className={`block w-10 h-6 rounded-full transition-colors ${client.enableFlag === 1 ? 'bg-indigo-500' : 'bg-slate-300'}`}></div>
                    <div className={`dot absolute left-1 top-1 bg-white w-4 h-4 rounded-full transition-transform ${client.enableFlag === 1 ? 'translate-x-4' : ''}`}></div>
                  </div>
                </label>
            </div>
            
            <div className="flex-1 text-sm space-y-2 mb-4 p-3 bg-slate-50 rounded-xl border border-slate-100">
               <div className="flex justify-between"><span className="text-slate-500">环境:</span><span className="font-mono text-xs">{client.environment || '-'}</span></div>
               <div className="flex justify-between"><span className="text-slate-500">身份:</span><span className="font-mono text-xs">{client.identity || '-'}</span></div>
               <div className="flex justify-between"><span className="text-slate-500">备注:</span><span className="truncate max-w-[200px]">{client.remark || '-'}</span></div>
            </div>

            <div className="flex justify-end gap-2 pt-3 border-t border-slate-100">
              {client.environment !== '0' && (
                 <button onClick={() => handleLogin(client)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-emerald-700 bg-emerald-50 hover:bg-emerald-100 rounded-lg transition"><Play size={14}/> 登录</button>
              )}
              <button onClick={() => openEditModal(client)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-indigo-700 bg-indigo-50 hover:bg-indigo-100 rounded-lg transition"><Edit2 size={14}/> 修改</button>
              <button onClick={() => handleDelete(client)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-rose-700 bg-rose-50 hover:bg-rose-100 rounded-lg transition"><Trash2 size={14}/> 删除</button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />

      {/* CRUD Modal */}
      {isModalOpen && editingClient && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm animate-in fade-in">
          <div className="bg-white rounded-2xl w-full max-w-lg shadow-xl overflow-hidden">
             <div className="p-6 border-b border-slate-100">
                <h3 className="text-lg font-semibold">{editingClient.ID ? '编辑用户配置' : '新增用户配置'}</h3>
             </div>
             <form onSubmit={handleSave} className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">环境</label>
                   <input required type="text" value={editingClient.environment || ''} onChange={e => setEditingClient({...editingClient, environment: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" placeholder="如: local" />
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">登录用户名</label>
                   <input required type="text" value={editingClient.loginName || ''} onChange={e => setEditingClient({...editingClient, loginName: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" />
                </div>
                <div className="grid grid-cols-2 gap-4">
                    <div>
                       <label className="block text-sm font-medium text-slate-700 mb-1">登录密码</label>
                       <input type="password" value={editingClient.password || ''} onChange={e => setEditingClient({...editingClient, password: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" />
                    </div>
                    <div>
                       <label className="block text-sm font-medium text-slate-700 mb-1">用户名称</label>
                       <input required type="text" value={editingClient.nickName || ''} onChange={e => setEditingClient({...editingClient, nickName: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" />
                    </div>
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">身份标记 (Identity)</label>
                   <input type="text" value={editingClient.identity || ''} onChange={e => setEditingClient({...editingClient, identity: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" />
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">状态</label>
                   <select value={editingClient.enableFlag} onChange={e => setEditingClient({...editingClient, enableFlag: Number(e.target.value)})} className="w-full px-3 py-2 border border-slate-200 rounded-lg">
                      <option value={1}>启用 (1)</option>
                      <option value={0}>停用 (0)</option>
                   </select>
                </div>
                <div>
                   <label className="block text-sm font-medium text-slate-700 mb-1">备注</label>
                   <textarea rows={2} value={editingClient.remark || ''} onChange={e => setEditingClient({...editingClient, remark: e.target.value})} className="w-full px-3 py-2 border border-slate-200 rounded-lg" />
                </div>
                
                <div className="pt-4 border-t border-slate-100 flex justify-end gap-2">
                   <button type="button" onClick={() => setIsModalOpen(false)} className="px-4 py-2 text-sm font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 rounded-lg">取消</button>
                   <button type="submit" className="px-6 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg shadow-sm">保存提交</button>
                </div>
             </form>
          </div>
        </div>
      )}
    </div>
  );
}
