import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Activity, Trash2, CheckCircle2, XCircle } from 'lucide-react';
import { getTbInterfaceLogList, deleteTbInterfaceLog, TbInterfaceLog } from '../../api/sysInterfaceLog';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

type InterfaceLogManagerProps = {
  interfacePaths?: string;
};

export default function InterfaceLogManager({ interfacePaths }: InterfaceLogManagerProps) {
  const [logs, setLogs] = useState<TbInterfaceLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const [total, setTotal] = useState(0);
  const [expandedLogId, setExpandedLogId] = useState<number | null>(null);
  const { confirm, dialogProps } = useConfirm();

  useEffect(() => {
    void loadData(page);
  }, [page, interfacePaths]);

  async function loadData(targetPage: number) {
    setLoading(true);
    try {
      const res = await getTbInterfaceLogList({ page: targetPage, pageSize: 20, interfacePaths });
      setLogs(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载执行日志失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(log: TbInterfaceLog) {
    const confirmed = await confirm(`确定要清除该日志存档记录吗？`, {
      title: '擦除记录',
      confirmText: '确定清除',
    });
    if (!confirmed) return;

    try {
      await deleteTbInterfaceLog({ ID: log.ID });
      toast.success('审计记录已清除');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('清除失败');
    }
  }

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[28px] border border-slate-200 bg-slate-950 px-6 py-8 shadow-sm">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.1),transparent_50%)] pointer-events-none"></div>
        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <h1 className="text-3xl font-semibold tracking-tight text-white sm:text-4xl">
              哨兵审计平台
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-400 sm:text-base">
              监控实时运行测试的 API 请求、报错追溯、高延迟定位及全量日志回放分析。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-white/5 border border-white/10 text-slate-200">
                <div className="flex items-center gap-2">
                    <span className="relative flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                    </span>
                    <span className="text-xs uppercase tracking-[0.2em] text-slate-400">已存档事务</span>
                </div>
                <div className="mt-3 text-3xl font-mono">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="flex flex-col gap-3">
        {loading ? (
          <div className="rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在获取溯源分析样本...
          </div>
        ) : logs.map((log) => (
          <div key={log.ID} className="group flex flex-col rounded-[20px] border border-slate-200 bg-white p-4 shadow-sm transition hover:border-slate-300">
             <div className="flex flex-col md:flex-row items-center gap-4 cursor-pointer" onClick={() => setExpandedLogId(prev => prev === log.ID ? null : log.ID)}>
                 {/* Status Badge */}
                 <div className="shrink-0 flex items-center justify-center w-12 h-12 rounded-[14px]">
                     {log.isSuccess === 1 ? (
                         <div className="flex items-center justify-center w-full h-full bg-emerald-50 text-emerald-500 rounded-[14px]"><CheckCircle2 size={24}/></div>
                     ) : (
                         <div className="flex items-center justify-center w-full h-full bg-rose-50 text-rose-500 rounded-[14px]"><XCircle size={24}/></div>
                     )}
                 </div>
    
                 {/* Main Info */}
                 <div className="flex-1 min-w-0 flex flex-col justify-center">
                     <div className="flex items-center gap-2 mb-1">
                         <span className="font-mono text-sm font-semibold truncate max-w-[200px] md:max-w-xs">{log.interfacePaths || '/'}</span>
                         <span className="text-[10px] text-slate-400 border border-slate-200 px-1.5 py-0.5 rounded">{log.environment || '默认'}</span>
                     </div>
                     <div className="text-xs text-slate-500 font-mono truncate max-w-sm md:max-w-xl opacity-70 group-hover:opacity-100 transition-opacity">Req: {log.reqParams || '-'}  |  Res: {log.resParams || '-'}</div>
                 </div>
    
                 {/* Actions */}
                 <div className="shrink-0 flex items-center gap-4 ml-auto" onClick={(e) => e.stopPropagation()}>
                     <span className="text-xs text-slate-400 font-mono">{new Date(log.UpdatedAt || '').toLocaleString()}</span>
                     <button className="p-2.5 rounded-xl hover:bg-rose-50 hover:text-rose-600 text-slate-400 transition" onClick={() => handleDelete(log)}>
                         <Trash2 size={16} />
                     </button>
                 </div>
             </div>
             
             {/* Expandable Details */}
             {expandedLogId === log.ID && (
               <div className="mt-4 pt-4 border-t border-slate-100 grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="flex flex-col">
                     <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Request Parameters</span>
                     <pre className="text-xs bg-slate-50 p-3 rounded-xl border border-slate-200 text-slate-600 font-mono overflow-auto max-h-48 whitespace-pre-wrap">{log.reqParams || 'Empty'}</pre>
                  </div>
                  <div className="flex flex-col">
                     <span className="text-xs font-semibold text-emerald-600/70 uppercase tracking-wider mb-2">Response Output</span>
                     <pre className="text-xs bg-emerald-50 p-3 rounded-xl border border-emerald-100 text-emerald-800 font-mono overflow-auto max-h-48 whitespace-pre-wrap">{log.resParams || 'Empty'}</pre>
                  </div>
               </div>
             )}
          </div>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
