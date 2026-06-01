import React, { useState, useEffect, useMemo } from 'react';
import './DatabaseBrowser.css';
import { X, Database, Table2, Loader2, ChevronRight, Search, CheckCircle2 } from 'lucide-react';
import { getRemoteDatabases, getRemoteTables, getRemoteTableComments, RemoteDatabase, getTbConnectionList, TbConnection } from '@/api/sysConnection';
import toast from 'react-hot-toast';

export interface DatabaseTableSelection {
  connectionId: number;
  databaseName: string;
  tableName: string;
  tableComment?: string;
}

interface DatabaseBrowserProps {
  open: boolean;
  onClose: () => void;
  environment: string;
  environments: string[];
  onEnvironmentChange: (environment: string) => void;
  projectId: number | string;
  focusedConnectionId?: number;
  onTableSelect?: (value: string, connectionId?: number) => void;
  selectionMode?: 'single' | 'multiple';
  selectedTables?: DatabaseTableSelection[];
  onTablesConfirm?: (tables: DatabaseTableSelection[]) => void;
  autoClose?: boolean;
}

export default function DatabaseBrowser({
  open,
  onClose,
  environment,
  environments,
  onEnvironmentChange,
  projectId,
  focusedConnectionId,
  onTableSelect,
  selectionMode = 'single',
  selectedTables = [],
  onTablesConfirm,
  autoClose = true,
}: DatabaseBrowserProps) {
  const [databases, setDatabases] = useState<RemoteDatabase[]>([]);
  const [selectedDb, setSelectedDb] = useState<RemoteDatabase | null>(null);
  const [tables, setTables] = useState<string[]>([]);
  const [loadingDbs, setLoadingDbs] = useState(false);
  const [loadingTables, setLoadingTables] = useState(false);
  const [loadingComments, setLoadingComments] = useState(false);
  const [tableFilter, setTableFilter] = useState('');
  const [tableComments, setTableComments] = useState<Record<string, string>>({});
  const [connections, setConnections] = useState<TbConnection[]>([]);
  const [selectedConnectionId, setSelectedConnectionId] = useState('');
  const [draftSelectedTables, setDraftSelectedTables] = useState<DatabaseTableSelection[]>([]);
  const focusedConnection = focusedConnectionId
    ? connections.find(conn => conn.ID === focusedConnectionId)
    : undefined;
  const isMultiple = selectionMode === 'multiple';

  const selectedTableKeys = useMemo(() => {
    const keys = new Set<string>();
    draftSelectedTables.forEach(item => {
      keys.add(`${item.connectionId}:${item.databaseName}:${item.tableName}`);
    });
    return keys;
  }, [draftSelectedTables]);

  useEffect(() => {
    if (open && projectId) {
      setSelectedDb(null);
      setTables([]);
      setTableFilter('');
      setDraftSelectedTables(selectedTables);
      void fetchConnections();
    }
    if (!open) {
      setSelectedDb(null);
      setTables([]);
      setTableFilter('');
      setTableComments({});
      setSelectedConnectionId(focusedConnectionId ? String(focusedConnectionId) : '');
      setDraftSelectedTables([]);
    }
  }, [open, environment, projectId, focusedConnectionId]);

  useEffect(() => {
    if (open && isMultiple) {
      setDraftSelectedTables(selectedTables);
    }
  }, [open, isMultiple, selectedTables]);

  const fetchConnections = async () => {
    try {
      const res = await getTbConnectionList({
        page: 1,
        pageSize: 999,
        connectionGroup: String(projectId),
        ...(focusedConnectionId ? {} : { envName: environment }),
      });
      if (res.code === 0 && res.data) {
        const nextConnections = focusedConnectionId
          ? res.data.list.filter(conn => conn.ID === focusedConnectionId)
          : res.data.list;
        setConnections(nextConnections);
        const current = focusedConnectionId ? String(focusedConnectionId) : selectedConnectionId;
        if (current && nextConnections.some(conn => String(conn.ID) === current)) {
          setSelectedConnectionId(current);
          void fetchDatabases(current);
        } else {
          setSelectedConnectionId(focusedConnectionId ? String(focusedConnectionId) : '');
          void fetchDatabases(focusedConnectionId ? String(focusedConnectionId) : '');
        }
      } else {
        setConnections([]);
        setSelectedConnectionId('');
        void fetchDatabases('');
      }
    } catch (e) {
      console.error(e);
      setConnections([]);
      setSelectedConnectionId('');
      void fetchDatabases('');
    }
  };

  const fetchDatabases = async (connectionId: string) => {
    setLoadingDbs(true);
    setSelectedDb(null);
    setTables([]);
    setTableFilter('');
    setTableComments({});
    try {
      const res = await getRemoteDatabases(focusedConnectionId
        ? { ID: Number(connectionId || focusedConnectionId) }
        : {
          connectionGroup: String(projectId),
          envName: environment,
          ...(connectionId ? { ID: Number(connectionId) } : {}),
        });
      if (res.code === 0 && res.data) {
        const nextDatabases = focusedConnectionId
          ? res.data.filter(dbItem => dbItem.connectionId === focusedConnectionId)
          : res.data;
        setDatabases(nextDatabases);
        if (nextDatabases.length === 1) {
          void handleSelectDb(nextDatabases[0]);
        }
      } else {
        setDatabases([]);
        toast.error(res.msg || '获取数据库列表失败');
      }
    } catch (e) {
      console.error(e);
      setDatabases([]);
      toast.error('获取数据库列表失败');
    } finally {
      setLoadingDbs(false);
    }
  };

  const handleSelectDb = async (dbItem: RemoteDatabase) => {
    setSelectedDb(dbItem);
    setTables([]);
    setTableComments({});
    setTableFilter('');
    setLoadingTables(true);
    setLoadingComments(true);
    void getRemoteTableComments({ ID: dbItem.connectionId, databaseName: dbItem.databaseName })
      .then((res) => {
        if (res.code === 0 && res.data) {
          setTableComments(res.data);
        }
      })
      .catch((e) => {
        console.error(e);
      })
      .finally(() => setLoadingComments(false));
    try {
      const res = await getRemoteTables({ ID: dbItem.connectionId, databaseName: dbItem.databaseName });
      if (res.code === 0 && res.data) {
        setTables(res.data);
      } else {
        toast.error(res.msg || '获取表列表失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('获取表列表失败');
    } finally {
      setLoadingTables(false);
    }
  };

  const handleConnectionChange = (connectionId: string) => {
    setSelectedConnectionId(connectionId);
    void fetchDatabases(connectionId);
  };

  const filteredTables = tables.filter(t =>
    !tableFilter || t.toLowerCase().includes(tableFilter.toLowerCase())
  );

  const toggleTableSelection = (tableName: string, dbItem: RemoteDatabase, tableComment: string) => {
    const key = `${dbItem.connectionId}:${dbItem.databaseName}:${tableName}`;
    setDraftSelectedTables(prev => {
      if (prev.some(item => `${item.connectionId}:${item.databaseName}:${item.tableName}` === key)) {
        return prev.filter(item => `${item.connectionId}:${item.databaseName}:${item.tableName}` !== key);
      }
      return [
        ...prev,
        {
          connectionId: dbItem.connectionId,
          databaseName: dbItem.databaseName,
          tableName,
          tableComment,
        },
      ];
    });
  };

  const handleConfirmSelectedTables = () => {
    onTablesConfirm?.(draftSelectedTables);
    if (autoClose) onClose();
  };

  // Calculate grid columns based on available space — aim for 4 columns
  const GRID_COLS = 4;

  if (!open) return null;

  return (
    <div className="db-browser-overlay" onClick={onClose}>
      <div className="db-browser-modal" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="db-browser-header">
          <div className="db-browser-header-left">
            <Database size={20} className="db-browser-header-icon" />
            <h2>数据库浏览器</h2>
            {focusedConnectionId ? (
              <>
                <div className="db-browser-env-select-wrap">
                  <span className="db-browser-env-label">环境</span>
                  <span className="db-browser-env-select db-browser-fixed-scope">
                    {focusedConnection?.envName || environment || '默认环境'}
                  </span>
                </div>
                <div className="db-browser-env-select-wrap db-browser-source-select-wrap">
                  <span className="db-browser-env-label">数据源</span>
                  <span
                    className="db-browser-env-select db-browser-source-select db-browser-fixed-scope"
                    title={focusedConnection ? `${focusedConnection.connectionName} / ${focusedConnection.connectionType}` : '当前选中数据源'}
                  >
                    {focusedConnection
                      ? `${focusedConnection.connectionName} / ${focusedConnection.connectionType}`
                      : '当前选中数据源'}
                  </span>
                </div>
              </>
            ) : (
              <>
                <div className="db-browser-env-select-wrap">
                  <span className="db-browser-env-label">环境</span>
                  <select
                    className="db-browser-env-select"
                    value={environment}
                    onChange={(e) => onEnvironmentChange(e.target.value)}
                  >
                    <option value="">全部环境</option>
                    {environments.map((env) => (
                      <option key={env} value={env}>{env}</option>
                    ))}
                  </select>
                </div>
                <div className="db-browser-env-select-wrap db-browser-source-select-wrap">
                  <span className="db-browser-env-label">数据源</span>
                  <select
                    className="db-browser-env-select db-browser-source-select"
                    value={selectedConnectionId}
                    onChange={(e) => handleConnectionChange(e.target.value)}
                  >
                    <option value="">全部数据源</option>
                    {connections.map((conn) => (
                      <option key={conn.ID} value={conn.ID}>
                        {conn.connectionName} / {conn.connectionType}
                      </option>
                    ))}
                  </select>
                </div>
              </>
            )}
          </div>
          <button className="db-browser-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="db-browser-body">
          {/* Left Panel — Database List */}
          <div className="db-browser-left">
            <div className="db-browser-left-header">
              <span className="db-browser-section-indicator" />
              <span className="db-browser-section-title">数据库列表</span>
              <span className="db-browser-count">{databases.length}</span>
            </div>
            <div className="db-browser-left-list">
              {loadingDbs ? (
                <div className="db-browser-loading">
                  <Loader2 size={22} className="db-browser-spinner" />
                  <span>加载中...</span>
                </div>
              ) : databases.length === 0 ? (
                <div className="db-browser-empty">
                  <Database size={32} className="db-browser-empty-icon" />
                  <span>当前环境无数据库配置</span>
                </div>
              ) : (
                databases.map((dbItem) => (
                  <button
                    key={`${dbItem.connectionId}:${dbItem.databaseName}`}
                    className={`db-browser-db-item ${selectedDb?.connectionId === dbItem.connectionId && selectedDb?.databaseName === dbItem.databaseName ? 'active' : ''}`}
                    onClick={() => handleSelectDb(dbItem)}
                  >
                    <div className="db-browser-db-item-left">
                      <Database size={15} className="db-browser-db-icon" />
                      <div className="db-browser-db-info">
                        <span className="db-browser-db-name">{dbItem.databaseName}</span>
                        <span className="db-browser-db-type">{dbItem.connectionName} / {dbItem.connectionType}</span>
                      </div>
                    </div>
                    <ChevronRight size={14} className="db-browser-chevron" />
                  </button>
                ))
              )}
            </div>
          </div>

          {/* Right Panel — Table Grid */}
          <div className="db-browser-right">
            {selectedDb ? (
              <>
                <div className="db-browser-right-header">
                  <div className="db-browser-right-header-info">
                    <Table2 size={16} className="db-browser-table-icon" />
                    <span className="db-browser-right-title">{selectedDb.databaseName}</span>
                    <span className="db-browser-table-count">
                      {loadingTables ? '...' : `${filteredTables.length} 张表`}
                    </span>
                    {loadingComments && (
                      <span className="db-browser-comment-loading">注释加载中...</span>
                    )}
                    {isMultiple && (
                      <span className="db-browser-selected-count">{draftSelectedTables.length} 已选</span>
                    )}
                  </div>
                  <div className="db-browser-search-box">
                    <Search size={14} className="db-browser-search-icon" />
                    <input
                      className="db-browser-search-input"
                      placeholder="搜索表名..."
                      value={tableFilter}
                      onChange={e => setTableFilter(e.target.value)}
                    />
                  </div>
                </div>
                <div className="db-browser-right-body">
                  {loadingTables ? (
                    <div className="db-browser-loading">
                      <Loader2 size={24} className="db-browser-spinner" />
                      <span>正在实时获取表列表...</span>
                    </div>
                  ) : filteredTables.length === 0 ? (
                    <div className="db-browser-empty">
                      <Table2 size={36} className="db-browser-empty-icon" />
                      <span>{tableFilter ? '没有匹配的表' : '该数据库暂无表'}</span>
                    </div>
                  ) : (
                    <div className="db-browser-grid-wrapper">
                      {/* Column Headers */}
                      <div className="db-browser-grid-header" style={{ gridTemplateColumns: `50px repeat(${GRID_COLS}, 1fr)` }}>
                        <div className="db-browser-grid-header-cell db-browser-grid-row-num">#</div>
                        {Array.from({ length: GRID_COLS }).map((_, i) => (
                          <div key={i} className="db-browser-grid-header-cell">列 {i + 1}</div>
                        ))}
                      </div>
                      {/* Table Cells */}
                      <div className="db-browser-grid-body">
                        {Array.from({ length: Math.ceil(filteredTables.length / GRID_COLS) }).map((_, rowIdx) => (
                          <div
                            key={rowIdx}
                            className="db-browser-grid-row"
                            style={{ gridTemplateColumns: `50px repeat(${GRID_COLS}, 1fr)` }}
                          >
                            <div className="db-browser-grid-cell db-browser-grid-row-num">
                              {rowIdx + 1}
                            </div>
                            {Array.from({ length: GRID_COLS }).map((_, colIdx) => {
                              const tableIdx = rowIdx * GRID_COLS + colIdx;
                              const tableName = filteredTables[tableIdx];
                              const tableComment = tableName ? tableComments[tableName] : '';
                              const selectedKey = tableName && selectedDb
                                ? `${selectedDb.connectionId}:${selectedDb.databaseName}:${tableName}`
                                : '';
                              const selected = selectedKey ? selectedTableKeys.has(selectedKey) : false;
                              return (
                                <div
                                  key={colIdx}
                                  className={`db-browser-grid-cell ${tableName ? 'has-data' : 'empty-cell'}${tableName && (onTableSelect || isMultiple) ? ' clickable-table-cell' : ''}${selected ? ' selected-table-cell' : ''}`}
                                  title={tableName ? [selectedDb ? `${selectedDb.databaseName}:${tableName}` : tableName, tableComment].filter(Boolean).join('\n') : ''}
                                  onClick={() => {
                                    if (tableName && selectedDb && isMultiple) {
                                      toggleTableSelection(tableName, selectedDb, tableComment);
                                      return;
                                    }
                                    if (tableName && selectedDb && onTableSelect) {
                                      onTableSelect(`${selectedDb.databaseName}:${tableName}`, selectedDb.connectionId);
                                      if (autoClose) onClose();
                                    }
                                  }}
                                >
                                  {tableName ? (
                                      <span className="db-browser-table-meta">
                                        <span className="db-browser-table-name">{tableName}</span>
                                        {tableComment ? (
                                          <span className="db-browser-table-comment">{tableComment}</span>
                                        ) : null}
                                        {selected ? (
                                          <CheckCircle2 size={16} className="db-browser-table-selected-icon" />
                                        ) : null}
                                      </span>
                                    ) : null}
                                  </div>
                              );
                            })}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </>
            ) : (
              <div className="db-browser-placeholder">
                <Database size={48} className="db-browser-placeholder-icon" />
                <h3>选择一个数据库</h3>
                <p>点击左侧的数据库名称查看其中的表</p>
              </div>
            )}
          </div>
        </div>
        {isMultiple && (
          <div className="db-browser-footer">
            <div className="db-browser-footer-info">
              <span className="db-browser-footer-count">{draftSelectedTables.length}</span>
              <span>张表已选择</span>
            </div>
            <div className="db-browser-footer-actions">
              <button className="db-browser-footer-cancel" onClick={onClose}>
                取消
              </button>
              <button className="db-browser-footer-confirm" onClick={handleConfirmSelectedTables}>
                确认选择
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
