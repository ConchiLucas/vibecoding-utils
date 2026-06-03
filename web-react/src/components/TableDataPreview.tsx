import React, { useState, useEffect, useCallback, useMemo } from 'react';
import './TableDataPreview.css';
import {
  X,
  Loader2,
  ChevronUp,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Database,
  FileSpreadsheet,
  Pencil,
  Save,
  RotateCcw,
  AlertTriangle,
  Filter,
  Code2,
  Copy,
  Eye,
  Sparkles,
} from 'lucide-react';
import {
  getRemoteTablePreview,
  getRemoteTablePage,
  updateRemoteTableRecord,
  getRemoteTableDDL,
  generateRemoteTableData,
  ColumnPreview,
  TableDataColumn,
  TableDataRow,
  TableDataCell,
} from '@/api/sysConnection';
import toast from 'react-hot-toast';

interface TableDataPreviewProps {
  open: boolean;
  onClose: () => void;
  connectionId: number;
  databaseName: string;
  tableName: string;
}

interface PendingChange {
  name: string;
  before: string;
  after: string;
  primaryKey: boolean;
}

interface ActiveFilter {
  column: string;
  value: string;
}

const DEFAULT_PAGE_SIZE = 20;
const PAGE_SIZE_OPTIONS = [20, 50, 100];
const MAX_GENERATE_COUNT = 50;

function getRequestErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object') {
    const anyError = error as any;
    const data = anyError.response?.data;
    if (data?.msg) return data.msg;
    if (typeof data === 'string' && data.trim()) {
      return `接口没有返回标准 JSON：${data.trim().slice(0, 160)}`;
    }
    if (anyError.message) return anyError.message;
  }
  return fallback;
}

