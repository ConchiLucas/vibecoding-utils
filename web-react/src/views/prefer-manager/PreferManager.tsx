import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Settings, Trash2 } from 'lucide-react';
import { getTbTablePreferList, deleteTbTablePrefer, TbTablePrefer } from '../../api/sysPrefer';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function PreferManager() {
  const [prefers, setPrefers] = useState<TbTablePrefer[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const [total, setTotal] = useState(0);
  const { confirm, dialogProps } = useConfirm();

  useEffect(() => {
    void loadData(page);
  }, [page]);

  async function loadData(targetPage: number) {
    setLoading(true);
    try {
      const res = await getTbTablePreferList({ page: targetPage, pageSize: 20 });
      setPrefers(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载系统展示偏好失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(prefer: TbTablePrefer) {
    const confirmed = await confirm(`确定要重置该展示偏好组吗？这是一次破坏性操作。`, {
      title: '还原偏好设置',
      confirmText: '确定重置',
    });
    if (!confirmed) return;

    try {
      await deleteTbTablePrefer({ ID: prefer.ID });
      toast.success('偏好项已移除');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('移除失败');
    }
  }

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[28px] border border-slate-200 bg-white px-6 py-8 shadow-sm">
        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
              个性化引擎视效参数
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理用户的 UI 过滤条件、表格可见性状态序列以及查询参数持久化记录。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-indigo-50 text-indigo-900 border border-indigo-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">偏好锚点数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-3 lg:grid-cols-3">
        {loading ? (
          <div className="col-span-3 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在拉取偏好配置图谱...
          </div>
        ) : prefers.map((prefer) => (
          <article key={prefer.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-indigo-300">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-50 text-slate-400 group-hover:bg-indigo-50 group-hover:text-indigo-500 transition-colors">
                  <Settings size={20} className="group-hover:animate-spin-slow" />
                </div>
                <div className="min-w-0">
                  <h3 className="text-sm font-semibold tracking-tight text-slate-800 truncate">@{prefer.userName || '全局设置'}</h3>
                  <div className="text-[11px] text-slate-400 mt-0.5">{prefer.databaseName}.{prefer.tableName}</div>
                </div>
              </div>
            </div>
            
            <div className="mt-5 space-y-2 text-xs">
                <div className="bg-slate-50 p-3 rounded-2xl border border-slate-100/50">
                    <span className="text-slate-400 block mb-1">持久化渲染结构(Value):</span>
                    <span className="font-mono text-[10px] text-indigo-900 break-words line-clamp-3 leading-loose">{prefer.columnValue || '{}'}</span>
                </div>
            </div>

            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => handleDelete(prefer)}
                className="inline-flex items-center justify-center p-2 rounded-xl text-rose-500 transition hover:bg-rose-50"
              >
                <Trash2 size={16} />
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
