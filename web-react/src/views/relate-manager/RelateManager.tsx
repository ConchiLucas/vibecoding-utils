import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Database, GitMerge, Search, Trash2 } from 'lucide-react';
import { getTbTableRelateList, deleteTbTableRelate, TbTableRelate, getTableComments } from '../../api/sysRelate';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';
import { getTbConnectionList, TbConnection } from '../../api/sysConnection';
import { useProjectStore } from '../../stores/useProjectStore';
import { resolveSelectedConnectionId } from '../config-manager/ConfigManagerSelection';

function formatTableLabel(databaseName?: string, tableName?: string) {
  if (databaseName && tableName) return `${databaseName}:${tableName}`;
  return tableName || databaseName || '-';
}

export default function RelateManager() {
  const [relates, setRelates] = useState<TbTableRelate[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const { confirm, dialogProps } = useConfirm();
  const { activeProjectId, activeConnectionId, setActiveConnectionId } = useProjectStore();
  const [connectionOptions, setConnectionOptions] = useState<TbConnection[]>([]);
  const [lineageFilter, setLineageFilter] = useState('');
  const [tableComments, setTableComments] = useState<Record<string, string>>({});

  const selectedConnection = connectionOptions.find(conn => conn.ID === activeConnectionId) || null;
  const environment = selectedConnection?.envName || '';

  useEffect(() => {
    void loadData(page);
  }, [page, activeProjectId]);

  useEffect(() => {
    void loadConnections();
  }, [activeProjectId]);

  useEffect(() => {
    if (!activeProjectId) return;
    if (connectionOptions.length === 0) return;
    const nextConnectionId = resolveSelectedConnectionId(activeConnectionId, connectionOptions);
    if (nextConnectionId !== activeConnectionId) {
      setActiveConnectionId(nextConnectionId);
    }
  }, [activeProjectId, activeConnectionId, connectionOptions, setActiveConnectionId]);

  useEffect(() => {
    if (!activeProjectId || !selectedConnection || relates.length === 0) {
      setTableComments({});
      return;
    }
    const uniqueTables = new Set<string>();
    for (const relation of relates) {
      if (relation.databaseName && relation.tableName) {
        uniqueTables.add(`${relation.databaseName}:${relation.tableName}`);
      }
      if (relation.relateDatabaseName && relation.relateTableName) {
        uniqueTables.add(`${relation.relateDatabaseName}:${relation.relateTableName}`);
      }
    }
    if (uniqueTables.size === 0) return;

    getTableComments({
      projectConfigId: activeProjectId,
      environment,
      connectionId: selectedConnection.ID,
      tables: Array.from(uniqueTables),
    })
      .then(res => {
        if (res.code === 0 && res.data) setTableComments(res.data);
      })
      .catch(() => {});
  }, [relates, environment, activeProjectId, selectedConnection]);

  async function loadConnections() {
    if (!activeProjectId) {
      setConnectionOptions([]);
      return;
    }
    try {
      const res = await getTbConnectionList({ page: 1, pageSize: 999, connectionGroup: String(activeProjectId) });
      if (res.code === 0) {
        const list = res.data.list || [];
        setConnectionOptions(list);
        if (list.length === 0) {
          setActiveConnectionId(null);
        }
      }
    } catch (error) {
      console.error(error);
    }
  }

  async function loadData(targetPage: number) {
    if (!activeProjectId) {
      setRelates([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await getTbTableRelateList({ page: targetPage, pageSize: 20, projectConfigId: activeProjectId });
      setRelates(res.data?.list ?? []);
    } catch (error) {
      console.error(error);
      toast.error('加载外键血缘关系失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(rel: TbTableRelate) {
    const confirmed = await confirm('确定要断开此关联关系吗？', {
      title: '解绑血缘关系',
      confirmText: '确定解绑',
    });
    if (!confirmed) return;

    try {
      await deleteTbTableRelate({ ID: rel.ID });
      toast.success('血缘记录已删除');
      void loadData(page);
    } catch (error) {
      toast.error('删除失败');
    }
  }

  const filterText = lineageFilter.toLowerCase();
  const filteredRelates = filterText
    ? relates.filter(relation =>
        relation.tableName?.toLowerCase().includes(filterText) ||
        relation.columnName?.toLowerCase().includes(filterText) ||
        relation.relateTableName?.toLowerCase().includes(filterText) ||
        relation.relateColumnName?.toLowerCase().includes(filterText) ||
        relation.databaseName?.toLowerCase().includes(filterText) ||
        relation.relateDatabaseName?.toLowerCase().includes(filterText)
      )
    : relates;

  return (
    <div className="max-w-full space-y-6">
      <div className="flex flex-col gap-4 rounded-2xl border border-slate-200/70 bg-white p-6 shadow-sm lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-xl font-extrabold text-slate-900">表级血缘指配中心</h2>
          <p className="mt-1 text-xs text-slate-500">AI 关系导入与血缘图谱</p>
        </div>
        <div
          className="flex h-10 w-full max-w-md items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm text-slate-600 shadow-sm lg:w-80"
          title={selectedConnection
            ? `${selectedConnection.envName || '默认环境'} / ${selectedConnection.connectionName} / ${selectedConnection.databaseName}`
            : '未选择数据库配置'}
        >
          <Database size={16} className="shrink-0 text-indigo-500" />
          <span className="min-w-0 truncate font-medium">
            {selectedConnection
              ? `${selectedConnection.envName || '默认环境'} / ${selectedConnection.connectionName} / ${selectedConnection.databaseName}`
              : '未选择数据库配置'}
          </span>
        </div>
      </div>

      <div className="px-2">
        <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h3 className="flex items-center gap-2 text-lg font-bold text-slate-800">
            <GitMerge size={20} className="text-indigo-500" />
            已存血缘图谱
          </h3>
          <div className="flex w-full items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 shadow-sm transition-all focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-100 sm:w-72">
            <Search size={14} className="shrink-0 text-slate-400" />
            <input
              type="text"
              placeholder="搜索源表或映射目标..."
              value={lineageFilter}
              onChange={event => setLineageFilter(event.target.value)}
              className="flex-1 border-none bg-transparent text-sm text-slate-700 outline-none placeholder-slate-400"
            />
          </div>
        </div>

        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {loading ? (
            <div className="col-span-full rounded-2xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">
              加载血缘链路...
            </div>
          ) : filteredRelates.length === 0 ? (
            <div className="col-span-full rounded-2xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">
              {lineageFilter ? `未找到与「${lineageFilter}」匹配的血缘记录` : '暂无血缘记录'}
            </div>
          ) : (
            filteredRelates.map(relation => {
              const sourceTableKey = `${relation.databaseName}:${relation.tableName}`;
              const targetTableKey = `${relation.relateDatabaseName}:${relation.relateTableName}`;

              return (
                <article key={relation.ID} className="group relative overflow-hidden rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-indigo-200 hover:shadow-md">
                  <div className="pointer-events-none absolute right-0 top-0 h-24 w-24 bg-gradient-to-bl from-indigo-50 to-transparent" />
                  <div className="relative z-10 flex flex-col gap-6">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-50 text-indigo-500">
                        <GitMerge size={18} className="-rotate-90" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="text-[11px] font-medium tracking-wide text-slate-400">源表节点</div>
                        <div className="truncate text-sm font-bold tracking-wide text-slate-700">
                          {formatTableLabel(relation.databaseName, relation.tableName)}.
                          <span className="text-indigo-600">{relation.columnName || '-'}</span>
                        </div>
                        {tableComments[sourceTableKey] && (
                          <div className="mt-0.5 truncate text-[11px] text-slate-400" title={tableComments[sourceTableKey]}>
                            {tableComments[sourceTableKey]}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="flex flex-col items-center">
                      <div className="relative z-0 h-8 w-px bg-slate-200">
                        <div className="absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2 whitespace-nowrap rounded border border-slate-200 bg-white px-2 py-0.5 font-mono text-[10px] text-slate-400 shadow-sm">
                          REFERENCES
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-50 text-emerald-500">
                        <GitMerge size={18} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="text-[11px] font-medium tracking-wide text-slate-400">映射目标</div>
                        <div className="truncate text-sm font-bold tracking-wide text-slate-700">
                          {formatTableLabel(relation.relateDatabaseName, relation.relateTableName)}.
                          <span className="text-emerald-600">{relation.relateColumnName || '-'}</span>
                        </div>
                        {tableComments[targetTableKey] && (
                          <div className="mt-0.5 truncate text-[11px] text-slate-400" title={tableComments[targetTableKey]}>
                            {tableComments[targetTableKey]}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="relative z-10 mt-5 flex items-center justify-between border-t border-slate-100 pt-4">
                    <span className="rounded bg-slate-50 px-2 py-1 text-[10px] font-medium text-slate-400">操作人: {relation.userName || '-'}</span>
                    <button onClick={() => handleDelete(relation)} className="flex items-center gap-1 rounded bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-500 transition hover:bg-rose-100 hover:text-rose-600">
                      <Trash2 size={12} />
                      移除
                    </button>
                  </div>
                </article>
              );
            })
          )}
        </div>
      </div>

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