export default function TableDataPreview({
  open,
  onClose,
  connectionId,
  databaseName,
  tableName,
}: TableDataPreviewProps) {
  const [tableColumns, setTableColumns] = useState<TableDataColumn[]>([]);
  const [rows, setRows] = useState<TableDataRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  const [detailOpen, setDetailOpen] = useState(false);
  const [columns, setColumns] = useState<ColumnPreview[]>([]);
  const [offset, setOffset] = useState(0);
  const [detailLoading, setDetailLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [draftValues, setDraftValues] = useState<Record<string, string>>({});

  const [activeFilterColumn, setActiveFilterColumn] = useState('');
  const [filterValue, setFilterValue] = useState('');
  const [appliedFilter, setAppliedFilter] = useState<ActiveFilter | null>(null);

  const [ddlOpen, setDdlOpen] = useState(false);
  const [ddlLoading, setDdlLoading] = useState(false);
  const [ddlSQL, setDdlSQL] = useState('');
  const [ddlError, setDdlError] = useState('');
  const [generateOpen, setGenerateOpen] = useState(false);
  const [generateCount, setGenerateCount] = useState('10');
  const [generating, setGenerating] = useState(false);

  const resetDraftValues = useCallback((nextColumns: ColumnPreview[]) => {
    const nextDraft: Record<string, string> = {};
    nextColumns.forEach(col => {
      nextDraft[col.name] = col.value ?? '';
    });
    setDraftValues(nextDraft);
  }, []);

  const fetchPage = useCallback(async (targetPage: number, targetPageSize: number, filter: ActiveFilter | null) => {
    setLoading(true);
    try {
      const res = await getRemoteTablePage({
        ID: connectionId,
        databaseName,
        tableName,
        page: targetPage,
        pageSize: targetPageSize,
        ...(filter ? { filterColumn: filter.column, filterValue: filter.value } : {}),
      });
      if (res.code === 0 && res.data) {
        setTableColumns(res.data.columns || []);
        setRows(res.data.rows || []);
        setTotal(res.data.total || 0);
        setPage(res.data.page || targetPage);
        setPageSize(res.data.pageSize || targetPageSize);
      } else {
        toast.error(res.msg || '获取表数据失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('获取表数据失败');
    } finally {
      setLoading(false);
    }
  }, [connectionId, databaseName, tableName]);

  const fetchRecord = useCallback(async (newOffset: number, filter: ActiveFilter | null) => {
    setDetailLoading(true);
    try {
      const res = await getRemoteTablePreview({
        ID: connectionId,
        databaseName,
        tableName,
        offset: newOffset,
        ...(filter ? { filterColumn: filter.column, filterValue: filter.value } : {}),
      });
      if (res.code === 0 && res.data) {
        const nextColumns = res.data.columns || [];
        setColumns(nextColumns);
        setTotal(res.data.total);
        setOffset(res.data.offset);
        setEditing(false);
        setConfirmOpen(false);
        setDetailOpen(true);
        resetDraftValues(nextColumns);
      } else {
        toast.error(res.msg || '获取记录失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('获取记录失败');
    } finally {
      setDetailLoading(false);
    }
  }, [connectionId, databaseName, tableName, resetDraftValues]);

  useEffect(() => {
    if (open && connectionId && tableName) {
      setTableColumns([]);
      setRows([]);
      setTotal(0);
      setPage(1);
      setPageSize(DEFAULT_PAGE_SIZE);
      setDetailOpen(false);
      setColumns([]);
      setOffset(0);
      setEditing(false);
      setConfirmOpen(false);
      setDraftValues({});
      setActiveFilterColumn('');
      setFilterValue('');
      setAppliedFilter(null);
      setDdlOpen(false);
      setDdlSQL('');
      setDdlError('');
      setGenerateOpen(false);
      setGenerateCount('10');
      setGenerating(false);
      void fetchPage(1, DEFAULT_PAGE_SIZE, null);
    }
  }, [open, connectionId, databaseName, tableName, fetchPage]);

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / pageSize)), [total, pageSize]);
  const pageStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const pageEnd = total === 0 ? 0 : Math.min(total, page * pageSize);

  const pendingChanges = useMemo(() => {
    return columns
      .map(col => {
        const nextValue = draftValues[col.name] ?? '';
        const currentValue = col.value ?? '';
        if (nextValue === currentValue) return null;
        return {
          name: col.name,
          before: col.isNull ? 'NULL' : currentValue === '' ? "''" : currentValue,
          after: nextValue === '' ? "''" : nextValue,
          primaryKey: Boolean(col.primaryKey),
        };
      })
      .filter((change): change is PendingChange => Boolean(change));
  }, [columns, draftValues]);

  const hasChanges = pendingChanges.length > 0;

  const handleCloseDetail = useCallback(() => {
    if (saving) return;
    if (editing && hasChanges) {
      toast.error('请先保存或取消当前修改');
      return;
    }
    setDetailOpen(false);
    setEditing(false);
    setConfirmOpen(false);
  }, [editing, hasChanges, saving]);

  const handleCloseAll = () => {
    if (generating) {
      toast.error('造数进行中，请稍等');
      return;
    }
    if (detailOpen && editing && hasChanges) {
      toast.error('请先保存或取消当前修改');
      return;
    }
    onClose();
  };

  useEffect(() => {
    if (!open) return;
    const handleKey = (e: KeyboardEvent) => {
      if (confirmOpen) {
        if (e.key === 'Escape') {
          setConfirmOpen(false);
        }
        return;
      }
      if (generateOpen) {
        if (e.key === 'Escape' && !generating) {
          setGenerateOpen(false);
        }
        return;
      }
      if (ddlOpen) {
        if (e.key === 'Escape') {
          setDdlOpen(false);
        }
        return;
      }
      if (detailOpen) {
        if (editing) {
          if (e.key === 'Escape') {
            e.preventDefault();
            setEditing(false);
            resetDraftValues(columns);
          }
          return;
        }
        if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
          e.preventDefault();
          if (offset > 0) {
            void fetchRecord(offset - 1, appliedFilter);
          }
        } else if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
          e.preventDefault();
          if (offset < total - 1) {
            void fetchRecord(offset + 1, appliedFilter);
          }
        } else if (e.key === 'Escape') {
          handleCloseDetail();
        }
        return;
      }
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        if (page > 1) {
          void fetchPage(page - 1, pageSize, appliedFilter);
        }
      } else if (e.key === 'ArrowRight') {
        e.preventDefault();
        if (page < totalPages) {
          void fetchPage(page + 1, pageSize, appliedFilter);
        }
      } else if (e.key === 'Escape') {
        handleCloseAll();
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [
    open,
    page,
    pageSize,
    total,
    totalPages,
    fetchPage,
    detailOpen,
    offset,
    fetchRecord,
    editing,
    confirmOpen,
    generateOpen,
    generating,
    ddlOpen,
    columns,
    resetDraftValues,
    appliedFilter,
    handleCloseDetail,
  ]);

  const handleStartEdit = () => {
    resetDraftValues(columns);
    setEditing(true);
  };

  const handleCancelEdit = () => {
    resetDraftValues(columns);
    setEditing(false);
    setConfirmOpen(false);
  };

  const handleSaveClick = () => {
    if (!hasChanges) {
      toast.error('没有需要保存的修改');
      return;
    }
    setConfirmOpen(true);
  };

  const handleConfirmSave = async () => {
    setSaving(true);
    try {
      const res = await updateRemoteTableRecord({
        ID: connectionId,
        databaseName,
        tableName,
        offset,
        ...(appliedFilter ? { filterColumn: appliedFilter.column, filterValue: appliedFilter.value } : {}),
        changes: pendingChanges.map(change => ({
          name: change.name,
          value: draftValues[change.name] ?? '',
        })),
      });
      if (res.code === 0 && res.data) {
        const nextColumns = res.data.columns || [];
        setColumns(nextColumns);
        setTotal(res.data.total);
        setOffset(res.data.offset);
        resetDraftValues(nextColumns);
        setEditing(false);
        setConfirmOpen(false);
        toast.success(res.msg || '修改成功');
        void fetchPage(page, pageSize, appliedFilter);
      } else {
        toast.error(res.msg || '修改失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('修改失败');
    } finally {
      setSaving(false);
    }
  };

  const renderDetailValue = (col: ColumnPreview) => {
    if (col.isNull) return 'NULL';
    if (col.value === '') return "''";
    return col.value;
  };

  const renderCellValue = (cell?: TableDataCell) => {
    if (!cell || cell.isNull) return 'NULL';
    if (cell.value === '') return "''";
    return cell.value;
  };

  const handleFilterColumnClick = (column: string) => {
    if (editing && hasChanges) {
      toast.error('请先保存或取消当前修改');
      return;
    }
    setActiveFilterColumn(column);
    setFilterValue(appliedFilter?.column === column ? appliedFilter.value : '');
  };

  const handleApplyFilter = () => {
    if (!activeFilterColumn) return;
    const nextFilter = { column: activeFilterColumn, value: filterValue };
    setAppliedFilter(nextFilter);
    setDetailOpen(false);
    void fetchPage(1, pageSize, nextFilter);
  };

  const handleClearFilter = () => {
    setAppliedFilter(null);
    setActiveFilterColumn('');
    setFilterValue('');
    setDetailOpen(false);
    void fetchPage(1, pageSize, null);
  };

  const handlePageChange = (nextPage: number) => {
    const clamped = Math.min(Math.max(nextPage, 1), totalPages);
    if (clamped === page || loading) return;
    void fetchPage(clamped, pageSize, appliedFilter);
  };

  const handlePageSizeChange = (nextPageSize: number) => {
    if (nextPageSize === pageSize || loading) return;
    void fetchPage(1, nextPageSize, appliedFilter);
  };

  const handleOpenDetail = (row: TableDataRow) => {
    if (editing && hasChanges) {
      toast.error('请先保存或取消当前修改');
      return;
    }
    setDetailOpen(true);
    setColumns([]);
    setOffset(row.offset);
    void fetchRecord(row.offset, appliedFilter);
  };

  const handleOpenDDL = async () => {
    setDdlOpen(true);
    setDdlError('');
    if (ddlSQL) return;

    setDdlLoading(true);
    try {
      const res = await getRemoteTableDDL({
        ID: connectionId,
        databaseName,
        tableName,
      });
      if (res.code === 0 && res.data?.sql) {
        setDdlSQL(res.data.sql);
      } else {
        setDdlError(res.msg || '获取建表 SQL 失败');
      }
    } catch (e) {
      console.error(e);
      setDdlError('获取建表 SQL 失败');
    } finally {
      setDdlLoading(false);
    }
  };

  const handleCopyDDL = async () => {
    if (!ddlSQL) return;
    try {
      await navigator.clipboard.writeText(ddlSQL);
      toast.success('已复制建表 SQL');
    } catch (e) {
      console.error(e);
      toast.error('复制失败');
    }
  };

  const handleOpenGenerate = () => {
    setGenerateCount('10');
    setGenerateOpen(true);
  };

  const handleConfirmGenerate = async () => {
    const count = Number(generateCount);
    if (!Number.isInteger(count) || count < 1 || count > MAX_GENERATE_COUNT) {
      toast.error(`请输入 1-${MAX_GENERATE_COUNT} 之间的整数`);
      return;
    }

    setGenerating(true);
    try {
      const res = await generateRemoteTableData({
        ID: connectionId,
        databaseName,
        tableName,
        count,
      });
      if (res.code === 0) {
        toast.success(res.msg || `造数成功，已插入 ${res.data?.inserted || count} 条`);
        setGenerateOpen(false);
        void fetchPage(page, pageSize, appliedFilter);
      } else {
        const responseData = (res as any).data;
        const message = res.msg || (typeof responseData === 'string' ? `接口没有返回标准 JSON：${responseData.trim().slice(0, 160)}` : '造数失败');
        toast.error(message);
      }
    } catch (e) {
      console.error(e);
      toast.error(getRequestErrorMessage(e, '造数失败'));
    } finally {
      setGenerating(false);
    }
  };

  const renderDetailModal = () => {
    if (!detailOpen) return null;

    return (
      <div
        className="tdp-detail-layer"
        onClick={e => {
          e.stopPropagation();
          handleCloseDetail();
        }}
      >
        <div className="tdp-detail-modal" onClick={e => e.stopPropagation()}>
          <div className="tdp-header">
            <div className="tdp-header-left">
              <h2 className="tdp-title">
                <FileSpreadsheet size={18} className="tdp-title-icon" />
                数据预览
              </h2>
              <span className="tdp-subtitle">{databaseName} → {tableName}</span>
            </div>
            <button className="tdp-close" onClick={handleCloseDetail}>
              <X size={18} />
            </button>
          </div>

          <div className="tdp-nav">
            <span className="tdp-nav-info">
              {total > 0 ? (
                <>第 <strong>{offset + 1}</strong> / <strong>{total}</strong> 条记录</>
              ) : (
                '暂无记录'
              )}
              {editing && <em>编辑中</em>}
              {appliedFilter && <em>{appliedFilter.column} = {appliedFilter.value || "''"}</em>}
            </span>
            <div className="tdp-nav-actions">
              {editing ? (
                <div className="tdp-edit-actions">
                  <button
                    className="tdp-action-btn tdp-action-primary"
                    disabled={detailLoading || saving || !hasChanges}
                    onClick={handleSaveClick}
                    title="保存修改"
                  >
                    <Save size={16} />
                    保存
                  </button>
                  <button
                    className="tdp-action-btn"
                    disabled={saving}
                    onClick={handleCancelEdit}
                    title="取消编辑"
                  >
                    <RotateCcw size={16} />
                    取消
                  </button>
                </div>
              ) : (
                <>
                  <button
                    className="tdp-action-btn"
                    disabled={detailLoading}
                    onClick={handleOpenDDL}
                    title="查看建表 SQL"
                  >
                    <Code2 size={16} />
                    建表 SQL
                  </button>
                  <button
                    className="tdp-action-btn"
                    disabled={detailLoading || columns.length === 0 || total === 0}
                    onClick={handleStartEdit}
                    title="编辑字段值"
                  >
                    <Pencil size={16} />
                    编辑
                  </button>
                </>
              )}
              <div className="tdp-nav-buttons">
                <button
                  className="tdp-nav-btn"
                  disabled={detailLoading || editing || offset <= 0}
                  onClick={() => void fetchRecord(offset - 1, appliedFilter)}
                  title="上一条"
                >
                  <ChevronUp size={18} />
                </button>
                <button
                  className="tdp-nav-btn"
                  disabled={detailLoading || editing || offset >= total - 1}
                  onClick={() => void fetchRecord(offset + 1, appliedFilter)}
                  title="下一条"
                >
                  <ChevronDown size={18} />
                </button>
              </div>
            </div>
          </div>

          <div className="tdp-body">
            {detailLoading ? (
              <div className="tdp-loading">
                <Loader2 size={24} className="tdp-spinner" />
                <span>加载中...</span>
              </div>
            ) : columns.length === 0 ? (
              <div className="tdp-empty">
                <Database size={36} className="tdp-empty-icon" />
                <span>暂无数据</span>
              </div>
            ) : (
              <table className="tdp-table">
                <thead>
                  <tr>
                    <th>字段名</th>
                    <th>字段值</th>
                    <th>字段注释</th>
                  </tr>
                </thead>
                <tbody>
                  {columns.map((col, idx) => (
                    <tr key={idx}>
                      <td className="tdp-col-name">
                        <div className="tdp-col-name-inner">
                          <span className="tdp-col-name-text">{col.name}</span>
                          {col.primaryKey && <b>PK</b>}
                        </div>
                      </td>
                      <td className={`tdp-col-value ${col.isNull ? 'tdp-col-value-empty' : ''}`}>
                        {editing ? (
                          <textarea
                            className="tdp-value-input"
                            value={draftValues[col.name] ?? ''}
                            placeholder={col.isNull ? 'NULL' : ''}
                            spellCheck={false}
                            onChange={e => {
                              const value = e.target.value;
                              setDraftValues(prev => ({ ...prev, [col.name]: value }));
                            }}
                          />
                        ) : (
                          renderDetailValue(col)
                        )}
                      </td>
                      <td className={`tdp-col-desc ${!col.description ? 'tdp-col-desc-empty' : ''}`}>
                        {col.description || '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    );
  };

  if (!open) return null;

  return (
    <div className="tdp-overlay" onClick={handleCloseAll}>
      <div className="tdp-modal tdp-modal-fullscreen" onClick={e => e.stopPropagation()}>
        <div className="tdp-header">
          <div className="tdp-header-left">
            <h2 className="tdp-title">
              <FileSpreadsheet size={18} className="tdp-title-icon" />
              数据表数据
            </h2>
            <span className="tdp-subtitle">{databaseName} → {tableName}</span>
          </div>
          <button className="tdp-close" onClick={handleCloseAll}>
            <X size={18} />
          </button>
        </div>

        <div className="tdp-nav tdp-list-nav">
          <span className="tdp-nav-info">
            {total > 0 ? (
              <>第 <strong>{pageStart}</strong> - <strong>{pageEnd}</strong> / <strong>{total}</strong> 条记录</>
            ) : (
              '暂无记录'
            )}
            <em>{tableColumns.length} 个字段</em>
            {appliedFilter && <em>{appliedFilter.column} = {appliedFilter.value || "''"}</em>}
          </span>
          <div className="tdp-nav-actions">
            {activeFilterColumn && (
              <form
                className="tdp-filter-form"
                onSubmit={e => {
                  e.preventDefault();
                  handleApplyFilter();
                }}
              >
                <span className="tdp-filter-label">
                  <Filter size={14} />
                  {activeFilterColumn}
                </span>
                <input
                  className="tdp-filter-input"
                  value={filterValue}
                  placeholder="输入过滤值"
                  spellCheck={false}
                  onChange={e => setFilterValue(e.target.value)}
                />
                <button className="tdp-filter-btn" type="submit" disabled={loading || saving}>
                  过滤
                </button>
                <button className="tdp-filter-clear" type="button" disabled={loading || saving} onClick={handleClearFilter} title="清除过滤">
                  <X size={14} />
                </button>
              </form>
            )}
            <button
              className="tdp-action-btn"
              disabled={loading}
              onClick={handleOpenDDL}
              title="查看建表 SQL"
            >
              <Code2 size={16} />
              建表 SQL
            </button>
            <button
              className="tdp-action-btn tdp-action-primary"
              disabled={loading || generating}
              onClick={handleOpenGenerate}
              title="调用默认 AI 造数"
            >
              {generating ? <Loader2 size={16} className="tdp-spinner" /> : <Sparkles size={16} />}
              造数
            </button>
          </div>
        </div>

        <div className="tdp-list-body">
          {loading ? (
            <div className="tdp-loading">
              <Loader2 size={24} className="tdp-spinner" />
              <span>加载中...</span>
            </div>
          ) : tableColumns.length === 0 ? (
            <div className="tdp-empty">
              <Database size={36} className="tdp-empty-icon" />
              <span>暂无数据</span>
            </div>
          ) : (
            <table className="tdp-data-grid">
              <thead>
                <tr>
                  <th className="tdp-row-index-col">#</th>
                  {tableColumns.map(col => (
                    <th key={col.name} className="tdp-data-col-head">
                      <button
                        type="button"
                        className={`tdp-data-col-btn ${activeFilterColumn === col.name ? 'tdp-data-col-btn-active' : ''}`}
                        onClick={() => handleFilterColumnClick(col.name)}
                        title={col.description ? `${col.name}\n${col.description}\n点击按该字段过滤` : `${col.name}\n点击按该字段过滤`}
                      >
                        <span>{col.name}</span>
                        {col.primaryKey && <b>PK</b>}
                      </button>
                      {col.description ? <small>{col.description}</small> : null}
                    </th>
                  ))}
                  <th className="tdp-action-col">操作</th>
                </tr>
              </thead>
              <tbody>
                {rows.length === 0 ? (
                  <tr>
                    <td className="tdp-grid-empty-row" colSpan={tableColumns.length + 2}>
                      暂无匹配数据
                    </td>
                  </tr>
                ) : rows.map(row => (
                  <tr key={row.offset}>
                    <td className="tdp-row-index-col">{row.offset + 1}</td>
                    {tableColumns.map((col, colIdx) => {
                      const cell = row.cells[colIdx];
                      const value = renderCellValue(cell);
                      return (
                        <td
                          key={`${row.offset}:${col.name}`}
                          className={`tdp-list-cell ${cell?.isNull ? 'tdp-list-cell-empty' : ''}`}
                          title={value}
                        >
                          {value}
                        </td>
                      );
                    })}
                    <td className="tdp-action-col">
                      <button
                        type="button"
                        className="tdp-row-action-btn"
                        onClick={() => handleOpenDetail(row)}
                        title="查看详情"
                      >
                        <Eye size={15} />
                        查看
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="tdp-pagination">
          <div className="tdp-pagination-info">
            <span>每页</span>
            <select
              value={pageSize}
              disabled={loading}
              onChange={e => handlePageSizeChange(Number(e.target.value))}
            >
              {PAGE_SIZE_OPTIONS.map(size => (
                <option key={size} value={size}>{size}</option>
              ))}
            </select>
            <span>条</span>
          </div>
          <div className="tdp-pagination-actions">
            <button className="tdp-nav-btn" disabled={loading || page <= 1} onClick={() => handlePageChange(1)} title="第一页">
              <ChevronsLeft size={18} />
            </button>
            <button className="tdp-nav-btn" disabled={loading || page <= 1} onClick={() => handlePageChange(page - 1)} title="上一页">
              <ChevronLeft size={18} />
            </button>
            <span className="tdp-page-indicator">
              <strong>{page}</strong>
              <i>/</i>
              <span>{totalPages}</span>
            </span>
            <button className="tdp-nav-btn" disabled={loading || page >= totalPages} onClick={() => handlePageChange(page + 1)} title="下一页">
              <ChevronRight size={18} />
            </button>
            <button className="tdp-nav-btn" disabled={loading || page >= totalPages} onClick={() => handlePageChange(totalPages)} title="最后一页">
              <ChevronsRight size={18} />
            </button>
          </div>
        </div>
      </div>

      {renderDetailModal()}

      {confirmOpen && (
        <div
          className="tdp-confirm-layer"
          onClick={e => {
            e.stopPropagation();
            if (!saving) setConfirmOpen(false);
          }}
        >
          <div className="tdp-confirm" onClick={e => e.stopPropagation()}>
            <div className="tdp-confirm-head">
              <span className="tdp-confirm-icon">
                <AlertTriangle size={20} />
              </span>
              <div>
                <h3>确认修改字段值</h3>
                <p>{databaseName} → {tableName}，第 {offset + 1} 条记录</p>
              </div>
            </div>
            <div className="tdp-confirm-body">
              <div className="tdp-confirm-summary">
                本次将修改 <strong>{pendingChanges.length}</strong> 个字段，提交后会直接写入远程数据库。
              </div>
              <div className="tdp-change-list">
                {pendingChanges.slice(0, 6).map(change => (
                  <div className="tdp-change-item" key={change.name}>
                    <div className="tdp-change-name">
                      <span>{change.name}</span>
                      {change.primaryKey && <b>PK</b>}
                    </div>
                    <div className="tdp-change-values">
                      <span>{change.before}</span>
                      <i>→</i>
                      <span>{change.after}</span>
                    </div>
                  </div>
                ))}
                {pendingChanges.length > 6 && (
                  <div className="tdp-change-more">还有 {pendingChanges.length - 6} 个字段未展开</div>
                )}
              </div>
            </div>
            <div className="tdp-confirm-actions">
              <button className="tdp-action-btn" disabled={saving} onClick={() => setConfirmOpen(false)}>
                取消
              </button>
              <button className="tdp-action-btn tdp-action-danger" disabled={saving} onClick={handleConfirmSave}>
                {saving ? <Loader2 size={16} className="tdp-spinner" /> : <Save size={16} />}
                确认保存
              </button>
            </div>
          </div>
        </div>
      )}

      {generateOpen && (
        <div
          className="tdp-confirm-layer"
          onClick={e => {
            e.stopPropagation();
            if (!generating) setGenerateOpen(false);
          }}
        >
          <div className="tdp-confirm tdp-generate-confirm" onClick={e => e.stopPropagation()}>
            <div className="tdp-confirm-head">
              <span className="tdp-confirm-icon tdp-generate-icon">
                <Sparkles size={20} />
              </span>
              <div>
                <h3>AI 造数</h3>
                <p>{databaseName} → {tableName}</p>
              </div>
            </div>
            <div className="tdp-confirm-body">
              <div className="tdp-generate-form">
                <label htmlFor="tdp-generate-count">数量</label>
                <input
                  id="tdp-generate-count"
                  className="tdp-generate-input"
                  type="number"
                  min={1}
                  max={MAX_GENERATE_COUNT}
                  step={1}
                  value={generateCount}
                  disabled={generating}
                  autoFocus
                  onChange={e => setGenerateCount(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      void handleConfirmGenerate();
                    }
                  }}
                />
              </div>
              <div className="tdp-generate-tip">
                将使用当前默认 AI 配置生成测试数据，并直接写入远程数据库；单次最多 {MAX_GENERATE_COUNT} 条。
              </div>
            </div>
            <div className="tdp-confirm-actions">
              <button className="tdp-action-btn" disabled={generating} onClick={() => setGenerateOpen(false)}>
                取消
              </button>
              <button className="tdp-action-btn tdp-action-primary" disabled={generating} onClick={handleConfirmGenerate}>
                {generating ? <Loader2 size={16} className="tdp-spinner" /> : <Sparkles size={16} />}
                确认造数
              </button>
            </div>
          </div>
        </div>
      )}

      {ddlOpen && (
        <div
          className="tdp-ddl-layer"
          onClick={e => {
            e.stopPropagation();
            if (!ddlLoading) setDdlOpen(false);
          }}
        >
          <div className="tdp-ddl" onClick={e => e.stopPropagation()}>
            <div className="tdp-ddl-head">
              <div>
                <h3>
                  <Code2 size={18} />
                  建表 SQL
                </h3>
                <p>{databaseName} → {tableName}</p>
              </div>
              <div className="tdp-ddl-actions">
                <button className="tdp-action-btn" disabled={!ddlSQL || ddlLoading} onClick={handleCopyDDL}>
                  <Copy size={16} />
                  复制
                </button>
                <button className="tdp-close" disabled={ddlLoading} onClick={() => setDdlOpen(false)} title="关闭">
                  <X size={18} />
                </button>
              </div>
            </div>
            <div className="tdp-ddl-body">
              {ddlLoading ? (
                <div className="tdp-ddl-empty">
                  <Loader2 size={22} className="tdp-spinner" />
                  <span>正在读取建表 SQL...</span>
                </div>
              ) : ddlError ? (
                <div className="tdp-ddl-empty tdp-ddl-error">{ddlError}</div>
              ) : (
                <pre className="tdp-ddl-code">{ddlSQL}</pre>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
