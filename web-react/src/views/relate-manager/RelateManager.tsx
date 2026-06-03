import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { ArrowRight, Database, GitMerge, Network, Search, Trash2 } from 'lucide-react';
import { getTbTableRelateList, deleteTbTableRelate, TbTableRelate, getTableComments } from '../../api/sysRelate';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';
import { getTbConnectionList, TbConnection } from '../../api/sysConnection';
import { useProjectStore } from '../../stores/useProjectStore';
import { resolveSelectedConnectionId } from '../config-manager/ConfigManagerSelection';
import {
  buildTableLineageViewModel,
  tableKey,
} from './lineageGrouping';
import type { TableLineageNode, TableRelationGroup } from './lineageGrouping';

function formatTableLabel(databaseName?: string, tableName?: string) {
  if (databaseName && tableName) return `${databaseName}:${tableName}`;
  return tableName || databaseName || '-';
}

function relationMatchesFilter(relation: TbTableRelate, filter: string) {
  if (!filter) return true;
  return [
    relation.databaseName,
    relation.tableName,
    relation.columnName,
    relation.columnType,
    relation.relateDatabaseName,
    relation.relateTableName,
    relation.relateColumnName,
    relation.relateColumnType,
  ].some(value => value?.toLowerCase().includes(filter));
}

function tableMatchesFilter(table: TableLineageNode, filter: string) {
  if (!filter) return true;
  if (table.key.toLowerCase().includes(filter)) return true;
  return [...table.outgoing, ...table.incoming].some(group =>
    group.tableKey.toLowerCase().includes(filter) ||
    group.relations.some(relation => relationMatchesFilter(relation, filter))
  );
}

function relationEndpointLabel(relation: TbTableRelate, direction: 'source' | 'target') {
  if (direction === 'source') {
    return `${formatTableLabel(relation.databaseName, relation.tableName)}.${relation.columnName || '-'}`;
  }
  return `${formatTableLabel(relation.relateDatabaseName, relation.relateTableName)}.${relation.relateColumnName || '-'}`;
}

