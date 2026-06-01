import { useCallback, useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { AlertCircle, Bookmark, Database, Loader2, Pencil, Plus, RefreshCcw, Table2, Trash2, X } from 'lucide-react';
import DatabaseBrowser, { DatabaseTableSelection } from '../../components/DatabaseBrowser';
import { getTbConnectionList, getRemoteTablePreview, ColumnPreview, TbConnection } from '../../api/sysConnection';
import {
  AgileTableSampleHistory,
  AgileTableSampleRecord,
  getAgileTableSampleHistory,
  getAgileTableSamples,
  saveAgileTableSamples,
} from '../../api/agileTableSample';
import { useProjectStore } from '../../stores/useProjectStore';
import './AgileTableSamplesManager.css';

type PreviewState = {
  loading: boolean;
  error?: string;
  columns: ColumnPreview[];
  total: number;
};

type BrowserMode = 'create' | 'edit';

const sampleKey = (sample: Pick<AgileTableSampleRecord, 'connectionId' | 'databaseName' | 'tableName'>) =>
  `${sample.connectionId}:${sample.databaseName}:${sample.tableName}`;

const renderCellValue = (column: ColumnPreview) => {
  if (column.isNull) return 'NULL';
  if (column.value === '') return "''";
  return column.value;
};

const formatHistoryTime = (value?: string) => {
  if (!value) return '-';
  return new Date(value).toLocaleString();
};

const asSampleList = (value: unknown): AgileTableSampleRecord[] => {
  return Array.isArray(value) ? value : [];
};

const asHistoryList = (value: unknown): AgileTableSampleHistory[] => {
  return Array.isArray(value) ? value : [];
};

export default function AgileTableSamplesManager() {
  const activeProject = useProjectStore(state => state.activeProject);
  const activeProjectId = useProjectStore(state => state.activeProjectId);
  const activeConnectionId = useProjectStore(state => state.activeConnectionId);
  const setActiveConnectionId = useProjectStore(state => state.setActiveConnectionId);

  const [connections, setConnections] = useState<TbConnection[]>([]);
  const [connectionLoading, setConnectionLoading] = useState(false);
  const [samples, setSamples] = useState<AgileTableSampleRecord[]>([]);
  const [previewMap, setPreviewMap] = useState<Record<string, PreviewState>>({});
  const [samplesLoading, setSamplesLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [browserOpen, setBrowserOpen] = useState(false);
  const [browserMode, setBrowserMode] = useState<BrowserMode>('create');
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [histories, setHistories] = useState<AgileTableSampleHistory[]>([]);
  const [activeHistoryId, setActiveHistoryId] = useState<number | null>(null);
  const [historyApplying, setHistoryApplying] = useState(false);
  const [nameDialogOpen, setNameDialogOpen] = useState(false);
  const [historyNameInput, setHistoryNameInput] = useState('');
  const [activeBusinessName, setActiveBusinessName] = useState('');
  const [pendingSelectionTables, setPendingSelectionTables] = useState<DatabaseTableSelection[]>([]);

  const activeConnection = useMemo(
    () => connections.find(conn => conn.ID === activeConnectionId),
    [connections, activeConnectionId]
  );

  const environments = useMemo(
    () => Array.from(new Set(connections.map(conn => conn.envName).filter(Boolean) as string[])),
    [connections]
  );

  const selectedTables = useMemo<DatabaseTableSelection[]>(
    () => samples.map(sample => ({
      connectionId: sample.connectionId,
      databaseName: sample.databaseName,
      tableName: sample.tableName,
      tableComment: sample.tableComment,
    })),
    [samples]
  );

  const browserSelectedTables = useMemo(
    () => (browserMode === 'edit' ? selectedTables : []),
    [browserMode, selectedTables]
  );

  const activeHistory = useMemo(
    () => histories.find(history => history.ID === activeHistoryId) || histories[0] || null,
    [histories, activeHistoryId]
  );

  const activeHistoryTables = useMemo(
    () => activeHistory?.tables || [],
    [activeHistory]
  );

  const loadConnections = useCallback(async () => {
    if (!activeProjectId) {
      setConnections([]);
      return;
    }
    setConnectionLoading(true);
    try {
      const res = await getTbConnectionList({
        page: 1,
        pageSize: 999,
        connectionGroup: String(activeProjectId),
      });
      const nextConnections = res.data?.list || [];
      setConnections(nextConnections);
      if (nextConnections.length === 0) {
        setActiveConnectionId(null);
        return;
      }
      const exists = activeConnectionId
        ? nextConnections.some(conn => conn.ID === activeConnectionId)
        : false;
      if (!exists) {
        setActiveConnectionId(nextConnections[0].ID);
      }
    } catch {
      toast.error('加载数据源失败');
      setConnections([]);
    } finally {
      setConnectionLoading(false);
    }
  }, [activeProjectId, activeConnectionId, setActiveConnectionId]);

  const loadPreviews = useCallback(async (nextSamples: AgileTableSampleRecord[]) => {
    if (nextSamples.length === 0) {
      setPreviewMap({});
      return;
    }

    const loadingMap = nextSamples.reduce<Record<string, PreviewState>>((acc, sample) => {
      acc[sampleKey(sample)] = { loading: true, columns: [], total: 0 };
      return acc;
    }, {});
    setPreviewMap(loadingMap);

    const results = await Promise.all(nextSamples.map(async sample => {
      const key = sampleKey(sample);
      try {
        const res = await getRemoteTablePreview({
          ID: sample.connectionId,
          databaseName: sample.databaseName,
          tableName: sample.tableName,
          offset: 0,
        });
        if (res.code === 0 && res.data) {
          return [key, {
            loading: false,
            columns: res.data.columns || [],
            total: res.data.total || 0,
          } satisfies PreviewState] as const;
        }
        return [key, {
          loading: false,
          columns: [],
          total: 0,
          error: res.msg || '加载失败',
        } satisfies PreviewState] as const;
      } catch (error: any) {
        return [key, {
          loading: false,
          columns: [],
          total: 0,
          error: error?.message || '加载失败',
        } satisfies PreviewState] as const;
      }
    }));

    setPreviewMap(Object.fromEntries(results));
  }, []);

  const loadSamples = useCallback(async () => {
    if (!activeProjectId || !activeConnectionId) {
      setSamples([]);
      setPreviewMap({});
      setActiveBusinessName('');
      return;
    }
    setSamplesLoading(true);
    try {
      const res = await getAgileTableSamples({
        projectConfigId: activeProjectId,
        connectionId: activeConnectionId,
      });
      const nextSamples = asSampleList(res.data);
      setSamples(nextSamples);
      setActiveBusinessName(nextSamples.find(sample => sample.businessName)?.businessName?.trim() || '');
      await loadPreviews(nextSamples);
    } catch {
      toast.error('加载表样本失败');
      setSamples([]);
      setPreviewMap({});
      setActiveBusinessName('');
    } finally {
      setSamplesLoading(false);
    }
  }, [activeProjectId, activeConnectionId, loadPreviews]);

  const loadHistory = useCallback(async (silent = false) => {
    if (!activeProjectId || !activeConnectionId) {
      setHistories([]);
      setActiveHistoryId(null);
      return;
    }
    setHistoryLoading(true);
    try {
      const res = await getAgileTableSampleHistory({
        projectConfigId: activeProjectId,
        connectionId: activeConnectionId,
      });
      const nextHistories = asHistoryList(res.data);
      setHistories(nextHistories);
      setActiveHistoryId(prev => {
        if (prev && nextHistories.some(history => history.ID === prev)) {
          return prev;
        }
        return nextHistories[0]?.ID || null;
      });
    } catch {
      if (!silent) {
        toast.error('加载业务方案失败');
      }
      setHistories([]);
      setActiveHistoryId(null);
    } finally {
      setHistoryLoading(false);
    }
  }, [activeProjectId, activeConnectionId]);

  useEffect(() => {
    void loadConnections();
  }, [loadConnections]);

  useEffect(() => {
    void loadSamples();
  }, [loadSamples]);

  const persistSelections = async (tables: DatabaseTableSelection[], historyName?: string) => {
    if (!activeProjectId || !activeConnectionId) {
      toast.error('请先选择项目和数据源');
      return false;
    }
    const normalizedBusinessName = historyName?.trim() || '';
    const scopedTables = tables
      .filter(table => table.connectionId === activeConnectionId)
      .map(table => ({
        databaseName: table.databaseName,
        tableName: table.tableName,
        tableComment: table.tableComment,
      }));

    setSaving(true);
    try {
      const res = await saveAgileTableSamples({
        projectConfigId: activeProjectId,
        connectionId: activeConnectionId,
        historyName: normalizedBusinessName || undefined,
        tables: scopedTables,
      });
      const nextSamples = asSampleList(res.data);
      setSamples(nextSamples);
      setActiveBusinessName(normalizedBusinessName);
      await loadPreviews(nextSamples);
      if (historyOpen) {
        await loadHistory(true);
      }
      setBrowserOpen(false);
      toast.success('表样本已保存');
      return true;
    } catch {
      toast.error('保存表样本失败');
      return false;
    } finally {
      setSaving(false);
    }
  };

  const openCreateBrowser = () => {
    setBrowserMode('create');
    setBrowserOpen(true);
  };

  const openEditBrowser = () => {
    setBrowserMode('edit');
    setBrowserOpen(true);
  };

  const handleBrowserTablesConfirm = (tables: DatabaseTableSelection[]) => {
    const scopedTables = tables.filter(table => table.connectionId === activeConnectionId);
    if (browserMode === 'edit') {
      void persistSelections(scopedTables, activeBusinessName || undefined);
      return;
    }
    if (scopedTables.length === 0) {
      toast.error('请至少选择一张表');
      return;
    }
    setPendingSelectionTables(scopedTables);
    setHistoryNameInput('');
    setBrowserOpen(false);
    setNameDialogOpen(true);
  };

  const confirmNamedSelection = async () => {
    const name = historyNameInput.trim();
    if (!name) {
      toast.error('请输入业务名称');
      return;
    }
    const ok = await persistSelections(pendingSelectionTables, name);
    if (ok) {
      setNameDialogOpen(false);
      setPendingSelectionTables([]);
    }
  };

  const cancelNamedSelection = () => {
    setNameDialogOpen(false);
    setPendingSelectionTables([]);
  };

  const openHistory = () => {
    setHistoryOpen(true);
    void loadHistory();
  };

  const applyActiveHistory = async () => {
    if (!activeHistory || !activeConnectionId) {
      toast.error('请选择一个业务方案');
      return;
    }
    setHistoryApplying(true);
    try {
      const ok = await persistSelections(activeHistoryTables.map(table => ({
        connectionId: activeConnectionId,
        databaseName: table.databaseName,
        tableName: table.tableName,
        tableComment: table.tableComment,
      })), activeHistory.historyName);
      if (ok) {
        setHistoryOpen(false);
      }
    } finally {
      setHistoryApplying(false);
    }
  };

  const removeSample = (sample: AgileTableSampleRecord) => {
    const nextTables = selectedTables.filter(item => sampleKey(item) !== sampleKey(sample));
    void persistSelections(nextTables, activeBusinessName || undefined);
  };

  const refreshPreviews = () => {
    void loadPreviews(samples);
  };

  const renderSample = (sample: AgileTableSampleRecord) => {
    const key = sampleKey(sample);
    const preview = previewMap[key];
    const columns = preview?.columns || [];

    return (
      <section className="ats-sample-row" key={key}>
        <div className="ats-sample-meta">
          <div className="ats-sample-title" title={`${sample.databaseName}.${sample.tableName}`}>
            <Table2 size={16} />
            <span>{sample.tableName}</span>
          </div>
          <div className="ats-sample-db" title={sample.databaseName}>{sample.databaseName}</div>
          {sample.tableComment ? (
            <div className="ats-sample-comment" title={sample.tableComment}>{sample.tableComment}</div>
          ) : null}
          <button
            type="button"
            className="ats-icon-button ats-remove-button"
            onClick={() => removeSample(sample)}
            title="移除"
          >
            <Trash2 size={15} />
          </button>
        </div>

        <div className="ats-sample-fields">
          {preview?.loading ? (
            <div className="ats-inline-state">
              <Loader2 size={18} className="ats-spin" />
              <span>加载中</span>
            </div>
          ) : preview?.error ? (
            <div className="ats-inline-state ats-inline-error">
              <AlertCircle size={18} />
              <span>{preview.error}</span>
            </div>
          ) : columns.length === 0 ? (
            <div className="ats-inline-state">
              <span>暂无字段</span>
            </div>
          ) : (
            <div className="ats-field-wrap-grid">
              {columns.map(column => {
                const value = renderCellValue(column);
                return (
                  <div
                    key={column.name}
                    className="ats-field-card"
                  >
                    <div className="ats-field-card-section ats-field-card-name" title={column.name}>
                      {column.name}
                    </div>
                    <div className="ats-field-card-section ats-field-card-desc" title={column.description || '-'}>
                      {column.description || '-'}
                    </div>
                    <div className="ats-field-card-section ats-field-card-value" title={value}>
                      {value}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </section>
    );
  };

  return (
    <div className="ats-page">
      <header className="ats-toolbar">
        <div className="ats-heading">
          <div className="ats-heading-icon">
            <Database size={20} />
          </div>
          <div>
            <h1>表样本</h1>
            <div className="ats-subtitle">
              <span>{activeProject || '未选择项目'}</span>
              {activeConnection ? (
                <span>{activeConnection.envName || '默认环境'} / {activeConnection.connectionName}</span>
              ) : null}
              {activeBusinessName ? (
                <span title={activeBusinessName}>{activeBusinessName}</span>
              ) : null}
            </div>
          </div>
        </div>

        <div className="ats-actions">
          {activeConnection ? (
            <div className="ats-connection-chip" title={`${activeConnection.connectionName} / ${activeConnection.connectionType}`}>
              <Database size={15} />
              <span>{activeConnection.connectionName}</span>
              <em>{activeConnection.connectionType}</em>
            </div>
          ) : null}
          <button
            type="button"
            className="ats-button ats-button-ghost"
            onClick={refreshPreviews}
            disabled={samples.length === 0 || samplesLoading}
            title="刷新"
          >
            <RefreshCcw size={16} />
          </button>
          <button
            type="button"
            className="ats-button ats-button-secondary"
            onClick={openHistory}
            disabled={!activeProjectId || !activeConnectionId || connectionLoading}
          >
            <Bookmark size={16} />
            业务方案
          </button>
          <button
            type="button"
            className="ats-button ats-button-secondary"
            onClick={openEditBrowser}
            disabled={!activeProjectId || !activeConnectionId || connectionLoading || saving}
          >
            <Pencil size={16} />
            编辑当前页面
          </button>
          <button
            type="button"
            className="ats-button ats-button-primary"
            onClick={openCreateBrowser}
            disabled={!activeProjectId || !activeConnectionId || connectionLoading || saving}
          >
            {saving ? <Loader2 size={16} className="ats-spin" /> : <Plus size={16} />}
            新建业务方案
          </button>
        </div>
      </header>

      <main className="ats-content">
        {!activeProjectId ? (
          <div className="ats-empty">请先选择项目</div>
        ) : connectionLoading ? (
          <div className="ats-empty">
            <Loader2 size={20} className="ats-spin" />
            <span>加载数据源</span>
          </div>
        ) : !activeConnectionId ? (
          <div className="ats-empty">暂无数据源</div>
        ) : samplesLoading ? (
          <div className="ats-empty">
            <Loader2 size={20} className="ats-spin" />
            <span>加载表样本</span>
          </div>
        ) : samples.length === 0 ? (
          <div className="ats-empty">
            <Database size={34} />
            <span>暂无表样本</span>
            <button
              type="button"
              className="ats-button ats-button-primary"
              onClick={openCreateBrowser}
            >
              <Plus size={16} />
              新建业务方案
            </button>
          </div>
        ) : (
          <div className="ats-sample-list">
            {samples.map(renderSample)}
          </div>
        )}
      </main>

      {historyOpen && (
        <div className="ats-history-overlay" onClick={() => setHistoryOpen(false)}>
          <div className="ats-history-modal" onClick={event => event.stopPropagation()}>
            <div className="ats-history-header">
              <div className="ats-history-title">
                <Bookmark size={18} />
                <span>业务方案</span>
              </div>
              <button
                type="button"
                className="ats-icon-button ats-history-close"
                onClick={() => setHistoryOpen(false)}
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>

            <div className="ats-history-body">
              <aside className="ats-history-list">
                {historyLoading ? (
                  <div className="ats-history-state">
                    <Loader2 size={18} className="ats-spin" />
                    <span>加载中</span>
                  </div>
                ) : histories.length === 0 ? (
                  <div className="ats-history-state">暂无业务方案</div>
                ) : (
                  histories.map(history => (
                    <button
                      key={history.ID}
                      type="button"
                      className={`ats-history-item ${activeHistory?.ID === history.ID ? 'active' : ''}`}
                      onClick={() => setActiveHistoryId(history.ID)}
                    >
                      <span className="ats-history-name">{history.historyName || '未命名方案'}</span>
                      <span className="ats-history-count">{history.tableCount} 张表</span>
                    </button>
                  ))
                )}
              </aside>

              <section className="ats-history-detail">
                {activeHistory ? (
                  <>
                    <div className="ats-history-detail-head">
                      <div>
                        <div className="ats-history-detail-title">{activeHistory.historyName || '未命名方案'}</div>
                        <div className="ats-history-detail-subtitle">{formatHistoryTime(activeHistory.UpdatedAt || activeHistory.CreatedAt)}</div>
                        <div className="ats-history-detail-count">{activeHistory.tableCount} 张表</div>
                      </div>
                      <button
                        type="button"
                        className="ats-button ats-button-primary"
                        onClick={applyActiveHistory}
                        disabled={historyApplying}
                      >
                        {historyApplying ? <Loader2 size={16} className="ats-spin" /> : null}
                        进入编辑
                      </button>
                    </div>
                    <div className="ats-history-table-list">
                      {activeHistoryTables.length === 0 ? (
                        <div className="ats-history-state">这个业务方案没有表</div>
                      ) : (
                        activeHistoryTables.map(table => (
                          <div
                            key={`${table.databaseName}:${table.tableName}`}
                            className="ats-history-table-item"
                            title={[`${table.databaseName}.${table.tableName}`, table.tableComment].filter(Boolean).join('\n')}
                          >
                            <span className="ats-history-table-name">{table.tableName}</span>
                            <span className="ats-history-table-db">{table.databaseName}</span>
                            {table.tableComment ? (
                              <span className="ats-history-table-comment">{table.tableComment}</span>
                            ) : null}
                          </div>
                        ))
                      )}
                    </div>
                  </>
                ) : (
                  <div className="ats-history-state">请选择左侧业务方案</div>
                )}
              </section>
            </div>
          </div>
        </div>
      )}

      {nameDialogOpen && (
        <div className="ats-name-overlay" onClick={cancelNamedSelection}>
          <div className="ats-name-modal" onClick={event => event.stopPropagation()}>
            <div className="ats-name-header">
              <div className="ats-name-title">保存业务方案</div>
              <button
                type="button"
                className="ats-icon-button ats-history-close"
                onClick={cancelNamedSelection}
                title="关闭"
              >
                <X size={18} />
              </button>
            </div>
            <div className="ats-name-body">
              <label className="ats-name-label" htmlFor="ats-history-name-input">业务名称</label>
              <input
                id="ats-history-name-input"
                className="ats-name-input"
                value={historyNameInput}
                onChange={event => setHistoryNameInput(event.target.value)}
                onKeyDown={event => {
                  if (event.key === 'Enter') {
                    event.preventDefault();
                    void confirmNamedSelection();
                  }
                }}
                autoFocus
                placeholder="例如：用户画像、订单链路、报表核对"
              />
              <div className="ats-name-hint">本次选择 {pendingSelectionTables.length} 张表；同名业务方案会覆盖，不会生成多条记录。</div>
            </div>
            <div className="ats-name-actions">
              <button
                type="button"
                className="ats-button ats-button-secondary"
                onClick={cancelNamedSelection}
              >
                取消
              </button>
              <button
                type="button"
                className="ats-button ats-button-primary"
                onClick={confirmNamedSelection}
                disabled={saving}
              >
                {saving ? <Loader2 size={16} className="ats-spin" /> : null}
                保存方案
              </button>
            </div>
          </div>
        </div>
      )}

      <DatabaseBrowser
        open={browserOpen}
        onClose={() => setBrowserOpen(false)}
        environment={activeConnection?.envName || ''}
        environments={environments}
        onEnvironmentChange={() => undefined}
        projectId={activeProjectId || ''}
        focusedConnectionId={activeConnectionId || undefined}
        selectionMode="multiple"
        selectedTables={browserSelectedTables}
        onTablesConfirm={handleBrowserTablesConfirm}
        autoClose={false}
      />
    </div>
  );
}
