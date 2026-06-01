import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Table, Trash2 } from 'lucide-react';
import { getTbTableList, deleteTbTable, TbTable } from '../../api/sysTable';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function TableManager() {
  const [tables, setTables] = useState<TbTable[]>([]);
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
      const res = await getTbTableList({ page: targetPage, pageSize: 20 });
      setTables(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载物理表数据失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(table: TbTable) {
    const confirmed = await confirm(`确定要删除物理表定义「${table.tableName}」吗？`, {
      title: '删除物理表',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbTable({ ID: table.ID });
      toast.success('物理表已删除');
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
              物理表管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理底层数据库中定义的物理表结构映射数据。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-blue-50 text-blue-900 border border-blue-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">物理表总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载物理表数据...
          </div>
        ) : tables.map((item) => (
          <article key={item.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <Table size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{item.tableName}</h3>
                  <p className="text-sm text-slate-500">{item.description}</p>
                </div>
              </div>
            </div>
            
            <div className="mt-6 grid gap-3 rounded-[20px] bg-slate-50 p-4 text-sm text-slate-600 sm:grid-cols-2">
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">数据库名</div>
                    <div className="mt-2 text-sm text-slate-800">{item.databaseName || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">数据源 ID</div>
                    <div className="mt-2 text-sm text-slate-800">No. {item.connectionId}</div>
                </div>
            </div>

            <div className="mt-5 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => handleDelete(item)}
                className="inline-flex items-center gap-2 rounded-2xl border border-rose-200 px-4 py-2 text-sm font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={16} /> 删除
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