export default function RelateManager() {
  const [relates, setRelates] = useState<TbTableRelate[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const { confirm, dialogProps } = useConfirm();
  const { activeProjectId, activeConnectionId, setActiveConnectionId } = useProjectStore();
  const [connectionOptions, setConnectionOptions] = useState<TbConnection[]>([]);
  const [lineageFilter, setLineageFilter] = useState('');
  const [selectedTableKey, setSelectedTableKey] = useState('');
  const [tableComments, setTableComments] = useState<Record<string, string>>({});

  const selectedConnection = connectionOptions.find(conn => conn.ID === activeConnectionId) || null;
  const environment = selectedConnection?.envName || '';
  const viewModel = useMemo(() => buildTableLineageViewModel(relates), [relates]);
  const filterText = lineageFilter.trim().toLowerCase();
  const visibleTables = useMemo(
    () => viewModel.tables.filter(table => tableMatchesFilter(table, filterText)),
    [viewModel.tables, filterText]
  );
  const selectedTable = visibleTables.find(table => table.key === selectedTableKey) || visibleTables[0] || null;

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
    if (!selectedTable) {
      setSelectedTableKey('');
      return;
    }
    if (selectedTable.key !== selectedTableKey) {
      setSelectedTableKey(selectedTable.key);
    }
  }, [selectedTable, selectedTableKey]);

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

  function renderRelationGroup(group: TableRelationGroup, direction: 'outgoing' | 'incoming') {
    const isOutgoing = direction === 'outgoing';
    const groupComment = tableComments[group.tableKey];
    const visibleRelations = group.relations.filter(relation => relationMatchesFilter(relation, filterText));
    if (filterText && visibleRelations.length === 0 && !group.tableKey.toLowerCase().includes(filterText)) return null;

    return (
      <section key={`${direction}-${group.tableKey}`} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="mb-3 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="truncate text-sm font-bold text-slate-800">{formatTableLabel(group.databaseName, group.tableName)}</div>
            {groupComment && <div className="mt-0.5 truncate text-[11px] text-slate-400" title={groupComment}>{groupComment}</div>}
          </div>
          <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-500">{group.relations.length} 字段</span>
        </div>
        <div className="space-y-2">
          {visibleRelations.map(relation => (
            <div key={`${direction}-${relation.ID}`} className="rounded-lg border border-slate-100 bg-slate-50/70 p-3">
              <div className="flex items-center gap-2 text-xs">
                <span className={`min-w-0 flex-1 truncate font-mono ${isOutgoing ? 'text-indigo-600' : 'text-slate-700'}`}>
                  {isOutgoing ? relation.columnName || '-' : relationEndpointLabel(relation, 'source')}
                </span>
                <ArrowRight size={13} className="shrink-0 text-slate-300" />
                <span className={`min-w-0 flex-1 truncate text-right font-mono ${isOutgoing ? 'text-slate-700' : 'text-emerald-600'}`}>
                  {isOutgoing ? relationEndpointLabel(relation, 'target') : relation.relateColumnName || '-'}
                </span>
              </div>
              <div className="mt-2 flex items-center justify-between gap-2 text-[10px] text-slate-400">
                <span className="truncate">
                  {isOutgoing ? relation.columnType || '-' : relation.relateColumnType || '-'}
                  {' -> '}
                  {isOutgoing ? relation.relateColumnType || '-' : relation.columnType || '-'}
                </span>
                <button onClick={() => handleDelete(relation)} className="shrink-0 rounded bg-rose-50 px-2 py-0.5 font-medium text-rose-500 transition hover:bg-rose-100 hover:text-rose-600">
                  移除
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>
    );
  }

  return (
    <div className="max-w-full space-y-6">
      <div className="flex flex-col gap-4 rounded-2xl border border-slate-200/70 bg-white p-6 shadow-sm lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-xl font-extrabold text-slate-900">表级血缘指配中心</h2>
          <p className="mt-1 text-xs text-slate-500">按表查看字段级关联关系</p>
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

      <div className="grid min-h-[520px] gap-5 xl:grid-cols-[360px_minmax(0,1fr)]">
        <aside className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="flex items-center gap-2 text-base font-bold text-slate-800">
                <Network size={18} className="text-indigo-500" />
                表关系总览
              </h3>
              <p className="mt-1 text-xs text-slate-400">{visibleTables.length} 张表 / {relates.length} 条字段关系</p>
            </div>
          </div>
          <div className="mb-4 flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 shadow-sm focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-100">
            <Search size={14} className="shrink-0 text-slate-400" />
            <input
              type="text"
              placeholder="搜索表、字段或库名..."
              value={lineageFilter}
              onChange={event => setLineageFilter(event.target.value)}
              className="min-w-0 flex-1 border-none bg-transparent text-sm text-slate-700 outline-none placeholder-slate-400"
            />
          </div>

          <div className="max-h-[620px] space-y-2 overflow-y-auto pr-1">
            {loading ? (
              <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">加载血缘链路...</div>
            ) : visibleTables.length === 0 ? (
              <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">
                {lineageFilter ? '没有匹配的表关系' : '暂无血缘记录'}
              </div>
            ) : (
              visibleTables.map(table => {
                const active = table.key === selectedTable?.key;
                return (
                  <button
                    key={table.key}
                    onClick={() => setSelectedTableKey(table.key)}
                    className={`w-full rounded-xl border p-3 text-left transition ${
                      active
                        ? 'border-indigo-300 bg-indigo-50 shadow-sm'
                        : 'border-slate-200 bg-white hover:border-indigo-200 hover:bg-slate-50'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className={`truncate text-sm font-bold ${active ? 'text-indigo-700' : 'text-slate-800'}`}>{table.tableName || '-'}</div>
                        <div className="mt-0.5 truncate text-[11px] text-slate-400">{table.databaseName || 'default'}</div>
                      </div>
                      <span className="shrink-0 rounded-full bg-white px-2 py-0.5 text-[11px] font-semibold text-slate-500 shadow-sm">{table.relatedTableCount} 表</span>
                    </div>
                    <div className="mt-3 grid grid-cols-3 gap-2 text-center text-[11px]">
                      <div className="rounded-lg bg-slate-100 px-2 py-1">
                        <div className="font-bold text-slate-700">{table.fieldRelationCount}</div>
                        <div className="text-slate-400">字段</div>
                      </div>
                      <div className="rounded-lg bg-indigo-50 px-2 py-1">
                        <div className="font-bold text-indigo-600">{table.outgoing.reduce((total, group) => total + group.relations.length, 0)}</div>
                        <div className="text-indigo-400">出去</div>
                      </div>
                      <div className="rounded-lg bg-emerald-50 px-2 py-1">
                        <div className="font-bold text-emerald-600">{table.incoming.reduce((total, group) => total + group.relations.length, 0)}</div>
                        <div className="text-emerald-400">进来</div>
                      </div>
                    </div>
                  </button>
                );
              })
            )}
          </div>
        </aside>

        <main className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          {!selectedTable ? (
            <div className="flex min-h-[420px] items-center justify-center rounded-2xl border border-dashed border-slate-200 bg-slate-50 text-sm text-slate-500">
              选择左侧表查看关联详情
            </div>
          ) : (
            <div className="space-y-5">
              <div className="flex flex-col gap-4 border-b border-slate-100 pb-5 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0">
                  <div className="text-xs font-medium text-slate-400">当前表</div>
                  <h3 className="mt-1 truncate text-2xl font-extrabold text-slate-900">{formatTableLabel(selectedTable.databaseName, selectedTable.tableName)}</h3>
                  {tableComments[tableKey(selectedTable.databaseName, selectedTable.tableName)] && (
                    <p className="mt-1 truncate text-sm text-slate-500">{tableComments[tableKey(selectedTable.databaseName, selectedTable.tableName)]}</p>
                  )}
                </div>
                <div className="grid grid-cols-3 gap-3 text-center">
                  <div className="rounded-xl bg-slate-50 px-4 py-2">
                    <div className="text-lg font-extrabold text-slate-800">{selectedTable.relatedTableCount}</div>
                    <div className="text-[11px] text-slate-400">关联表</div>
                  </div>
                  <div className="rounded-xl bg-indigo-50 px-4 py-2">
                    <div className="text-lg font-extrabold text-indigo-600">{selectedTable.outgoing.length}</div>
                    <div className="text-[11px] text-indigo-400">关联出去</div>
                  </div>
                  <div className="rounded-xl bg-emerald-50 px-4 py-2">
                    <div className="text-lg font-extrabold text-emerald-600">{selectedTable.incoming.length}</div>
                    <div className="text-[11px] text-emerald-400">被关联</div>
                  </div>
                </div>
              </div>

              <section>
                <div className="mb-3 flex items-center gap-2">
                  <GitMerge size={17} className="text-indigo-500" />
                  <h4 className="text-sm font-bold text-slate-800">它关联出去的表</h4>
                  <span className="rounded-full bg-indigo-50 px-2 py-0.5 text-[11px] font-semibold text-indigo-500">{selectedTable.outgoing.length}</span>
                </div>
                {selectedTable.outgoing.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 py-8 text-center text-sm text-slate-400">没有向外关联的字段</div>
                ) : (
                  <div className="grid gap-3 lg:grid-cols-2">
                    {selectedTable.outgoing.map(group => renderRelationGroup(group, 'outgoing'))}
                  </div>
                )}
              </section>

              <section>
                <div className="mb-3 flex items-center gap-2">
                  <GitMerge size={17} className="text-emerald-500" />
                  <h4 className="text-sm font-bold text-slate-800">有哪些表关联到它</h4>
                  <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-500">{selectedTable.incoming.length}</span>
                </div>
                {selectedTable.incoming.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 py-8 text-center text-sm text-slate-400">没有其他表关联到当前表</div>
                ) : (
                  <div className="grid gap-3 lg:grid-cols-2">
                    {selectedTable.incoming.map(group => renderRelationGroup(group, 'incoming'))}
                  </div>
                )}
              </section>
            </div>
          )}
        </main>
      </div>

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
