import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { Search, Database, GitMerge, Clock, LayoutGrid, FileSearch, Play, X, Loader2, Code2, History, Trash2, ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Eye } from 'lucide-react';
import Editor, { type OnMount } from '@monaco-editor/react';
import toast from 'react-hot-toast';
import DatabaseBrowser from '@/components/DatabaseBrowser';
import TableDataPreview from '@/components/TableDataPreview';
import RelateManager from '@/views/relate-manager/RelateManager';
import { fuzzyQuery, getClientData, getHistoryTableNames, getPreferColumnValueList } from '@/api/sysKeywords';
import {
  clearRemoteSQLHistory,
  deleteRemoteSQLHistory,
  getRemoteDatabases,
  getRemoteSQLHistory,
  getRemoteTables,
  getTbConnectionList,
  queryRemoteSQL,
  RemoteSQLHistoryRecord,
  RemoteDatabase,
  RemoteSQLQueryResult,
  saveRemoteSQLHistory,
  TbConnection,
} from '@/api/sysConnection';
import { useProjectStore } from '@/stores/useProjectStore';
import { resolveSelectedConnectionId } from '@/views/config-manager/ConfigManagerSelection';

export default function KeywordsManager() {
  const { activeProjectId, activeConnectionId, setActiveConnectionId } = useProjectStore();
  const [keyword, setKeyword] = useState('');
  const [databaseName, setDatabaseName] = useState('');
  const [tableName, setTableName] = useState('');
  const [connections, setConnections] = useState<string[]>([]);
  const [connectionOptions, setConnectionOptions] = useState<TbConnection[]>([]);
  const [databaseOptions, setDatabaseOptions] = useState<RemoteDatabase[]>([]);
  const [loadingDatabases, setLoadingDatabases] = useState(false);
  const [tableOptions, setTableOptions] = useState<any[]>([]);
  const [showOptions, setShowOptions] = useState(false);
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<any[]>([]);
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [showDbBrowser, setShowDbBrowser] = useState(false);
  const [showSqlQuery, setShowSqlQuery] = useState(false);
  const [showRelateSettings, setShowRelateSettings] = useState(false);
  const [showKeywordDetail, setShowKeywordDetail] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewConnId, setPreviewConnId] = useState(0);
  const [previewDbName, setPreviewDbName] = useState('');
  const [previewTableName, setPreviewTableName] = useState('');
  const rightPanelRef = useRef<HTMLDivElement>(null);

  const handleSelectTable = (tblName: string, shouldScroll = false) => {
    setSelectedTable(tblName);
    if (!shouldScroll) return;

    setTimeout(() => {
      const target = rightPanelRef.current?.querySelector(`[data-table-name="${tblName}"]`);
      if (target) {
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }, 50);
  };

  // History state
  const [historyTables, setHistoryTables] = useState<string[]>([]);
  const [historyKeywords, setHistoryKeywords] = useState<string[]>([]);
  const [showKeywordOptions, setShowKeywordOptions] = useState(false);

  useEffect(() => {
    fetchConnections();
  }, [activeProjectId]);

  useEffect(() => {
    fetchHistoryTableNames();
  }, [activeProjectId, activeConnectionId]);

  useEffect(() => {
    if (!activeProjectId) return;
    if (connectionOptions.length === 0) return;
    const nextConnectionId = resolveSelectedConnectionId(activeConnectionId, connectionOptions);
    if (nextConnectionId !== activeConnectionId) {
      setActiveConnectionId(nextConnectionId);
    }
  }, [activeProjectId, activeConnectionId, connectionOptions, setActiveConnectionId]);

  useEffect(() => {
    if (!showKeywordDetail) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setShowKeywordDetail(false);
    };
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [showKeywordDetail]);

  const fetchConnections = async () => {
    if (!activeProjectId) {
      setConnections([]);
      setConnectionOptions([]);
      return;
    }
    try {
      const res = await getTbConnectionList({ page: 1, pageSize: 999, connectionGroup: String(activeProjectId) });
      if (res.code === 0) {
        const list = res.data.list || [];
        const envs = Array.from(new Set(list.map((c: TbConnection) => c.envName))).filter(Boolean) as string[];
        setConnections(envs);
        setConnectionOptions(list);
        if (list.length === 0) {
          setActiveConnectionId(null);
        }
      }
    } catch (e) { console.error(e); }
  };

  const fetchHistoryTableNames = async () => {
    if (!activeProjectId) {
      setHistoryTables([]);
      return;
    }
    try {
      const res: any = await getHistoryTableNames({
        projectConfigId: activeProjectId,
        connectionId: activeConnectionId,
      });
      if (res.code === 0 && res.data) setHistoryTables(res.data);
    } catch (e) { console.error(e); }
  };

  const fetchHistoryKeywords = async (dbTableStr: string) => {
    if (!dbTableStr || !activeProjectId) {
      setHistoryKeywords([]);
      return;
    }
    try {
      const res: any = await getPreferColumnValueList({
        databaseStr: dbTableStr,
        projectConfigId: activeProjectId,
        connectionId: activeConnectionId,
      });
      if (res.code === 0 && res.data) {
        setHistoryKeywords(res.data.map((item: any) => item.value).filter(Boolean));
      } else {
        setHistoryKeywords([]);
      }
    } catch (e) { setHistoryKeywords([]); }
  };

  const splitDatabaseTable = (value: string) => {
    const parts = value.split(':');
    if (parts.length < 2) {
      return { database: '', table: value.trim() };
    }
    return {
      database: parts[0].trim(),
      table: parts.slice(1).join(':').trim(),
    };
  };

  const applyDatabaseTableValue = (value: string) => {
    const parsed = splitDatabaseTable(value);
    setDatabaseName(parsed.database);
    setTableName(parsed.table);
    fetchHistoryKeywords(value);
  };

  const buildDatabaseTableValue = () => {
    const db = databaseName.trim();
    const table = tableName.trim();
    return db ? `${db}:${table}` : table;
  };

  const tableEntryMatchesCurrentDatabase = (value: string) => {
    const currentDatabase = databaseName.trim().toLowerCase();
    if (!currentDatabase) return true;
    const parsed = splitDatabaseTable(value);
    if (!parsed.database) return true;
    return parsed.database.toLowerCase() === currentDatabase;
  };

  const tableEntryMatchesSearch = (value: string) => {
    const searchValue = tableName.trim().toLowerCase();
    if (!searchValue) return true;
    const parsed = splitDatabaseTable(value);
    return value.toLowerCase().includes(searchValue) || parsed.table.toLowerCase().includes(searchValue);
  };

  const visibleTableOptions = tableOptions.filter((opt) => tableEntryMatchesCurrentDatabase(String(opt.value || '')));
  const visibleHistoryTables = historyTables
    .filter(tableEntryMatchesCurrentDatabase)
    .filter(tableEntryMatchesSearch);

  const handleTableSearch = async (val: string) => {
    const parsed = splitDatabaseTable(val);
    if (parsed.database) {
      setDatabaseName(parsed.database);
      setTableName(parsed.table);
    } else {
      setTableName(val);
    }
    setShowOptions(true);
    try {
      const res: any = await fuzzyQuery({ tableName: parsed.table || val });
      if (res.code === 0 && res.data) setTableOptions(res.data);
      else setTableOptions([]);
    } catch (e) { setTableOptions([]); }
  };

  const handleSearch = async () => {
    const targetConnection = connectionOptions.find(conn => conn.ID === activeConnectionId);
    const databaseStr = buildDatabaseTableValue();
    if (!activeProjectId || !databaseStr || !keyword || !targetConnection) {
      toast.error('请先选择项目和数据库配置，并填写库、表名与关键字');
      return;
    }
    setLoading(true);
    setResults([]);
    setSelectedTable(null);
    try {
      const res: any = await getClientData({
        databaseStr,
        value: keyword,
        environment: targetConnection.envName || '',
        projectConfigId: activeProjectId,
        connectionId: targetConnection.ID,
      });
      if (res.code === 0 && res.data && res.data.length > 0) {
        setResults(res.data);
        setSelectedTable(res.data[0].tableName);
        toast.success(`共命中 ${res.data.length} 张关联表`);
      } else {
        setResults([]);
        toast.error(res.msg || '未找到匹配数据');
      }
    } catch (e) {
      toast.error('搜索失败，请检查服务连接');
    } finally {
      setLoading(false);
    }
  };

  const formatRecordCount = (count: unknown) => {
    const value = Number(count ?? 0);
    if (!Number.isFinite(value)) return '0 条';
    return `${value.toLocaleString()} 条`;
  };

  const selectedConnection = useMemo(
    () => connectionOptions.find(conn => conn.ID === activeConnectionId) || null,
    [activeConnectionId, connectionOptions],
  );
  const selectedConnectionId = selectedConnection ? String(selectedConnection.ID) : '';
  const environment = selectedConnection?.envName || '';

  useEffect(() => {
    if (!showKeywordDetail || !activeProjectId || !selectedConnection) {
      setDatabaseOptions([]);
      setLoadingDatabases(false);
      return;
    }

    let cancelled = false;
    setLoadingDatabases(true);
    getRemoteDatabases({
      connectionGroup: String(activeProjectId),
      envName: selectedConnection.envName || '',
      ID: selectedConnection.ID,
    })
      .then(res => {
        if (cancelled) return;
        if (res.code === 0 && Array.isArray(res.data)) {
          setDatabaseOptions(res.data);
        } else {
          setDatabaseOptions([]);
        }
      })
      .catch(() => {
        if (!cancelled) setDatabaseOptions([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingDatabases(false);
      });

    return () => {
      cancelled = true;
    };
  }, [showKeywordDetail, activeProjectId, selectedConnection]);

  const openKeywordDetail = (conn: TbConnection) => {
    setActiveConnectionId(conn.ID);
    setDatabaseName(conn.databaseName || '');
    setShowKeywordDetail(true);
    setResults([]);
    setSelectedTable(null);
  };

  const openConnectionSqlQuery = (conn: TbConnection) => {
    setActiveConnectionId(conn.ID);
    setShowSqlQuery(true);
  };

  const openConnectionDatabaseBrowser = (conn: TbConnection) => {
    setActiveConnectionId(conn.ID);
    setShowDbBrowser(true);
  };

  const openTablePreview = (value: string, connectionId?: number) => {
    const parsed = splitDatabaseTable(value);
    if (!parsed.database || !parsed.table) return;

    setPreviewDbName(parsed.database);
    setPreviewTableName(parsed.table);
    setPreviewConnId(connectionId || selectedConnection?.ID || activeConnectionId || 0);
    setPreviewOpen(true);
  };

  const openConnectionRelateSettings = (conn: TbConnection) => {
    setActiveConnectionId(conn.ID);
    setShowRelateSettings(true);
  };

  const databaseSelectOptions = useMemo(() => {
    const names = new Set<string>();
    if (databaseName.trim()) names.add(databaseName.trim());
    if (selectedConnection?.databaseName?.trim()) names.add(selectedConnection.databaseName.trim());
    databaseOptions.forEach(db => {
      if (db.databaseName?.trim()) names.add(db.databaseName.trim());
    });
    return Array.from(names);
  }, [databaseName, selectedConnection, databaseOptions]);

  const handleDatabaseChange = (value: string) => {
    setDatabaseName(value);
    if (tableName.trim()) {
      fetchHistoryKeywords(value ? `${value}:${tableName.trim()}` : tableName.trim());
    }
  };

  const renderConnectionCards = () => (
    <div className="flex-1 overflow-y-auto bg-white px-8 py-8">
      <div className="mx-auto max-w-[1500px]">
        <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-2xl font-extrabold text-slate-900">选择数据库配置</h1>
            <p className="mt-1 text-sm text-slate-500">选择一个数据源后进入关键字探查详情</p>
          </div>
        </div>

        <div className="grid gap-4 xl:grid-cols-2">
          {connectionOptions.map((conn) => {
            return (
              <article
                key={conn.ID}
                role="button"
                tabIndex={0}
                onClick={() => openKeywordDetail(conn)}
                onKeyDown={(event) => {
                  if (event.currentTarget !== event.target) return;
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    openKeywordDetail(conn);
                  }
                }}
                className="group rounded-2xl border-2 border-slate-200 bg-white p-5 shadow-sm transition-all hover:border-indigo-300 hover:bg-slate-50 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
              >
                <div className="mb-4 flex items-start justify-between gap-4">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-indigo-50 text-indigo-600">
                      <Database size={20} />
                    </div>
                    <div className="min-w-0">
                      <h4 className="flex items-center gap-2 truncate font-bold text-slate-800">
                        <span className="truncate">{conn.connectionName}</span>
                        {conn.envName && <span className="shrink-0 rounded bg-emerald-100 px-2 py-0.5 text-[10px] font-medium text-emerald-700">{conn.envName}</span>}
                      </h4>
                      <p className="mt-0.5 truncate font-mono text-xs text-slate-500">{conn.connectionType}</p>
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        openConnectionDatabaseBrowser(conn);
                      }}
                      className="inline-flex h-10 items-center gap-2 rounded-xl border border-emerald-500 bg-emerald-500 px-4 text-sm font-bold text-white shadow-lg shadow-emerald-500/20 transition hover:border-emerald-600 hover:bg-emerald-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40"
                      title="浏览数据库"
                    >
                      <Eye size={16} /> 浏览
                    </button>
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        openConnectionSqlQuery(conn);
                      }}
                      className="inline-flex h-9 items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-600 shadow-sm transition hover:border-sky-200 hover:bg-sky-50 hover:text-sky-700 focus:outline-none focus:ring-2 focus:ring-sky-500/30"
                      title="数据库查询"
                    >
                      <FileSearch size={14} /> 查询
                    </button>
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        openConnectionRelateSettings(conn);
                      }}
                      className="inline-flex h-9 items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 text-xs font-semibold text-slate-600 shadow-sm transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
                      title="表关系设置"
                    >
                      <GitMerge size={14} /> 关系
                    </button>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2 rounded-lg bg-slate-50 p-3 font-mono text-xs text-slate-600">
                  <div className="truncate"><span className="text-slate-400">Host: </span>{conn.connectionUrl || '-'}</div>
                  <div className="truncate"><span className="text-slate-400">Port: </span>{conn.port || '-'}</div>
                  <div className="truncate"><span className="text-slate-400">DB: </span>{conn.databaseName || '-'}</div>
                  <div className="truncate"><span className="text-slate-400">User: </span>{conn.dbLoginName || '-'}</div>
                </div>
              </article>
            );
          })}
          {connectionOptions.length === 0 && (
            <div className="col-span-1 rounded-2xl border-2 border-dashed border-slate-200 py-12 text-center text-sm text-slate-400 xl:col-span-2">
              当前项目暂无数据库配置
            </div>
          )}
        </div>
      </div>
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-white text-slate-800 border border-slate-200/60 rounded-3xl m-6 shadow-sm">
      {renderConnectionCards()}

      {showKeywordDetail && (
      <div className="fixed inset-0 z-[10000] flex flex-col overflow-hidden bg-slate-50 text-slate-900">
        <div className="relative flex min-h-[72px] items-center justify-center border-b border-slate-800 bg-slate-900 px-6 shadow-md">
          <h1 className="truncate text-2xl font-extrabold tracking-tight text-teal-400">全局关键字探查</h1>
          <button
            type="button"
            onClick={() => setShowKeywordDetail(false)}
            className="absolute right-6 top-1/2 flex h-10 w-10 -translate-y-1/2 shrink-0 items-center justify-center rounded-xl border border-slate-700 text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
            title="关闭全局关键字探查"
            aria-label="关闭全局关键字探查"
          >
            <X size={18} />
          </button>
        </div>

        <div className="min-h-0 flex-1 flex flex-col overflow-hidden bg-white">
      {/* Top Search Bar */}
      <div className="relative z-30 border-b border-slate-100 px-8 py-6 bg-slate-50/50">
        <div className="mx-auto mb-6 flex max-w-[1180px] flex-wrap items-center justify-center gap-2 text-sm">
          {selectedConnection ? (
            <>
              <span className="inline-flex h-8 items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 font-semibold text-slate-800 shadow-sm">
                <Database size={15} className="text-indigo-500" />
                {selectedConnection.connectionName}
              </span>
              {selectedConnection.envName && (
                <span className="inline-flex h-8 items-center rounded-lg bg-emerald-100 px-3 text-xs font-semibold text-emerald-700">
                  {selectedConnection.envName}
                </span>
              )}
              <span className="inline-flex h-8 items-center rounded-lg bg-slate-100 px-3 font-mono text-xs text-slate-600">
                {selectedConnection.connectionType}
              </span>
              <span className="inline-flex h-8 items-center rounded-lg bg-slate-100 px-3 font-mono text-xs text-slate-600">
                Host: {selectedConnection.connectionUrl || '-'}
              </span>
              <span className="inline-flex h-8 items-center rounded-lg bg-slate-100 px-3 font-mono text-xs text-slate-600">
                Port: {selectedConnection.port || '-'}
              </span>
              <span className="inline-flex h-8 items-center rounded-lg bg-slate-100 px-3 font-mono text-xs text-slate-600">
                DB: {selectedConnection.databaseName || '-'}
              </span>
              <span className="inline-flex h-8 items-center rounded-lg bg-slate-100 px-3 font-mono text-xs text-slate-600">
                User: {selectedConnection.dbLoginName || '-'}
              </span>
            </>
          ) : (
            <div className="inline-flex h-8 items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 text-slate-500 shadow-sm">
              <Database size={18} className="text-indigo-500" />
              未选择数据库配置
            </div>
          )}
        </div>
        <div className="flex items-center justify-center gap-2 max-w-[1600px] mx-auto">
          {/* Database Name Input */}
          <div className="relative w-52 shrink-0">
            <div className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl h-11 px-4 shadow-sm focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-100 transition-all">
              <Database size={16} className="text-indigo-400 shrink-0" />
              <select
                className="min-w-0 flex-1 bg-transparent text-sm text-slate-700 outline-none font-mono disabled:text-slate-400"
                value={databaseName}
                onChange={(e) => handleDatabaseChange(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                disabled={loadingDatabases && databaseSelectOptions.length === 0}
                title={databaseName || '数据库'}
              >
                {!databaseName && <option value="">{loadingDatabases ? '加载数据库...' : '选择数据库'}</option>}
                {databaseSelectOptions.map(name => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Table Name Input */}
          <div className="relative w-[420px] shrink">
            <div className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl h-11 px-4 shadow-sm focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-100 transition-all">
              <Search size={16} className="text-slate-400 shrink-0" />
              <input
                className="bg-transparent text-sm text-slate-700 flex-1 outline-none font-mono placeholder-slate-400"
                placeholder="输入或选择表名"
                value={tableName}
                onChange={(e) => handleTableSearch(e.target.value)}
                onFocus={() => {
                  setShowOptions(true);
                  if (tableName) handleTableSearch(tableName);
                }}
                onBlur={() => setTimeout(() => setShowOptions(false), 200)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
            {showOptions && (
              <div className="absolute z-50 w-full mt-2 bg-white border border-slate-100 shadow-xl rounded-xl max-h-60 overflow-y-auto">
                {/* Fuzzy match results */}
                {visibleTableOptions.map((opt, idx) => (
                  <div key={`fz-${idx}`}
                    className="px-4 py-3 hover:bg-slate-50 cursor-pointer text-slate-700 text-sm font-mono border-b border-slate-50 last:border-0 flex items-center gap-2 transition-colors"
                    onClick={() => {
                      applyDatabaseTableValue(opt.value);
                      setShowOptions(false);
                    }}
                  >
                    <Database size={14} className="text-indigo-400 shrink-0" />
                    {opt.value}
                  </div>
                ))}
                {/* History — shown when no fuzzy results */}
                {visibleTableOptions.length === 0 && visibleHistoryTables.map((t, idx) => (
                  <div key={`hist-${idx}`}
                    className="px-4 py-3 hover:bg-slate-50 cursor-pointer text-slate-600 text-sm font-mono border-b border-slate-50 last:border-0 flex items-center gap-2 transition-colors"
                    onClick={() => {
                      applyDatabaseTableValue(t);
                      setShowOptions(false);
                    }}
                  >
                    <Clock size={14} className="text-slate-400 shrink-0" />
                    {t}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Keyword Input — shows history for selected table */}
          <div className="relative w-44 shrink-0">
            <div className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl h-11 px-4 shadow-sm focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-100 transition-all">
              <span className="text-indigo-400 text-xs font-mono shrink-0">▣</span>
              <input
                className="bg-transparent text-sm text-slate-700 flex-1 outline-none placeholder-slate-400"
                placeholder="关键字"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onFocus={() => historyKeywords.length > 0 && setShowKeywordOptions(true)}
                onBlur={() => setTimeout(() => setShowKeywordOptions(false), 200)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
            {showKeywordOptions && historyKeywords.length > 0 && (
              <div className="absolute z-50 w-full mt-2 bg-white border border-slate-100 shadow-xl rounded-xl max-h-56 overflow-y-auto">
                <div className="px-4 py-2 text-[11px] text-slate-500 uppercase tracking-wider border-b border-slate-50 flex items-center gap-1.5 bg-slate-50">
                  <Clock size={12} /> 历史搜索值
                </div>
                {historyKeywords.map((kw, idx) => (
                  <div key={idx}
                    className="px-4 py-3 hover:bg-indigo-50 hover:text-indigo-700 cursor-pointer text-slate-700 text-sm font-mono border-b border-slate-50 last:border-0 transition-colors"
                    onClick={() => { setKeyword(kw); setShowKeywordOptions(false); }}
                  >
                    {kw}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Search Button */}
          <button
            onClick={handleSearch}
            disabled={loading}
            className="h-11 px-6 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl shadow-md hover:shadow-lg flex items-center gap-2 text-sm font-medium transition-all disabled:opacity-50 shrink-0"
          >
            {loading ? (
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            ) : <Search size={16} />}
            检索数据
          </button>

          {/* Database Browser */}
          <button
            onClick={() => setShowDbBrowser(true)}
            className="h-11 px-5 bg-white border border-slate-200 shadow-sm hover:border-emerald-300 hover:bg-emerald-50 hover:text-emerald-600 text-slate-600 rounded-xl flex items-center gap-2 text-sm font-medium transition-all shrink-0"
          >
            <LayoutGrid size={16} /> 数据库浏览
          </button>
        </div>
      </div>

      {/* Results Area */}
      {results.length === 0 ? (
        <div className="relative z-0 flex-1 flex flex-col items-center justify-center text-slate-400 bg-slate-50/30 rounded-b-3xl overflow-hidden">
          <Database size={56} className="mb-4 opacity-40 text-slate-400" />
          <p className="text-sm font-medium">请输入关键字并开始探查</p>
        </div>
      ) : (
        <div className="relative z-0 flex-1 flex overflow-hidden bg-white rounded-b-3xl">
          {/* Left Panel - Table List */}
          <div className="w-96 shrink-0 border-r border-slate-100 bg-slate-50 flex flex-col">
            <div className="px-5 py-4 border-b border-slate-100 bg-white shadow-sm z-10">
              <div className="flex items-center gap-2 text-xs font-bold text-slate-500 uppercase tracking-wider">
                <span className="w-1.5 h-3.5 bg-indigo-500 rounded-sm inline-block"></span>
                匹配数据表归属
              </div>
            </div>
            <div className="flex-1 overflow-y-auto py-2">
              {results.map((r, i) => (
                <button
                  key={i}
                  onClick={() => handleSelectTable(r.tableName, true)}
                  title={r.tableName}
                  className={`w-full text-left px-5 py-3 text-sm transition-colors border-b border-slate-100/50 last:border-0 ${
                    selectedTable === r.tableName
                      ? 'text-indigo-700 bg-indigo-50 font-semibold border-l-4 border-l-indigo-500'
                      : 'text-slate-600 hover:text-indigo-600 hover:bg-white font-medium border-l-4 border-l-transparent'
                  }`}
                >
                  <span className="flex min-w-0 items-center justify-between gap-3">
                    <span className="min-w-0 truncate">{r.tableName}</span>
                    <span className={`shrink-0 rounded-md border px-2 py-0.5 text-[11px] font-semibold ${
                      selectedTable === r.tableName
                        ? 'border-indigo-200 bg-white text-indigo-600'
                        : 'border-slate-200 bg-white text-slate-500'
                    }`}>
                      {formatRecordCount(r.recordCount)}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* Right Panel - Column Details */}
          <div ref={rightPanelRef} className="min-w-0 flex-1 overflow-y-auto p-8 bg-slate-50/50">
            {results.map((db, idx) => (
              <div
                key={idx}
                data-table-name={db.tableName}
                className={`mb-8 rounded-2xl border overflow-hidden transition-all bg-white cursor-pointer ${
                  selectedTable === db.tableName
                    ? 'border-indigo-200 shadow-md shadow-indigo-100 ring-4 ring-indigo-50'
                    : 'border-slate-200 hover:border-indigo-200 hover:shadow-sm'
                }`}
                onClick={() => handleSelectTable(db.tableName)}
              >
                {/* Table Header */}
                <div className="flex items-center justify-between px-6 py-4 bg-slate-50/80 border-b border-slate-100">
                  <div className="flex items-center gap-3">
                    <Database size={16} className="text-indigo-500" />
                    <span className="text-sm font-bold text-slate-800">{db.tableName}</span>
                    {db.tableAnnotation && (
                      <span className="text-xs text-slate-600 bg-white border border-slate-200 px-2.5 py-1 rounded-md shadow-sm">{db.tableAnnotation}</span>
                    )}
                    <span className="text-xs text-indigo-600 bg-indigo-50 border border-indigo-100 px-2.5 py-1 rounded-md shadow-sm font-semibold">
                      命中 {formatRecordCount(db.recordCount)}
                    </span>
                  </div>
                  <span className="text-xs text-slate-400 font-mono shadow-sm bg-white border border-slate-200 px-2 py-0.5 rounded-md">db: {db.databaseName}</span>
                </div>

                {/* Column Header */}
                <div className="grid grid-cols-3 px-6 py-3 bg-white border-b border-slate-100">
                  <div className="text-xs text-slate-500 font-bold tracking-wide">字段名称</div>
                  <div className="text-xs text-slate-500 font-bold tracking-wide">字段描述</div>
                  <div className="text-xs text-slate-500 font-bold tracking-wide">检索命中值</div>
                </div>

                {/* Column Rows */}
                <div className="divide-y divide-slate-50 bg-white">
                  {db.columnList?.map((col: any, cidx: number) => {
                    const isHit = col.value && (col.value === keyword || String(col.value).includes(keyword));
                    return (
                      <div key={cidx} className={`grid grid-cols-3 px-6 py-3 items-center ${isHit ? 'bg-amber-50/60' : 'hover:bg-slate-50'} transition-colors`}>
                        <code className="text-xs font-mono text-slate-700 bg-slate-100 border border-slate-200 px-2 py-1 rounded w-fit">
                          {col.name}
                        </code>
                        <span className="text-xs text-slate-500">{col.description || '-'}</span>
                        <div>
                          {col.value ? (
                            <span className={`inline-block text-xs font-mono px-2 py-1 rounded-md border ${
                              isHit
                                ? 'bg-amber-100 border-amber-200 text-amber-700 font-bold shadow-sm'
                                : 'bg-slate-50 text-slate-600 border-slate-200'
                            }`}>
                              {col.value}
                            </span>
                          ) : (
                            <span className="text-xs text-slate-300">-</span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
        </div>
      </div>
      )}

      {/* Database Browser Modal */}
      <DatabaseBrowser
        open={showDbBrowser}
        onClose={() => setShowDbBrowser(false)}
        environment={environment}
        environments={connections}
        onEnvironmentChange={() => undefined}
        projectId={activeProjectId || 0}
        focusedConnectionId={selectedConnectionId ? Number(selectedConnectionId) : undefined}
        autoClose={false}
        onTableSelect={(value, connectionId) => {
          applyDatabaseTableValue(value);
          openTablePreview(value, connectionId);
        }}
      />

      <TableDataPreview
        open={previewOpen}
        onClose={() => setPreviewOpen(false)}
        connectionId={previewConnId}
        databaseName={previewDbName}
        tableName={previewTableName}
      />

      <DatabaseQueryModal
        open={showSqlQuery}
        onClose={() => setShowSqlQuery(false)}
        projectId={activeProjectId || 0}
        environment={environment}
        connections={connectionOptions}
        selectedConnectionId={selectedConnectionId}
      />

      <RelateSettingsModal
        open={showRelateSettings}
        onClose={() => setShowRelateSettings(false)}
      />
    </div>
  );
}

interface RelateSettingsModalProps {
  open: boolean;
  onClose: () => void;
}

function RelateSettingsModal({ open, onClose }: RelateSettingsModalProps) {
  useEffect(() => {
    if (!open) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[10000] flex flex-col overflow-hidden bg-slate-950 text-white">
      <div className="flex min-h-[72px] items-center justify-between border-b border-slate-800 bg-slate-900 px-6 shadow-md">
        <div className="min-w-0">
          <h2 className="truncate text-lg font-extrabold tracking-tight">
            <span className="text-teal-400">表关系设置</span>
            <span className="font-normal text-slate-300"> / 全局关键字探查</span>
          </h2>
          <p className="mt-1 text-xs text-slate-400">按表查看字段级关联关系，维护关键字探查的数据链路</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-700 text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
          title="关闭表关系设置"
          aria-label="关闭表关系设置"
        >
          <X size={18} />
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto bg-slate-50 p-6 text-slate-900">
        <RelateManager embedded />
      </div>
    </div>
  );
}

const SQL_QUERY_LIMIT = 200;
const SQL_RESULT_PAGE_SIZE = 200;
const DEFAULT_SQL = `SELECT *
FROM your_table;`;
const SQL_TABLE_COMPLETION_LIMIT = 30;
const SQL_TABLE_COMPLETION_MIN_CHARS = 1;
const SQL_TABLE_CONTEXT_PATTERN = /(?:^|[\s(,;])(?:from|join|update|into|table|desc|describe)\s+(?:[`"\[]?[\w$-]+[`"\]]?\.)?$/i;
const SQL_HISTORY_LIMIT_PER_SCOPE = 50;

const isCursorInsideSQLLiteralOrComment = (text: string) => {
  let inSingleQuote = false;
  let inDoubleQuote = false;
  let inLineComment = false;
  let inBlockComment = false;

  for (let i = 0; i < text.length; i += 1) {
    const ch = text[i];
    const next = text[i + 1];

    if (inLineComment) {
      if (ch === '\n') inLineComment = false;
      continue;
    }

    if (inBlockComment) {
      if (ch === '*' && next === '/') {
        inBlockComment = false;
        i += 1;
      }
      continue;
    }

    if (inSingleQuote) {
      if (ch === "'" && next === "'") {
        i += 1;
        continue;
      }
      if (ch === "'" && text[i - 1] !== '\\') inSingleQuote = false;
      continue;
    }

    if (inDoubleQuote) {
      if (ch === '"' && next === '"') {
        i += 1;
        continue;
      }
      if (ch === '"' && text[i - 1] !== '\\') inDoubleQuote = false;
      continue;
    }

    if (ch === '-' && next === '-') {
      inLineComment = true;
      i += 1;
      continue;
    }
    if (ch === '/' && next === '*') {
      inBlockComment = true;
      i += 1;
      continue;
    }
    if (ch === "'") inSingleQuote = true;
    if (ch === '"') inDoubleQuote = true;
  }

  return inSingleQuote || inDoubleQuote || inLineComment || inBlockComment;
};

const isLikelyTableCompletionContext = (prefixBeforeWord: string) => {
  return SQL_TABLE_CONTEXT_PATTERN.test(prefixBeforeWord.slice(-600));
};

const getTableSuggestionRank = (tableName: string, keyword: string) => {
  const normalizedTable = tableName.toLowerCase();
  const normalizedKeyword = keyword.toLowerCase();
  if (normalizedTable === normalizedKeyword) return 0;
  if (normalizedTable.startsWith(normalizedKeyword)) return 1;
  if (normalizedTable.includes(normalizedKeyword)) return 2;
  return 3;
};

const formatSQLHistoryTime = (value: string) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
};

interface DatabaseQueryModalProps {
  open: boolean;
  onClose: () => void;
  projectId: number | string;
  environment: string;
  connections: TbConnection[];
  selectedConnectionId: string;
}

function DatabaseQueryModal({
  open,
  onClose,
  projectId,
  environment,
  connections,
  selectedConnectionId,
}: DatabaseQueryModalProps) {
  const [localEnvironment, setLocalEnvironment] = useState(environment);
  const [queryConnectionId, setQueryConnectionId] = useState('');
  const [databaseName, setDatabaseName] = useState('');
  const [databaseOptions, setDatabaseOptions] = useState<RemoteDatabase[]>([]);
  const [loadingDatabases, setLoadingDatabases] = useState(false);
  const [sqlText, setSqlText] = useState(DEFAULT_SQL);
  const [queryLoading, setQueryLoading] = useState(false);
  const [queryResult, setQueryResult] = useState<RemoteSQLQueryResult | null>(null);
  const [queryError, setQueryError] = useState('');
  const [hasQueried, setHasQueried] = useState(false);
  const [resultPage, setResultPage] = useState(1);
  const [showSQLHistory, setShowSQLHistory] = useState(false);
  const [sqlHistoryEntries, setSQLHistoryEntries] = useState<RemoteSQLHistoryRecord[]>([]);
  const tableCompletionCacheRef = useRef<Map<string, string[]>>(new Map());
  const tableCompletionNamesRef = useRef<string[]>([]);
  const tableCompletionMetaRef = useRef({ databaseName: '', loading: false });
  const tableCompletionDisposableRef = useRef<{ dispose: () => void } | null>(null);
  const restoredHistoryIdRef = useRef<number | null>(null);

  const selectedConnection = connections.find(conn => String(conn.ID) === queryConnectionId);
  const effectiveEnvironment = localEnvironment || selectedConnection?.envName || '';
  const effectiveDatabaseName = databaseName || selectedConnection?.databaseName || '';
  const sqlHistoryListScope = useMemo(() => ({
    projectConfigId: Number(projectId) || 0,
    ...(selectedConnectionId ? { connectionId: Number(selectedConnectionId) } : {}),
  }), [projectId, selectedConnectionId]);
  const resultTotalRows = queryResult?.rows.length || 0;
  const resultTotalPages = Math.max(1, Math.ceil(resultTotalRows / SQL_RESULT_PAGE_SIZE));
  const resultCurrentPage = Math.min(resultPage, resultTotalPages);
  const pagedResultRows = useMemo(() => {
    if (!queryResult) return [];
    const start = (resultCurrentPage - 1) * SQL_RESULT_PAGE_SIZE;
    return queryResult.rows.slice(start, start + SQL_RESULT_PAGE_SIZE);
  }, [queryResult, resultCurrentPage]);

  const fetchSQLHistory = useCallback(async () => {
    if (!open || !sqlHistoryListScope.projectConfigId) {
      setSQLHistoryEntries([]);
      return;
    }
    try {
      const res = await getRemoteSQLHistory({
        ...sqlHistoryListScope,
        limit: SQL_HISTORY_LIMIT_PER_SCOPE,
      });
      if (res.code === 0 && Array.isArray(res.data)) {
        setSQLHistoryEntries(res.data);
      } else {
        setSQLHistoryEntries([]);
      }
    } catch (e) {
      console.error(e);
      setSQLHistoryEntries([]);
    }
  }, [open, sqlHistoryListScope]);

  const executeSQLQuery = useCallback(async ({
    connectionId,
    targetDatabaseName,
    sql,
    envName = '',
    persistHistory = false,
    successToast = true,
  }: {
    connectionId: string | number;
    targetDatabaseName: string;
    sql: string;
    envName?: string;
    persistHistory?: boolean;
    successToast?: boolean;
  }) => {
    const trimmedSQL = sql.trim();
    const connectionIdNumber = Number(connectionId);
    if (!connectionIdNumber || !trimmedSQL) return false;

    setQueryLoading(true);
    setHasQueried(true);
    setQueryResult(null);
    setQueryError('');
    setResultPage(1);
    try {
      const res: any = await queryRemoteSQL({
        ID: connectionIdNumber,
        databaseName: targetDatabaseName,
        sql: trimmedSQL,
        limit: SQL_QUERY_LIMIT,
      });
      if (res.code === 0 && res.data) {
        setQueryResult(res.data);
        if (persistHistory) {
          const connection = connections.find(conn => conn.ID === connectionIdNumber);
          try {
            const historyRes = await saveRemoteSQLHistory({
              projectConfigId: Number(projectId) || 0,
              envName: envName || connection?.envName || '',
              connectionId: connectionIdNumber,
              databaseName: targetDatabaseName || connection?.databaseName || '',
              sql: trimmedSQL,
            });
            if (historyRes.code === 0 && Array.isArray(historyRes.data)) {
              setSQLHistoryEntries(historyRes.data.filter((entry: RemoteSQLHistoryRecord) => entry.connectionId === connectionIdNumber));
            }
          } catch (historyError) {
            console.error(historyError);
          }
        }
        if (successToast) {
          toast.success(`查询完成，返回 ${res.data.returned || 0} 行`);
        }
        return true;
      }

      setQueryResult(null);
      setQueryError(res.msg || '查询失败，请检查 SQL 或数据库连接');
      return false;
    } catch (e) {
      console.error(e);
      setQueryError('查询失败，请检查 SQL 或数据库连接');
      if (successToast) {
        toast.error('查询失败，请检查 SQL 或数据库连接');
      }
      return false;
    } finally {
      setQueryLoading(false);
    }
  }, [connections, projectId]);

  useEffect(() => {
    if (!open) return;

    const selected = selectedConnectionId
      ? connections.find(conn => String(conn.ID) === selectedConnectionId)
      : undefined;
    const fallback = connections.find(conn => !environment || conn.envName === environment);
    const initialConnection = selected || fallback;
    const initialConnectionId = initialConnection?.ID ? String(initialConnection.ID) : '';

    restoredHistoryIdRef.current = null;
    setLocalEnvironment(initialConnection?.envName || environment);
    setQueryConnectionId(initialConnectionId);
    setDatabaseName(initialConnection?.databaseName || '');
    setSqlText(DEFAULT_SQL);
    setQueryResult(null);
    setQueryError('');
    setHasQueried(false);
    setResultPage(1);
    setShowSQLHistory(false);
  }, [open, environment, selectedConnectionId, connections]);

  useEffect(() => {
    if (!open) return;
    void fetchSQLHistory();
  }, [open, fetchSQLHistory]);

  useEffect(() => {
    if (!open || hasQueried || queryLoading || restoredHistoryIdRef.current || sqlHistoryEntries.length === 0) {
      return;
    }
    const latest = sqlHistoryEntries[0];
    if (!latest?.connectionId || !latest.sql?.trim()) return;
    if (selectedConnectionId && String(latest.connectionId) !== selectedConnectionId) return;

    restoredHistoryIdRef.current = latest.id;
    setDatabaseName(latest.databaseName || '');
    setSqlText(latest.sql);
    setShowSQLHistory(false);

    void executeSQLQuery({
      connectionId: latest.connectionId,
      targetDatabaseName: latest.databaseName || '',
      sql: latest.sql,
      envName: latest.envName || '',
      persistHistory: false,
      successToast: false,
    });
  }, [open, hasQueried, queryLoading, sqlHistoryEntries, selectedConnectionId, executeSQLQuery]);

  useEffect(() => {
    if (!open || !queryConnectionId || !projectId) {
      setDatabaseOptions([]);
      return;
    }

    const fallbackDatabase = connections.find(conn => String(conn.ID) === queryConnectionId)?.databaseName || '';
    setLoadingDatabases(true);
    void getRemoteDatabases({
      connectionGroup: String(projectId),
      envName: localEnvironment,
      ID: Number(queryConnectionId),
    })
      .then((res: any) => {
        if (res.code === 0 && Array.isArray(res.data)) {
          const nextOptions = res.data.filter((db: RemoteDatabase) => db.connectionId === Number(queryConnectionId));
          setDatabaseOptions(nextOptions);
          setDatabaseName(prev => {
            if (prev && nextOptions.some((db: RemoteDatabase) => db.databaseName === prev)) return prev;
            if (fallbackDatabase && nextOptions.some((db: RemoteDatabase) => db.databaseName === fallbackDatabase)) return fallbackDatabase;
            return nextOptions[0]?.databaseName || fallbackDatabase;
          });
        } else {
          setDatabaseOptions([]);
          setDatabaseName(fallbackDatabase);
        }
      })
      .catch((e) => {
        console.error(e);
        setDatabaseOptions([]);
        setDatabaseName(fallbackDatabase);
      })
      .finally(() => setLoadingDatabases(false));
  }, [open, projectId, localEnvironment, queryConnectionId, connections]);

  useEffect(() => {
    const connectionId = queryConnectionId.trim();
    const targetDatabaseName = effectiveDatabaseName.trim();
    tableCompletionMetaRef.current = { databaseName: targetDatabaseName, loading: false };

    if (!open || !connectionId) {
      tableCompletionNamesRef.current = [];
      return;
    }

    const cacheKey = `${connectionId}:${targetDatabaseName}`;
    const cachedTables = tableCompletionCacheRef.current.get(cacheKey);
    if (cachedTables) {
      tableCompletionNamesRef.current = cachedTables;
      return;
    }

    let cancelled = false;
    tableCompletionNamesRef.current = [];
    tableCompletionMetaRef.current = { databaseName: targetDatabaseName, loading: true };

    void getRemoteTables({
      ID: Number(connectionId),
      databaseName: targetDatabaseName,
    })
      .then((res: any) => {
        if (cancelled) return;
        if (res.code === 0 && Array.isArray(res.data)) {
          const tables = Array.from(new Set(res.data.filter(Boolean) as string[]))
            .sort((a, b) => a.localeCompare(b));
          tableCompletionCacheRef.current.set(cacheKey, tables);
          tableCompletionNamesRef.current = tables;
        } else {
          tableCompletionNamesRef.current = [];
        }
      })
      .catch((e) => {
        console.error(e);
        if (!cancelled) tableCompletionNamesRef.current = [];
      })
      .finally(() => {
        if (!cancelled) {
          tableCompletionMetaRef.current = { databaseName: targetDatabaseName, loading: false };
        }
      });

    return () => {
      cancelled = true;
    };
  }, [open, queryConnectionId, effectiveDatabaseName]);

  useEffect(() => {
    return () => {
      tableCompletionDisposableRef.current?.dispose();
      tableCompletionDisposableRef.current = null;
    };
  }, []);

  const handleSqlEditorMount: OnMount = (_, monaco) => {
    tableCompletionDisposableRef.current?.dispose();
    tableCompletionDisposableRef.current = monaco.languages.registerCompletionItemProvider('sql', {
      triggerCharacters: ['.', '`', '"', '['],
      provideCompletionItems: (model: any, position: any) => {
        const tables = tableCompletionNamesRef.current;
        if (!tables.length || tableCompletionMetaRef.current.loading) {
          return { suggestions: [] };
        }

        const textUntilCursor = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });
        if (isCursorInsideSQLLiteralOrComment(textUntilCursor)) {
          return { suggestions: [] };
        }

        const word = model.getWordUntilPosition(position);
        const keyword = word.word.trim();
        if (keyword.length < SQL_TABLE_COMPLETION_MIN_CHARS) {
          return { suggestions: [] };
        }

        const prefixBeforeWord = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: word.startColumn,
        });
        if (!isLikelyTableCompletionContext(prefixBeforeWord)) {
          return { suggestions: [] };
        }

        const lowerKeyword = keyword.toLowerCase();
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };

        const suggestions = tables
          .map(tableName => ({ tableName, rank: getTableSuggestionRank(tableName, lowerKeyword) }))
          .filter(item => item.rank < 3)
          .sort((a, b) => a.rank - b.rank || a.tableName.localeCompare(b.tableName))
          .slice(0, SQL_TABLE_COMPLETION_LIMIT)
          .map((item, index) => ({
            label: item.tableName,
            kind: monaco.languages.CompletionItemKind.Struct,
            insertText: item.tableName,
            detail: tableCompletionMetaRef.current.databaseName
              ? `表 · ${tableCompletionMetaRef.current.databaseName}`
              : '表',
            range,
            sortText: `${item.rank}-${String(index).padStart(2, '0')}-${item.tableName}`,
          }));

        return { suggestions };
      },
    });
  };

  const handleToggleSQLHistory = async () => {
    if (!showSQLHistory) {
      await fetchSQLHistory();
    }
    setShowSQLHistory(prev => !prev);
  };

  const handleUseHistorySQL = (entry: RemoteSQLHistoryRecord) => {
    if (selectedConnectionId && String(entry.connectionId || '') !== selectedConnectionId) {
      toast.error('只能使用当前选中数据源的查询历史');
      return;
    }
    setDatabaseName(entry.databaseName || '');
    setSqlText(entry.sql);
    setQueryResult(null);
    setQueryError('');
    setHasQueried(false);
    setResultPage(1);
    setShowSQLHistory(false);
  };

  const handleRemoveHistorySQL = async (id: number) => {
    try {
      const res = await deleteRemoteSQLHistory({
        ...sqlHistoryListScope,
        id,
      });
      if (res.code === 0 && Array.isArray(res.data)) {
        setSQLHistoryEntries(res.data);
      }
    } catch (e) {
      console.error(e);
      toast.error('删除历史失败');
    }
  };

  const handleClearSQLHistory = async () => {
    try {
      const res = await clearRemoteSQLHistory(sqlHistoryListScope);
      if (res.code === 0) {
        setSQLHistoryEntries([]);
      } else {
        toast.error(res.msg || '清空历史失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('清空历史失败');
    }
  };

  const handleRunQuery = async () => {
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    if (!queryConnectionId) {
      toast.error('请先选择数据源');
      return;
    }
    if (!sqlText.trim()) {
      toast.error('请输入 SQL 查询语句');
      return;
    }

    await executeSQLQuery({
      connectionId: queryConnectionId,
      targetDatabaseName: databaseName,
      sql: sqlText,
      envName: effectiveEnvironment,
      persistHistory: true,
      successToast: true,
    });
  };

  const renderCell = (value: unknown) => {
    if (value === null || typeof value === 'undefined') return <span className="text-slate-300">NULL</span>;
    if (typeof value === 'boolean') return value ? 'true' : 'false';
    const text = String(value);
    return text === '' ? <span className="text-slate-300">空字符串</span> : text;
  };

  const goToResultPage = (page: number) => {
    setResultPage(Math.min(Math.max(page, 1), resultTotalPages));
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[10000] bg-white" onClick={onClose}>
      <div
        className="flex h-screen w-screen flex-col overflow-hidden bg-white"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-6 py-4">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-50 text-sky-600 ring-1 ring-sky-100">
              <Code2 size={19} />
            </div>
            <div className="min-w-0">
              <h2 className="text-base font-bold text-slate-900">数据库查询</h2>
              <p className="mt-0.5 text-xs text-slate-500">只读 SQL 查询，接口默认截取前 {SQL_QUERY_LIMIT} 条结果</p>
            </div>
          </div>

          <div className="flex min-w-0 flex-1 items-center justify-end gap-2 pl-6">
            <label className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-500 shadow-sm">
              环境
              <span className="max-w-32 truncate text-sm font-medium text-slate-700">
                {effectiveEnvironment || '默认环境'}
              </span>
            </label>
            <label className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-500 shadow-sm">
              数据源
              <span
                className="max-w-56 truncate text-sm font-medium text-slate-700"
                title={selectedConnection ? `${selectedConnection.connectionName} / ${selectedConnection.connectionType}` : '未选择数据源'}
              >
                {selectedConnection
                  ? `${selectedConnection.connectionName} / ${selectedConnection.connectionType}`
                  : '未选择数据源'}
              </span>
            </label>
            <label className="flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-500 shadow-sm">
              数据库
              <select
                className="max-w-48 bg-transparent text-sm font-medium text-slate-700 outline-none"
                value={databaseName}
                onChange={(e) => {
                  setDatabaseName(e.target.value);
                  setQueryResult(null);
                  setQueryError('');
                  setHasQueried(false);
                  setResultPage(1);
                }}
                disabled={loadingDatabases || (!databaseOptions.length && !databaseName)}
              >
                {databaseName && !databaseOptions.some(db => db.databaseName === databaseName) && (
                  <option value={databaseName}>{databaseName}</option>
                )}
                {loadingDatabases ? (
                  <option value={databaseName}>加载中...</option>
                ) : databaseOptions.length > 0 ? (
                  databaseOptions.map(db => <option key={`${db.connectionId}:${db.databaseName}`} value={db.databaseName}>{db.databaseName}</option>)
                ) : (
                  <option value={databaseName}>{databaseName || '默认库'}</option>
                )}
              </select>
            </label>
            <button
              className="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-100 hover:text-slate-800"
              onClick={onClose}
              title="关闭"
            >
              <X size={17} />
            </button>
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col bg-white">
          <div className="flex min-h-[300px] basis-[45%] flex-col border-b border-slate-200">
            <div className="flex items-center justify-between border-b border-slate-200 bg-white px-5 py-3">
              <div className="flex items-center gap-2">
                <span className="rounded-md bg-slate-100 px-2 py-1 font-mono text-xs font-semibold text-slate-600">SQL</span>
                <span className="rounded-md border border-emerald-100 bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700">只读查询</span>
                <span className="rounded-md border border-sky-100 bg-sky-50 px-2 py-1 text-xs font-semibold text-sky-700">自动取前 {SQL_QUERY_LIMIT} 条</span>
                {selectedConnection && (
                  <span className="max-w-80 truncate rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs text-slate-500">
                    {selectedConnection.connectionUrl || selectedConnection.connectionName}
                  </span>
                )}
              </div>
              <div className="relative flex items-center gap-2">
                <button
                  onClick={handleToggleSQLHistory}
                  disabled={!projectId}
                  className="flex h-9 items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-600 shadow-sm transition hover:border-sky-200 hover:bg-sky-50 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
                  title="查看历史"
                >
                  <History size={16} />
                  历史
                  {sqlHistoryEntries.length > 0 && (
                    <span className="rounded-md bg-slate-100 px-1.5 py-0.5 text-[11px] text-slate-500">
                      {sqlHistoryEntries.length}
                    </span>
                  )}
                </button>
                {showSQLHistory && (
                  <div className="absolute right-0 top-11 z-50 flex max-h-[460px] w-[560px] max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-2xl shadow-slate-900/15">
                    <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50 px-4 py-3">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <History size={15} className="text-sky-500" />
                          <span className="text-sm font-bold text-slate-800">历史 SQL</span>
                          <span className="rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[11px] font-semibold text-slate-500">
                            {sqlHistoryEntries.length}
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-slate-500">保存成功查询的环境、数据源、数据库和 SQL</p>
                      </div>
                      <div className="flex shrink-0 items-center gap-1">
                        {sqlHistoryEntries.length > 0 && (
                          <button
                            onClick={handleClearSQLHistory}
                            className="flex h-8 items-center gap-1.5 rounded-lg px-2 text-xs font-semibold text-slate-500 transition hover:bg-rose-50 hover:text-rose-600"
                            title="清空当前项目历史"
                          >
                            <Trash2 size={13} />
                            清空
                          </button>
                        )}
                        <button
                          onClick={() => setShowSQLHistory(false)}
                          className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
                          title="关闭"
                        >
                          <X size={15} />
                        </button>
                      </div>
                    </div>
                    <div className="min-h-0 flex-1 overflow-y-auto p-2">
                      {sqlHistoryEntries.length === 0 ? (
                        <div className="flex min-h-40 flex-col items-center justify-center text-slate-400">
                          <History size={30} className="mb-2 opacity-50" />
                          <p className="text-sm font-medium">当前项目暂无成功查询记录</p>
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {sqlHistoryEntries.map((entry) => (
                            <div
                              key={entry.id}
                              onClick={() => handleUseHistorySQL(entry)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' || e.key === ' ') {
                                  e.preventDefault();
                                  handleUseHistorySQL(entry);
                                }
                              }}
                              className="group w-full rounded-lg border border-slate-100 bg-white p-3 text-left transition hover:border-sky-200 hover:bg-sky-50"
                              title="载入 SQL"
                              role="button"
                              tabIndex={0}
                            >
                              <div className="mb-2 flex items-center justify-between gap-3">
                                <div className="flex min-w-0 flex-wrap items-center gap-1.5 text-[11px] font-medium text-slate-500">
                                  <span className="shrink-0 font-semibold text-slate-400">{formatSQLHistoryTime(entry.createdAt)}</span>
                                  <span className="max-w-28 truncate rounded-md border border-slate-200 bg-white px-1.5 py-0.5">
                                    {entry.envName || '默认环境'}
                                  </span>
                                  <span className="max-w-44 truncate rounded-md border border-slate-200 bg-white px-1.5 py-0.5">
                                    {entry.connectionName || `数据源 ${entry.connectionId}`}{entry.connectionType ? ` / ${entry.connectionType}` : ''}
                                  </span>
                                  <span className="max-w-28 truncate rounded-md border border-slate-200 bg-white px-1.5 py-0.5">
                                    {entry.databaseName || '默认库'}
                                  </span>
                                </div>
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleRemoveHistorySQL(entry.id);
                                  }}
                                  className="flex h-7 w-7 items-center justify-center rounded-md text-slate-300 opacity-0 transition hover:bg-rose-50 hover:text-rose-600 group-hover:opacity-100"
                                  title="删除"
                                >
                                  <Trash2 size={13} />
                                </button>
                              </div>
                              <pre className="max-h-24 overflow-hidden whitespace-pre-wrap break-words font-mono text-xs leading-5 text-slate-700">{entry.sql}</pre>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
                <button
                  onClick={handleRunQuery}
                  disabled={queryLoading || !queryConnectionId}
                  className="flex h-9 items-center gap-2 rounded-xl bg-sky-600 px-4 text-sm font-semibold text-white shadow-sm transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {queryLoading ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                  查询
                </button>
              </div>
            </div>
            <div className="min-h-0 flex-1 bg-[#1e1e1e]">
              <Editor
                height="100%"
                defaultLanguage="sql"
                theme="vs-dark"
                value={sqlText}
                onChange={(value) => setSqlText(value || '')}
                onMount={handleSqlEditorMount}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineHeight: 22,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  wordWrap: 'on',
                  automaticLayout: true,
                  tabSize: 2,
                  padding: { top: 14, bottom: 14 },
                  quickSuggestions: { other: true, comments: false, strings: false },
                  acceptSuggestionOnEnter: 'off',
                  tabCompletion: 'on',
                  suggest: { showWords: false },
                }}
              />
            </div>
          </div>

          <div className="flex min-h-0 flex-1 flex-col bg-slate-50">
            <div className="flex items-center justify-between border-b border-slate-200 bg-white px-5 py-3">
              <div className="flex items-center gap-2">
                <Database size={16} className="text-sky-500" />
                <span className="text-sm font-bold text-slate-800">查询结果</span>
                {queryResult && (
                  <>
                    <span className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-500">
                      {queryResult.returned} 行
                    </span>
                    <span className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-500">
                      {queryResult.elapsedMs} ms
                    </span>
                    {queryResult.truncated && (
                      <span className="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-semibold text-amber-700">
                        仅展示前 {queryResult.limit} 条
                      </span>
                    )}
                  </>
                )}
              </div>
              <div className="flex items-center gap-2">
                {queryResult && resultTotalRows > 0 ? (
                  <>
                    <span className="text-xs font-medium text-slate-400">
                      第 {resultCurrentPage} / {resultTotalPages} 页 · 每页 {SQL_RESULT_PAGE_SIZE} 行
                    </span>
                    <div className="flex items-center gap-1 rounded-xl border border-slate-200 bg-slate-50 p-1">
                      <button
                        className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 transition hover:bg-white hover:text-sky-600 disabled:cursor-not-allowed disabled:opacity-35"
                        disabled={resultCurrentPage <= 1}
                        onClick={() => goToResultPage(1)}
                        title="首页"
                      >
                        <ChevronsLeft size={15} />
                      </button>
                      <button
                        className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 transition hover:bg-white hover:text-sky-600 disabled:cursor-not-allowed disabled:opacity-35"
                        disabled={resultCurrentPage <= 1}
                        onClick={() => goToResultPage(resultCurrentPage - 1)}
                        title="上一页"
                      >
                        <ChevronLeft size={15} />
                      </button>
                      <button
                        className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 transition hover:bg-white hover:text-sky-600 disabled:cursor-not-allowed disabled:opacity-35"
                        disabled={resultCurrentPage >= resultTotalPages}
                        onClick={() => goToResultPage(resultCurrentPage + 1)}
                        title="下一页"
                      >
                        <ChevronRight size={15} />
                      </button>
                      <button
                        className="flex h-7 w-7 items-center justify-center rounded-lg text-slate-500 transition hover:bg-white hover:text-sky-600 disabled:cursor-not-allowed disabled:opacity-35"
                        disabled={resultCurrentPage >= resultTotalPages}
                        onClick={() => goToResultPage(resultTotalPages)}
                        title="末页"
                      >
                        <ChevronsRight size={15} />
                      </button>
                    </div>
                  </>
                ) : (
                  <span className="text-xs text-slate-400">结果列表按字段列横向滚动</span>
                )}
              </div>
            </div>

            <div className="min-h-0 flex-1 overflow-auto p-5">
              {!hasQueried && !queryResult ? (
                <div className="flex h-full min-h-56 flex-col items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white text-slate-400">
                  <FileSearch size={42} className="mb-3 opacity-45" />
                  <p className="text-sm font-medium">执行查询后展示前 {SQL_QUERY_LIMIT} 条结果</p>
                </div>
              ) : queryLoading ? (
                <div className="flex h-full min-h-56 flex-col items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500">
                  <Loader2 size={28} className="mb-3 animate-spin text-sky-500" />
                  <p className="text-sm font-medium">正在查询数据库...</p>
                </div>
              ) : queryError ? (
                <div className="flex h-full min-h-56 flex-col items-center justify-center rounded-xl border border-rose-200 bg-rose-50 px-8 text-center text-rose-700">
                  <Database size={38} className="mb-3 opacity-55" />
                  <p className="max-w-4xl text-sm font-semibold leading-6">{queryError}</p>
                  <p className="mt-2 text-xs text-rose-500">如果当前是 Oracle 数据源，不要使用 MySQL 的 LIMIT 语法；可以直接查询，系统会自动只展示前 {SQL_QUERY_LIMIT} 条。</p>
                </div>
              ) : queryResult && queryResult.columns.length > 0 ? (
                <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
                  <div className="overflow-auto">
                    <table className="min-w-full border-collapse text-left text-sm">
                      <thead className="sticky top-0 z-10 bg-slate-100">
                        <tr>
                          <th className="sticky left-0 z-20 w-16 border-b border-r border-slate-200 bg-slate-100 px-3 py-2 text-xs font-bold text-slate-500">#</th>
                          {queryResult.columns.map((column) => (
                            <th key={column} className="whitespace-nowrap border-b border-r border-slate-200 px-4 py-2 text-xs font-bold text-slate-600 last:border-r-0">
                              {column}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {pagedResultRows.map((row, rowIndex) => (
                          <tr key={rowIndex} className="odd:bg-white even:bg-slate-50/70 hover:bg-sky-50">
                            <td className="sticky left-0 border-r border-slate-200 bg-inherit px-3 py-2 text-xs font-semibold text-slate-400">
                              {(resultCurrentPage - 1) * SQL_RESULT_PAGE_SIZE + rowIndex + 1}
                            </td>
                            {queryResult.columns.map((column, columnIndex) => (
                              <td key={`${rowIndex}-${column}-${columnIndex}`} className="max-w-[360px] whitespace-nowrap border-r border-slate-100 px-4 py-2 font-mono text-xs text-slate-700 last:border-r-0">
                                <span className="block overflow-hidden text-ellipsis">{renderCell(row[columnIndex])}</span>
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {queryResult.rows.length === 0 && (
                    <div className="flex min-h-40 items-center justify-center text-sm text-slate-400">查询成功，无返回记录</div>
                  )}
                </div>
              ) : (
                <div className="flex h-full min-h-56 flex-col items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-400">
                  <Database size={38} className="mb-3 opacity-45" />
                  <p className="text-sm font-medium">暂无查询结果</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
