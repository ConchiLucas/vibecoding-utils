import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { AlignLeft, Trash2 } from 'lucide-react';
import { getTbColumnList, deleteTbColumn, TbColumn } from '../../api/sysColumn';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function EntityColumnManager() {
  const [columns, setColumns] = useState<TbColumn[]>([]);
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
      const res = await getTbColumnList({ page: targetPage, pageSize: 20 });
      setColumns(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载逻辑字段属性失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(col: TbColumn) {
    const confirmed = await confirm(`确定要清除逻辑字段「${col.columnName}」吗？`, {
      title: '解绑属性列',
      confirmText: '确定解绑',
    });
    if (!confirmed) return;

    try {
      await deleteTbColumn({ ID: col.ID });
      toast.success('逻辑属性由于删除');
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
              逻辑实体属性管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理每个应用实体包含的逻辑字段、类型校验规则及绑定格式要求。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-fuchsia-50 text-fuchsia-900 border border-fuchsia-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">注册属性总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
        {loading ? (
          <div className="col-span-full rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载逻辑字段池...
          </div>
        ) : columns.map((col) => (
          <article key={col.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
                  <AlignLeft size={20} />
                </div>
                <div className="min-w-0">
                  <h3 className="text-base font-semibold tracking-tight text-slate-950 truncate">{col.columnName}</h3>
                  <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-mono bg-fuchsia-100 text-fuchsia-700 px-1.5 py-0.5 rounded">{col.columnType}</span>
                      {col.required === 1 && <span className="text-[10px] text-rose-500 uppercase tracking-wider font-semibold">Required</span>}
                  </div>
                </div>
              </div>
            </div>
            
            <div className="mt-5 space-y-2 text-xs">
                {col.description && <div className="text-slate-500">{col.description}</div>}
                
                <div className="grid grid-cols-2 gap-2 mt-3 bg-slate-50 p-3 rounded-2xl border border-slate-100/50">
                    <div>
                        <span className="text-slate-400 mr-1">所属实体:</span>
                        <span className="font-semibold text-slate-700 truncate block">{col.entityName}</span>
                    </div>
                    {col.maxLength > 0 && (
                        <div>
                            <span className="text-slate-400 mr-1">最大限制:</span>
                            <span className="font-semibold text-slate-700">{col.maxLength}</span>
                        </div>
                    )}
                    {col.formatValue && (
                        <div className="col-span-2">
                             <span className="text-slate-400 mr-1">格式化正则:</span>
                             <span className="font-mono text-[10px] text-slate-600">{col.formatValue}</span>
                        </div>
                    )}
                     {col.enumValue && (
                        <div className="col-span-2">
                             <span className="text-slate-400 mr-1">可用枚举:</span>
                             <span className="font-mono text-[10px] text-slate-600">{col.enumValue}</span>
                        </div>
                    )}
                </div>
            </div>

            <div className="mt-5 pt-4 flex justify-between items-center border-t border-slate-50">
                <span className="text-[10px] text-slate-400">服务: {col.serverName || '-'}</span>
              <button
                type="button"
                onClick={() => handleDelete(col)}
                className="inline-flex items-center gap-1.5 rounded-xl border border-rose-200 px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={14} /> 移除字段
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
