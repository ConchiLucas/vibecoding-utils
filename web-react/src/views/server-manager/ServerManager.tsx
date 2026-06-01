import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Server, Trash2 } from 'lucide-react';
import { getTbInterfaceServerList, deleteTbInterfaceServer, TbInterfaceServer } from '../../api/sysInterfaceServer';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function ServerManager() {
  const [servers, setServers] = useState<TbInterfaceServer[]>([]);
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
      const res = await getTbInterfaceServerList({ page: targetPage, pageSize: 20 });
      setServers(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载服务配置数据失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(server: TbInterfaceServer) {
    const confirmed = await confirm(`确定要删除服务配置「${server.serverName}」吗？`, {
      title: '删除服务配置',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbInterfaceServer({ ID: server.ID });
      toast.success('服务配置已删除');
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
              服务管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理系统中的后端服务实例、路由前缀和端口配置。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-teal-50 text-teal-900 border border-teal-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">服务节点总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载服务数据...
          </div>
        ) : servers.map((sysInterfaceServer) => (
          <article key={sysInterfaceServer.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <Server size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{sysInterfaceServer.serverName}</h3>
                  <p className="text-sm font-mono text-slate-500">{sysInterfaceServer.projectName || '-'}</p>
                </div>
              </div>
            </div>

            <div className="mt-5 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => handleDelete(sysInterfaceServer)}
                className="inline-flex items-center gap-2 rounded-2xl border border-rose-200 px-4 py-2 text-sm font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={16} />
                删除服务
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
