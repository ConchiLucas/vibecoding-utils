import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Database, Trash2 } from 'lucide-react';
import { getDictDataList, deleteDictData, DictData } from '../../api/sysDict';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function DictManager() {
  const [dicts, setDicts] = useState<DictData[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const { confirm, dialogProps } = useConfirm();

  useEffect(() => {
    void loadData(page);
  }, [page]);

  async function loadData(targetPage: number) {
    setLoading(true);
    try {
      const res = await getDictDataList({ page: targetPage, pageSize: 20 });
      setDicts(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载字典数据失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(dict: DictData) {
    const confirmed = await confirm(`确定要删除字典「${dict.dictLabel}」吗？`, {
      title: '删除字典数据',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteDictData({ ID: dict.ID });
      toast.success('字典已删除');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('删除失败');
    }
  }

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[28px] border border-slate-200 bg-white px-6 py-8 shadow-sm">
        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
              字典管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理系统中的公共字典数据和映射关系。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-sky-50 text-sky-900 border border-sky-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">字典数据总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载字典数据...
          </div>
        ) : dicts.map((dict) => (
          <article key={dict.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <Database size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{dict.dictLabel}</h3>
                  <p className="text-sm font-mono text-slate-500">[{dict.dictType}]</p>
                </div>
              </div>
            </div>
            
            <div className="mt-6 grid gap-3 rounded-[20px] bg-slate-50 p-4 text-sm text-slate-600 sm:grid-cols-2">
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">键值 (Value)</div>
                    <div className="mt-2 font-mono font-medium text-slate-800 bg-slate-200 px-2 rounded">{dict.dictValue || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">样式类</div>
                    <div className="mt-2 font-medium text-slate-800">{dict.labelClass || '-'}</div>
                </div>
                <div className="col-span-2">
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">扩展参数</div>
                    <div className="mt-2 font-mono text-xs text-slate-500 break-all">{dict.extendParams || '无'}</div>
                </div>
            </div>

            <div className="mt-5 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => handleDelete(dict)}
                className="inline-flex items-center gap-2 rounded-2xl border border-rose-200 px-4 py-2 text-sm font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={16} />
                删除字典
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
