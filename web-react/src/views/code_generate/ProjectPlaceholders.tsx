import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Plus, Trash2, Edit2, ArrowLeft, KeySquare } from 'lucide-react';
import toast from 'react-hot-toast';
import { motion, AnimatePresence } from 'framer-motion';
import { getProjectPlaceHolderList, createProjectPlaceHolder, updateProjectPlaceHolder, deleteProjectPlaceHolder } from '@/api/project_placeholder';

export default function ProjectPlaceholders() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const [placeholders, setPlaceholders] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  
  const [showModal, setShowModal] = useState(false);
  const [currentHolder, setCurrentHolder] = useState<any>({});

  const fetchPlaceholders = async () => {
    setLoading(true);
    try {
      const res = await getProjectPlaceHolderList();
      const allHolders = res.data || [];
      // Filter for this specific project
      const filtered = allHolders.filter((h: any) => h.projectId === parseInt(projectId!));
      setPlaceholders(filtered);
    } catch (e) {
      toast.error('获取项目占位符失败');
    }
    setLoading(false);
  };

  useEffect(() => {
    if (projectId) {
      fetchPlaceholders();
    }
  }, [projectId]);

  const handleSave = async () => {
    if (!currentHolder.holderKey || !currentHolder.holderValue) {
      toast.error('Key 和 Value 不能为空');
      return;
    }
    
    try {
      const payload = {
        ...currentHolder,
        projectId: parseInt(projectId!),
        userName: 'conchi' // default user
      };
      
      if (currentHolder.ID) {
        await updateProjectPlaceHolder(payload);
        toast.success('更新成功');
      } else {
        await createProjectPlaceHolder(payload);
        toast.success('创建成功');
      }
      setShowModal(false);
      fetchPlaceholders();
    } catch (e) {
      toast.error('保存失败');
    }
  };

  const handleDelete = async (data: any) => {
    if (confirm('确定删除该占位符吗？')) {
      try {
        await deleteProjectPlaceHolder(data);
        toast.success('删除成功');
        fetchPlaceholders();
      } catch (e) {
        toast.error('删除失败');
      }
    }
  };

  const displayHolders = placeholders.filter(p => !searchTerm || 
    p.holderKey.toLowerCase().includes(searchTerm.toLowerCase()) || 
    p.holderDesc.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="flex flex-col w-full h-[calc(100vh-140px)] border border-slate-200/60 rounded-3xl overflow-hidden bg-slate-50 shadow-xl mt-2 animate-in fade-in duration-300">
      
      {/* Top Banner */}
      <div className="bg-gradient-to-r from-slate-800 to-slate-900 text-white px-6 py-4 flex items-center justify-between shadow-md z-20">
         <div className="flex items-center gap-4">
             <button onClick={() => navigate('/projects')} className="p-2 hover:bg-white/10 rounded-xl transition-colors flex items-center gap-1 text-sm font-medium border border-slate-700">
                 <ArrowLeft size={16} /> 返回主版
             </button>
             <div className="h-6 w-px bg-slate-700"></div>
             <div>
                <h1 className="text-lg font-bold tracking-tight text-amber-400">项目占位符字典池 <span className="text-slate-300 font-normal">/ Project ID: {projectId}</span></h1>
             </div>
         </div>
         <button
            onClick={() => {
              setCurrentHolder({});
              setShowModal(true);
            }}
            className="flex items-center gap-2 bg-amber-500 hover:bg-amber-600 text-white px-4 py-2 rounded-xl shadow-lg transition-colors font-medium text-sm"
          >
            <Plus size={16} />
            <span>新增变量占位符</span>
          </button>
      </div>

      <div className="p-6 flex-1 overflow-auto">
         {/* Search Filter */}
         <div className="mb-6 flex gap-4">
            <div className="relative flex-1 max-w-md">
              <input 
                type="text" 
                placeholder="搜索占位符 Key 或者描述..." 
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="w-full pl-4 pr-10 py-2.5 bg-white border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-amber-500/50 text-sm shadow-sm"
              />
            </div>
         </div>

         {/* Table */}
         <div className="bg-white border border-slate-200 rounded-2xl shadow-sm overflow-hidden">
            <table className="w-full text-left border-collapse text-sm">
                <thead>
                    <tr className="bg-slate-50 border-b border-slate-200">
                        <th className="py-3 px-4 font-semibold text-slate-600">占位符 Key</th>
                        <th className="py-3 px-4 font-semibold text-slate-600">填充 Value</th>
                        <th className="py-3 px-4 font-semibold text-slate-600">占位符描述</th>
                        <th className="py-3 px-4 font-semibold text-slate-600 w-1/4">示例值</th>
                        <th className="py-3 px-4 font-semibold text-slate-600 text-right w-32">操作</th>
                    </tr>
                </thead>
                <tbody>
                    {loading ? (
                       <tr><td colSpan={5} className="py-8 text-center text-slate-400">拉取字典池...</td></tr>
                    ) : displayHolders.length === 0 ? (
                       <tr><td colSpan={5} className="py-8 text-center text-slate-400">当前项目尚未注册独立的宏定义占位符...</td></tr>
                    ) : (
                       displayHolders.map(p => (
                         <tr key={p.ID} className="border-b border-slate-100 hover:bg-slate-50/50 transition-colors">
                             <td className="py-3 px-4"><span className="bg-slate-100 text-slate-700 font-mono px-2 py-0.5 rounded text-xs">{p.holderKey}</span></td>
                             <td className="py-3 px-4 font-medium text-slate-800">{p.holderValue}</td>
                             <td className="py-3 px-4 text-slate-500">{p.holderDesc}</td>
                             <td className="py-3 px-4 text-slate-400 italic text-xs">{p.exampleValue}</td>
                             <td className="py-3 px-4 text-right">
                                <button onClick={() => { setCurrentHolder(p); setShowModal(true); }} className="p-1.5 text-slate-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors mr-2"><Edit2 size={15} /></button>
                                <button onClick={() => handleDelete(p)} className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"><Trash2 size={15} /></button>
                             </td>
                         </tr>
                       ))
                    )}
                </tbody>
            </table>
         </div>
      </div>

       {/* Form Modal */}
       <AnimatePresence>
        {showModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onClick={() => setShowModal(false)} />
            <motion.div initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }} className="relative w-full max-w-lg bg-white rounded-3xl shadow-2xl p-8">
              <div className="flex items-center gap-3 mb-6">
                 <div className="p-2.5 bg-amber-100 text-amber-600 rounded-xl"><KeySquare size={20} /></div>
                 <h2 className="text-xl font-bold text-slate-800">{currentHolder.ID ? '编辑节点占位符' : '注册新版占位符'}</h2>
              </div>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">宏变量 Key (引擎识别关键字)</label>
                  <input type="text" value={currentHolder.holderKey || ''} onChange={e => setCurrentHolder({ ...currentHolder, holderKey: e.target.value })} className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-amber-500/50 font-mono text-sm" placeholder="例如: {[<moduleName>]}" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">实际映射代码段 Value</label>
                  <input type="text" value={currentHolder.holderValue || ''} onChange={e => setCurrentHolder({ ...currentHolder, holderValue: e.target.value })} className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-amber-500/50 font-mono text-sm" placeholder="例如: UserService" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">说明备注 (Desc)</label>
                  <input type="text" value={currentHolder.holderDesc || ''} onChange={e => setCurrentHolder({ ...currentHolder, holderDesc: e.target.value })} className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-amber-500/50 text-sm" placeholder="简单的标识说明" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">渲染示例 (Example)</label>
                  <input type="text" value={currentHolder.exampleValue || ''} onChange={e => setCurrentHolder({ ...currentHolder, exampleValue: e.target.value })} className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-amber-500/50 text-sm" placeholder="用于提示用户的样例" />
                </div>
              </div>
              <div className="mt-8 flex justify-end gap-3">
                <button onClick={() => setShowModal(false)} className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-xl text-sm font-medium transition-colors">取消退出</button>
                <button onClick={handleSave} className="px-5 py-2.5 bg-amber-500 hover:bg-amber-600 text-white rounded-xl text-sm font-medium transition-colors shadow-lg shadow-amber-500/20">保存到注册表</button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
};
