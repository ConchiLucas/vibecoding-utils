import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, Trash2, Edit2, Code2, X } from 'lucide-react';
import { getPlaceholderList, createPlaceholder, updatePlaceholder, deletePlaceholder } from '@/api/placeholder';
import toast from 'react-hot-toast';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const PlaceholderModal: React.FC<Props> = ({ isOpen, onClose }) => {
  const [placeholders, setPlaceholders] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showItemModal, setShowItemModal] = useState(false);
  const [currentItem, setCurrentItem] = useState<any>({});

  const fetchItems = async () => {
    if (!isOpen) return;
    setLoading(true);
    try {
      const res = await getPlaceholderList();
      setPlaceholders(res.data || []);
    } catch (e) {
      toast.error('获取占位符列表失败');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchItems();
  }, [isOpen]);

  const handleSave = async () => {
    try {
      if (currentItem.ID) {
        await updatePlaceholder(currentItem);
        toast.success('更新成功');
      } else {
        await createPlaceholder({ ...currentItem, userName: 'conchi' });
        toast.success('创建成功');
      }
      setShowItemModal(false);
      fetchItems();
    } catch (e) {
      toast.error('保存失败');
    }
  };

  const handleDelete = async (data: any) => {
    if (confirm('确定删除该全局占位符吗？')) {
      try {
        await deletePlaceholder(data);
        toast.success('删除成功');
        fetchItems();
      } catch (e) {
        toast.error('删除失败');
      }
    }
  };

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
          <motion.div
            initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
            className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm"
            onClick={() => !showItemModal && onClose()}
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }}
            className="relative w-full max-w-4xl bg-white rounded-3xl shadow-2xl p-8 max-h-[85vh] overflow-y-auto"
          >
            <div className="flex justify-between items-center mb-6">
              <div>
                <h1 className="text-2xl font-extrabold text-slate-800 tracking-tight">全局占位符管理</h1>
                <p className="text-slate-500 mt-1 text-sm">定制你在模板中可以使用的基础变量名</p>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={() => {
                    setCurrentItem({});
                    setShowItemModal(true);
                  }}
                  className="flex items-center gap-2 bg-gradient-to-r from-violet-500 to-fuchsia-500 hover:from-violet-600 hover:to-fuchsia-600 text-white px-4 py-2 rounded-xl shadow-lg shadow-violet-500/20 transition-all hover:scale-105 active:scale-95 text-sm"
                >
                  <Plus size={16} />
                  <span>新建占位符</span>
                </button>
                <button
                  onClick={onClose}
                  className="p-2 text-slate-400 hover:bg-slate-100 rounded-xl transition-colors"
                >
                  <X size={24} />
                </button>
              </div>
            </div>

            {/* List */}
            <div className="bg-white rounded-2xl shadow-sm border border-slate-100 overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="bg-slate-50/50">
                      <th className="py-3 px-5 font-medium text-slate-500 border-b border-slate-100 text-sm">标识符 (Key)</th>
                      <th className="py-3 px-5 font-medium text-slate-500 border-b border-slate-100 text-sm">替换值 (Value)</th>
                      <th className="py-3 px-5 font-medium text-slate-500 border-b border-slate-100 text-sm">描述</th>
                      <th className="py-3 px-5 font-medium text-slate-500 border-b border-slate-100 text-sm">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {loading ? (
                      <tr>
                        <td colSpan={4} className="py-12 text-center text-slate-400">加载中...</td>
                      </tr>
                    ) : placeholders.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="py-12 text-center text-slate-400">暂无全局占位符记录</td>
                      </tr>
                    ) : (
                      placeholders.map((item) => (
                        <tr key={item.ID} className="hover:bg-slate-50 border-b border-slate-50 last:border-0 transition-colors">
                          <td className="py-3 px-5">
                            <div className="inline-flex items-center gap-2 px-3 py-1 bg-violet-50 text-violet-700 rounded-lg font-mono text-sm">
                              <Code2 size={14} />
                              {item.holderKey}
                            </div>
                          </td>
                          <td className="py-3 px-5 text-slate-700 font-mono text-sm">{item.holderValue || '-'}</td>
                          <td className="py-3 px-5 text-slate-500 text-sm max-w-[200px] truncate">{item.holderDesc || '-'}</td>
                          <td className="py-3 px-5">
                            <div className="flex gap-2">
                              <button onClick={() => { setCurrentItem(item); setShowItemModal(true); }} className="p-1.5 text-slate-400 hover:text-violet-600 hover:bg-violet-50 rounded-md transition-colors">
                                <Edit2 size={16} />
                              </button>
                              <button onClick={() => handleDelete(item)} className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors">
                                <Trash2 size={16} />
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Inner Form Modal */}
            <AnimatePresence>
              {showItemModal && (
                <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
                  <motion.div
                    initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                    className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm"
                    onClick={() => setShowItemModal(false)}
                  />
                  <motion.div
                    initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }}
                    className="relative w-full max-w-md bg-white rounded-3xl shadow-2xl p-6"
                  >
                    <h2 className="text-xl font-bold text-slate-800 mb-5">{currentItem.ID ? '编辑占位符' : '添加占位符'}</h2>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">标识 (Key)</label>
                        <input
                          type="text"
                          value={currentItem.holderKey || ''}
                          onChange={e => setCurrentItem({ ...currentItem, holderKey: e.target.value })}
                          className="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-violet-500/50 text-sm"
                          placeholder="{[<Author>]}"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">替换结果 (Value)</label>
                        <input
                          type="text"
                          value={currentItem.holderValue || ''}
                          onChange={e => setCurrentItem({ ...currentItem, holderValue: e.target.value })}
                          className="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-violet-500/50 text-sm"
                          placeholder="渲染的具体内容"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">描述 (Description)</label>
                        <input
                          type="text"
                          value={currentItem.holderDesc || ''}
                          onChange={e => setCurrentItem({ ...currentItem, holderDesc: e.target.value })}
                          className="w-full px-3 py-2.5 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-violet-500/50 text-sm"
                          placeholder="选填"
                        />
                      </div>
                    </div>
                    <div className="mt-6 flex justify-end gap-2">
                      <button
                        onClick={() => setShowItemModal(false)}
                        className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-xl transition-colors font-medium"
                      >
                        取消
                      </button>
                      <button
                        onClick={handleSave}
                        className="px-4 py-2 text-sm bg-slate-800 hover:bg-slate-900 text-white rounded-xl transition-colors font-medium"
                      >
                        保存
                      </button>
                    </div>
                  </motion.div>
                </div>
              )}
            </AnimatePresence>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};
