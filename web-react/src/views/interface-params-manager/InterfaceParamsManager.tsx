import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Package, Trash2, ArrowRightLeft } from 'lucide-react';
import { getTbInterfaceParamsList, deleteTbInterfaceParams, TbInterfaceParams } from '../../api/sysInterfaceParams';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function InterfaceParamsManager() {
  const [paramsList, setParamsList] = useState<TbInterfaceParams[]>([]);
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
      const res = await getTbInterfaceParamsList({ page: targetPage, pageSize: 20 });
      setParamsList(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载关联参数池失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(param: TbInterfaceParams) {
    const confirmed = await confirm(`确定要解绑该参数项吗？`, {
      title: '解绑执行参数',
      confirmText: '确定解绑',
    });
    if (!confirmed) return;

    try {
      await deleteTbInterfaceParams({ ID: param.ID });
      toast.success('参数集已解绑');
      void loadData(page);
    } catch (error) {
      console.error(error);
      toast.error('解绑失败');
    }
  }

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[28px] border border-slate-200 bg-white px-6 py-8 shadow-sm">
        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-2xl space-y-4">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
              全托管接口容器态
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              配置自动化测试调用的前置参数集合、断言返回值依赖以及环境变量约束。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-teal-50 text-teal-900 border border-teal-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">参数集覆盖量</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 xl:grid-cols-2">
        {loading ? (
          <div className="col-span-full rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在读取运行时参数矩阵...
          </div>
        ) : paramsList.map((param) => (
          <article key={param.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
             <div className="flex items-center gap-3 mb-4 pb-4 border-b border-slate-100">
                 <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-slate-50 text-slate-500 shrink-0"><Package size={20}/></div>
                 <div className="flex-1 overflow-hidden">
                     <div className="text-sm font-semibold text-slate-900 truncate" title={param.interfacePaths}>{param.interfacePaths || '/根节点'}</div>
                     <div className="text-xs text-slate-400 mt-0.5 space-x-2">
                         <span>Env: <span className="font-medium text-slate-600">{param.environment || '默认'}</span></span>
                         <span>|</span>
                         <span>Identity: <span className="font-medium text-slate-600">{param.identity || '未配置'}</span></span>
                     </div>
                 </div>
             </div>

             <div className="flex flex-col md:flex-row gap-4 items-stretch opacity-80 group-hover:opacity-100 transition-opacity">
                 <div className="flex-1 rounded-2xl bg-slate-50 border border-slate-100 p-3">
                     <div className="text-[10px] text-slate-400 font-bold uppercase tracking-wider mb-2">Request Mock</div>
                     <div className="text-xs font-mono text-slate-600 break-all h-20 overflow-y-auto">{param.interfaceParams || '{}'}</div>
                 </div>
                 
                 <div className="flex items-center justify-center text-slate-300 transform md:rotate-0 rotate-90">
                     <ArrowRightLeft size={16} />
                 </div>

                 <div className="flex-1 rounded-2xl bg-emerald-50 border border-emerald-100 p-3">
                     <div className="text-[10px] text-emerald-600/70 font-bold uppercase tracking-wider mb-2">Response Assert</div>
                     <div className="text-xs font-mono text-emerald-800 break-all h-20 overflow-y-auto">{param.responseParams || '{}'}</div>
                 </div>
             </div>

            <div className="mt-5 pt-4 flex justify-between items-center text-xs">
              <span className="text-slate-400 ml-1">操作人: {param.userName || '-'}</span>
              <button
                type="button"
                onClick={() => handleDelete(param)}
                className="inline-flex items-center gap-1.5 rounded-xl border border-rose-200 px-4 py-2 font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={14} /> 销毁配置
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
