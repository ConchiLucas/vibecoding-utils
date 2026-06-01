import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Columns3, Trash2 } from 'lucide-react';
import { getTbTableColumnList, deleteTbTableColumn, TbTableColumn } from '../../api/sysTableColumn';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function TableColumnManager() {
  const [columns, setColumns] = useState<TbTableColumn[]>([]);
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
      const res = await getTbTableColumnList({ page: targetPage, pageSize: 20 });
      setColumns(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载物理字段数据失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(col: TbTableColumn) {
    const confirmed = await confirm(`确定要删除表字段「${col.columnName}」吗？`, {
      title: '删除表字段',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbTableColumn({ ID: col.ID });
      toast.success('表字段已删除');
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
              物理表字段管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理物理表的字面列属性、类型、默认值。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-sky-50 text-sky-900 border border-sky-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">全量字段数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-3">
        {loading ? (
          <div className="col-span-3 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载物理字段数据...
          </div>
        ) : columns.map((col) => (
          <article key={col.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
                  <Columns3 size={20} />
                </div>
                <div>
                  <h3 className="text-base font-semibold tracking-tight text-slate-950">{col.columnName}</h3>
                  <p className="text-xs font-mono text-slate-500 bg-slate-100 px-1.5 py-0.5 rounded inline-block mt-1">{col.columnType}({col.columnSize})</p>
                </div>
              </div>
            </div>
            
            <div className="mt-4 grid gap-2 rounded-[16px] bg-slate-50 p-3 text-xs text-slate-600 grid-cols-2">
                <div>
                    <span className="text-slate-400">是否允许为空：</span>
                    <span className="font-semibold text-slate-800">{col.isEmpty === 1 ? '是' : '否'}</span>
                </div>
                <div>
                    <span className="text-slate-400">默认值：</span>
                    <span className="font-semibold text-slate-800">{col.defaultValue || '—'}</span>
                </div>
                <div className="col-span-2 mt-1 border-t border-slate-200/60 pt-2">
                    <span className="text-slate-400">关联表 ID：</span>
                    <span className="font-medium text-slate-800">{col.tableId || '-'}</span>
                </div>
            </div>

            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => handleDelete(col)}
                className="inline-flex items-center gap-1.5 rounded-xl border border-rose-200 px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={14} /> 移除
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
