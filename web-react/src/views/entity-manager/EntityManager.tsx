import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Blocks, Trash2 } from 'lucide-react';
import { getTbEntityList, deleteTbEntity, TbEntity } from '../../api/sysEntity';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function EntityManager() {
  const [entities, setEntities] = useState<TbEntity[]>([]);
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
      const res = await getTbEntityList({ page: targetPage, pageSize: 20 });
      setEntities(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载逻辑实体失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(ent: TbEntity) {
    const confirmed = await confirm(`确定要移除逻辑实体「${ent.entityName}」吗？`, {
      title: '删除实体声明',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteTbEntity({ ID: ent.ID });
      toast.success('实体已删除');
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
              业务实体构建
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              配置上层应用所需的高级数据大表实体属性。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-purple-50 text-purple-900 border border-purple-100">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">业务实体总计</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载逻辑实体库...
          </div>
        ) : entities.map((ent) => (
          <article key={ent.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <Blocks size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{ent.entityName}</h3>
                  <p className="text-xs mt-1 px-2 py-0.5 bg-slate-100 text-slate-500 rounded font-medium inline-block">{ent.serverName}</p>
                </div>
              </div>
            </div>
            
            <div className="mt-5 space-y-3">
                <div className="grid grid-cols-2 gap-3 text-sm">
                    <div className="border border-slate-100 p-3 rounded-2xl">
                         <div className="text-slate-400 text-xs mb-1">字段总数</div>
                         <div className="font-semibold text-slate-800">{ent.columnCount || 0}</div>
                    </div>
                    <div className="border border-slate-100 p-3 rounded-2xl">
                         <div className="text-slate-400 text-xs mb-1">是否嵌套实体</div>
                         <div className="font-semibold text-slate-800">{ent.containEntity === 1 ? '是' : '否'}</div>
                    </div>
                </div>
                {ent.requiredColumn && (
                    <div className="bg-slate-50 rounded-2xl p-4 text-xs">
                        <div className="text-slate-400 mb-2 uppercase tracking-[0.1em]">必填约束链路</div>
                        <div className="font-mono text-slate-600 break-words">{ent.requiredColumn}</div>
                    </div>
                )}
            </div>

            <div className="mt-5 flex justify-end gap-3 pt-4 border-t border-slate-100">
              <button
                type="button"
                onClick={() => handleDelete(ent)}
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
