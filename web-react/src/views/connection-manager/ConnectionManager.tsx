import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Cable, Trash2, Plug, DatabaseZap } from 'lucide-react';
import { getTbConnectionList, deleteTbConnection, testConnection, initConnection, TbConnection } from '../../api/sysConnection';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function ConnectionManager() {
  const [connections, setConnections] = useState<TbConnection[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [opLoading, setOpLoading] = useState<Record<number, string>>({});
  const { confirm, dialogProps } = useConfirm();

  useEffect(() => {
    void loadData(page);
  }, [page]);

  async function loadData(targetPage: number) {
    setLoading(true);
    try {
      const res = await getTbConnectionList({ page: targetPage, pageSize: 20 });
      setConnections(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载连接数据失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(conn: TbConnection) {
    const confirmed = await confirm(`确定要删除数据源连接「${conn.connectionName}」吗？`, {
      title: '删除数据源连接',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbConnection({ ID: conn.ID });
      toast.success('数据源连接已删除');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('删除失败');
    }
  }

  async function handleTest(conn: TbConnection) {
    setOpLoading(prev => ({ ...prev, [conn.ID]: 'test' }));
    try {
      await testConnection({ ID: conn.ID });
      toast.success(`「${conn.connectionName}」连接成功 ✓`);
    } catch (err: any) {
      toast.error(err?.response?.data?.msg || '连接失败');
    } finally {
      setOpLoading(prev => { const n = { ...prev }; delete n[conn.ID]; return n; });
    }
  }

  async function handleInit(conn: TbConnection) {
    const confirmed = await confirm(
      `确定要导入「${conn.connectionName}」的表结构吗？\n这将清空并重新导入所有表和字段信息。`,
      { title: '导入表结构', confirmText: '确定导入' }
    );
    if (!confirmed) return;
    setOpLoading(prev => ({ ...prev, [conn.ID]: 'init' }));
    try {
      await initConnection({ ID: conn.ID });
      toast.success('表结构导入成功，已更新所有表和字段数据');
    } catch (err: any) {
      toast.error(err?.response?.data?.msg || '导入失败');
    } finally {
      setOpLoading(prev => { const n = { ...prev }; delete n[conn.ID]; return n; });
    }
  }

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[28px] border border-slate-200 bg-white px-6 py-8 shadow-sm">
        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
              数据库连接管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理系统依赖的所有 MySQL、PostgreSQL、Oracle、SQL Server、SQLite 与 ClickHouse 外部数据库连接和数据源。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-orange-50 text-orange-900 border border-orange-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">连接总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载数据源明细...
          </div>
        ) : connections.map((conn) => (
          <article key={conn.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <Cable size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{conn.connectionName}</h3>
                  <p className="text-sm font-mono text-slate-500">[{conn.connectionType}]</p>
                </div>
              </div>
            </div>
            
            <div className="mt-6 grid gap-3 rounded-[20px] bg-slate-50 p-4 text-sm text-slate-600 sm:grid-cols-2">
                <div className="col-span-2">
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">连接地址</div>
                    <div className="mt-2 font-mono font-medium text-slate-800 bg-slate-200 px-2 rounded truncate" title={conn.connectionUrl}>{conn.connectionUrl || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">数据库名</div>
                    <div className="mt-2 text-sm text-slate-800">{conn.databaseName || '-'}</div>
                </div>
                 <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">端口号</div>
                    <div className="mt-2 text-sm text-slate-800">{conn.port || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">登录用户</div>
                    <div className="mt-2 text-sm text-slate-800">{conn.dbLoginName || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">连接分组</div>
                    <div className="mt-2 text-sm text-slate-800">{conn.connectionGroup || '-'}</div>
                </div>
            </div>

            <div className="mt-5 flex justify-end gap-2 flex-wrap">
              <button
                type="button"
                disabled={!!opLoading[conn.ID]}
                onClick={() => handleTest(conn)}
                className="inline-flex items-center gap-2 rounded-2xl border border-emerald-200 px-4 py-2 text-sm font-medium text-emerald-700 transition hover:bg-emerald-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {opLoading[conn.ID] === 'test' ? (
                  <span className="w-4 h-4 border-2 border-emerald-300 border-t-emerald-600 rounded-full animate-spin" />
                ) : <Plug size={16} />}
                测试连接
              </button>
              <button
                type="button"
                disabled={!!opLoading[conn.ID]}
                onClick={() => handleInit(conn)}
                className="inline-flex items-center gap-2 rounded-2xl border border-indigo-200 px-4 py-2 text-sm font-medium text-indigo-700 transition hover:bg-indigo-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {opLoading[conn.ID] === 'init' ? (
                  <span className="w-4 h-4 border-2 border-indigo-300 border-t-indigo-600 rounded-full animate-spin" />
                ) : <DatabaseZap size={16} />}
                导入表结构
              </button>
              <button
                type="button"
                onClick={() => handleDelete(conn)}
                className="inline-flex items-center gap-2 rounded-2xl border border-rose-200 px-4 py-2 text-sm font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={16} />
                删除连接
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
