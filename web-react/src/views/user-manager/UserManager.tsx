import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { User, Trash2 } from 'lucide-react';
import { getUserList, deleteUser, TbUser } from '../../api/tbUser';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';

export default function UserManager() {
  const [users, setUsers] = useState<TbUser[]>([]);
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
      const res = await getUserList({ page: targetPage, pageSize: 20 });
      setUsers(res.data?.list ?? []);
      setTotal(res.data?.total ?? 0);
    } catch (error) {
      console.error(error);
      toast.error('加载用户列表失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(user: TbUser) {
    const confirmed = await confirm(`确定要删除用户「${user.nickName || user.userName}」吗？`, {
      title: '删除用户',
      confirmText: '确定删除',
    });
    if (!confirmed) return;

    try {
      await deleteUser({ id: user.ID });
      toast.success('用户已删除');
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
              用户管理
            </h1>
            <p className="max-w-xl text-sm leading-7 text-slate-600 sm:text-base">
              管理系统的用户信息和状态。
            </p>
          </div>
          <div className="flex gap-3">
             <div className="rounded-2xl px-6 py-4 bg-slate-900 text-white">
                <div className="text-xs uppercase tracking-[0.2em] opacity-70">用户总数</div>
                <div className="mt-3 text-3xl font-semibold">{total}</div>
             </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        {loading ? (
          <div className="col-span-2 rounded-[24px] border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-sm text-slate-500">
            正在加载用户数据...
          </div>
        ) : users.map((user) => (
          <article key={user.ID} className="group rounded-[24px] border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-600">
                  <User size={24} />
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-tight text-slate-950">{user.nickName || user.userName}</h3>
                  <p className="text-sm text-slate-500">@{user.userName}</p>
                </div>
              </div>
              <div className="flex gap-2">
                <span className={`px-2.5 py-1 rounded-full text-xs font-semibold ${user.enable === 1 ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700'}`}>
                  {user.enable === 1 ? '正常' : '禁用'}
                </span>
              </div>
            </div>
            
            <div className="mt-6 grid gap-3 rounded-[20px] bg-slate-50 p-4 text-sm text-slate-600 sm:grid-cols-2">
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">手机号</div>
                    <div className="mt-2 font-medium text-slate-800">{user.phone || '-'}</div>
                </div>
                <div>
                    <div className="text-[11px] uppercase tracking-[0.2em] text-slate-400">邮箱</div>
                    <div className="mt-2 font-medium text-slate-800">{user.email || '-'}</div>
                </div>
            </div>

            <div className="mt-5 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => handleDelete(user)}
                className="inline-flex items-center gap-2 rounded-2xl border border-rose-200 px-4 py-2 text-sm font-medium text-rose-700 transition hover:bg-rose-50"
              >
                <Trash2 size={16} />
                删除用户
              </button>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
