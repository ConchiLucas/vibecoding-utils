import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Link, GitMerge, Trash2, LayoutGrid, Database } from 'lucide-react';
import { getTbTableRelateList, deleteTbTableRelate, TbTableRelate, createTbTableRelate, getTableComments } from '../../api/sysRelate';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';
import DatabaseBrowser from '../../components/DatabaseBrowser';

import { getTbConnectionList, TbConnection } from '../../api/sysConnection';
import { getRemoteColumns, ClientColumnVO } from '../../api/sysRelate';
import { Search } from 'lucide-react';
import { useProjectStore } from '../../stores/useProjectStore';
import { resolveSelectedConnectionId } from '../config-manager/ConfigManagerSelection';

export default function RelateManager() {
  const [relates, setRelates] = useState<TbTableRelate[]>([]);
  const [loading, setLoading] = useState(true);
  const [page] = useState(1);
  const { confirm, dialogProps } = useConfirm();
  const { activeProjectId, activeConnectionId, setActiveConnectionId } = useProjectStore();

  // Database scope follows the globally selected database connection.
  const [connectionOptions, setConnectionOptions] = useState<TbConnection[]>([]);
  const [envOptions, setEnvOptions] = useState<string[]>([]);

  // Dual-pane builder state
  const [sourceDbStr, setSourceDbStr] = useState('');
  const [targetDbStr, setTargetDbStr] = useState('');
  
  const [sourceColumns, setSourceColumns] = useState<ClientColumnVO[]>([]);
  const [targetColumns, setTargetColumns] = useState<ClientColumnVO[]>([]);
  const [isRetrievingSource, setIsRetrievingSource] = useState(false);
  const [isRetrievingTarget, setIsRetrievingTarget] = useState(false);

  const [sourceCol, setSourceCol] = useState<ClientColumnVO | null>(null);
  const [targetCol, setTargetCol] = useState<ClientColumnVO | null>(null);

  const [showSourceBrowser, setShowSourceBrowser] = useState(false);
  const [showTargetBrowser, setShowTargetBrowser] = useState(false);
  const [lineageFilter, setLineageFilter] = useState('');
  const [tableComments, setTableComments] = useState<Record<string, string>>({});

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
    setSourceColumns([]);
    setTargetColumns([]);
    setSourceCol(null);
    setTargetCol(null);
  }, [activeProjectId]);

  const selectedConnection = connectionOptions.find(conn => conn.ID === activeConnectionId) || null;
  const environment = selectedConnection?.envName || '';

  // Fetch table comments when relates or environment change
  useEffect(() => {
    if (!activeProjectId || !selectedConnection || relates.length === 0) {
      setTableComments({});
      return;
    }
    const uniqueTables = new Set<string>();
    for (const r of relates) {
      if (r.databaseName && r.tableName) uniqueTables.add(`${r.databaseName}:${r.tableName}`);
      if (r.relateDatabaseName && r.relateTableName) uniqueTables.add(`${r.relateDatabaseName}:${r.relateTableName}`);
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
      setEnvOptions([]);
      return;
    }
    try {
      const res = await getTbConnectionList({ page: 1, pageSize: 999, connectionGroup: String(activeProjectId) });
      if (res.code === 0) {
        const list = res.data.list || [];
        const envs = Array.from(new Set(list.map((c: TbConnection) => c.envName))).filter(Boolean) as string[];
        setConnectionOptions(list);
        setEnvOptions(envs);
        if (list.length === 0) {
          setActiveConnectionId(null);
        }
      }
    } catch (e) {
      console.error(e);
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

  async function handleRetrieveSource() {
    if (!activeProjectId || !selectedConnection || !sourceDbStr) return toast.error('请选择项目和数据库配置，并输入源数据表名');
    setIsRetrievingSource(true);
    try {
      const res = await getRemoteColumns({
        environment,
        databaseStr: sourceDbStr,
        projectConfigId: activeProjectId,
        connectionId: selectedConnection.ID,
      });
      if (res.code === 0 && res.data) setSourceColumns(res.data);
      else toast.error(res.msg || '检索源表字段为空');
    } catch {
       toast.error('检索失败');
    } finally {
      setIsRetrievingSource(false);
    }
  }

  async function handleRetrieveTarget() {
    if (!activeProjectId || !selectedConnection || !targetDbStr) return toast.error('请选择项目和数据库配置，并输入目标映射表名');
    setIsRetrievingTarget(true);
    try {
      const res = await getRemoteColumns({
        environment,
        databaseStr: targetDbStr,
        projectConfigId: activeProjectId,
        connectionId: selectedConnection.ID,
      });
      if (res.code === 0 && res.data) setTargetColumns(res.data);
      else toast.error(res.msg || '检索目标表字段为空');
    } catch {
       toast.error('检索失败');
    } finally {
      setIsRetrievingTarget(false);
    }
  }

  async function handleDelete(rel: TbTableRelate) {
    const confirmed = await confirm(`确定要断开此关联关系吗？`, {
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

  async function handleSubmitRelate() {
     if (!activeProjectId) {
         return toast.error('请先选择一个项目配置');
     }
     if (!sourceDbStr || !sourceCol) {
         return toast.error('原表及源字段必须选择');
     }
     const msg = targetCol 
        ? `确定要绑定 ${sourceDbStr}表的${sourceCol.name} 和 ${targetDbStr}表的${targetCol.name} 吗？`
        : `确定只添加 ${sourceDbStr}表的${sourceCol.name} 作为关键字段吗？`;
        
     const confirmed = await confirm(msg, {
         title: '签发关系绑定任务',
         confirmText: '确认签发'
     });
     if (!confirmed) return;

     try {
         // Parse the browser selection into separate database/table fields before saving.
         const [sDb, sTb] = sourceDbStr.includes(':') ? sourceDbStr.split(':', 2) : ['', sourceDbStr];
         const [tDb, tTb] = targetDbStr && targetDbStr.includes(':') ? targetDbStr.split(':', 2) : ['', targetDbStr || ''];
         
         await createTbTableRelate({
            projectConfigId: activeProjectId,
            databaseName: sDb,
            tableName: sTb,
            columnName: sourceCol.name,
            columnType: sourceCol.columnType || 'varchar',
            relateDatabaseName: targetCol ? tDb : '',
            relateTableName: targetCol ? tTb : '',
            relateColumnName: targetCol ? targetCol.name : '',
            relateColumnType: targetCol ? (targetCol.columnType || 'varchar') : '',
         });
         toast.success('绑定任务签发成功');
         loadData(1);
         setSourceCol(null);
         setTargetCol(null);
         setSourceColumns([]);
         setTargetColumns([]);
     } catch(err) {
         toast.error('签发失败');
     }
  }

  return (
    <div className="space-y-6 max-w-full">

      <div className="bg-white rounded-3xl overflow-hidden border border-slate-200/60 shadow-sm p-8 relative">
          {/* Header */}
          <div className="flex items-center justify-between mb-8 border-b border-slate-100 pb-5">
            <div>
              <h2 className="text-xl font-extrabold text-slate-900 mb-1">表级血缘指配中心</h2>
              <p className="text-xs text-slate-500">建立数据库表与表之间的字段级映射关联，构建清晰的数据血缘图谱</p>
            </div>
            <div className="flex items-center gap-3">
               <div
                 className="flex h-10 w-72 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm text-slate-600 shadow-sm"
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
          </div>

          <div className="grid grid-cols-1 xl:grid-cols-2 gap-8 items-start relative">
             
             {/* Center connector UI */}
             <div className="hidden xl:flex absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-10 w-12 h-12 bg-white rounded-full items-center justify-center border-4 border-slate-50 shadow-sm">
                <Link size={18} className="text-indigo-500" />
             </div>

             {/* Source Table Panel */}
             <div className="bg-slate-50/50 border border-slate-200 rounded-2xl overflow-hidden shadow-sm">
                <div className="p-5 border-b border-slate-200 flex items-center justify-between bg-white">
                   <h3 className="font-bold text-slate-800 flex items-center gap-2 text-sm"><GitMerge size={16} className="text-indigo-500" /> 主对象源表 (Source Index)</h3>
                </div>
                <div className="p-5">
                   <div className="flex items-center gap-2 mb-4">
                      <div className="flex items-center gap-2 bg-white border border-slate-200 shadow-sm rounded-xl p-1 focus-within:border-indigo-400 transition-colors">
                        <div className="flex items-center gap-2 pl-3">
                          <Search size={14} className="text-slate-400" />
                          <input type="text" placeholder="e.g db:table" value={sourceDbStr} onChange={e => setSourceDbStr(e.target.value)} className="bg-transparent border-none text-sm text-slate-700 placeholder-slate-400 focus:ring-0 w-48 outline-none" />
                        </div>
                        <button onClick={handleRetrieveSource} disabled={isRetrievingSource} className="bg-slate-100 hover:bg-slate-200 text-slate-600 text-xs px-4 py-2 rounded-lg transition font-medium border border-slate-200">
                          {isRetrievingSource ? '检索中...' : '检索字段'}
                        </button>
                      </div>
                      <button
                        onClick={() => setShowSourceBrowser(true)}
                        title="打开数据库浏览器"
                        className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-xl shadow-sm hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-600 transition-all"
                      >
                        <LayoutGrid size={14} /> 浏览
                      </button>
                   </div>
                   
                   {/* Source Items Table */}
                   <div className="overflow-x-auto bg-white rounded-xl border border-slate-100">
                     <table className="w-full text-left text-xs min-w-[500px]">
                        <thead className="text-slate-500 border-b border-slate-100 bg-slate-50/80">
                           <tr>
                              <th className="font-semibold py-3 px-2 w-10"></th>
                              <th className="font-semibold py-3 w-1/4">结构字段名</th>
                              <th className="font-semibold py-3 w-20">类型</th>
                              <th className="font-semibold py-3 w-16">长度</th>
                              <th className="font-semibold py-3 px-2">字段注释</th>
                           </tr>
                        </thead>
                        <tbody className="text-slate-600 divide-y divide-slate-50">
                           {sourceColumns.length === 0 ? (
                              <tr><td colSpan={5} className="text-center py-12 text-slate-400">输入源表名并检索字段</td></tr>
                           ) : sourceColumns.map((col, idx) => (
                              <tr key={idx} className={`transition cursor-pointer ${sourceCol?.name === col.name ? 'bg-indigo-50/50' : 'hover:bg-slate-50'}`} onClick={() => setSourceCol(col)}>
                                 <td className="py-3 px-4">
                                    <input type="radio" name="sourceCol" checked={sourceCol?.name === col.name} readOnly className="text-indigo-600 focus:ring-indigo-500 border-slate-300 bg-white" />
                                 </td>
                                 <td className={`py-3 font-medium ${sourceCol?.name === col.name ? 'text-indigo-700' : 'text-slate-700'}`}>{col.name}</td>
                                 <td className="py-3"><span className="px-2 py-0.5 rounded border border-slate-200 bg-slate-100 text-slate-500 font-mono text-[11px]">{col.columnType || 'unknown'}</span></td>
                                 <td className="py-3 text-slate-400">{col.length || '-'}</td>
                                 <td className="py-3 px-2 text-slate-500 truncate max-w-[150px]" title={col.description}>{col.description}</td>
                              </tr>
                           ))}
                        </tbody>
                     </table>
                   </div>
                </div>
             </div>

             {/* Target Table Panel */}
             <div className="bg-slate-50/50 border border-slate-200 rounded-2xl overflow-hidden shadow-sm">
                <div className="p-5 border-b border-slate-200 flex items-center justify-between bg-white">
                   <h3 className="font-bold text-slate-800 flex items-center gap-2 text-sm"><GitMerge size={16} className="text-emerald-500" /> 映射关联表 (Relate Target)</h3>
                </div>
                <div className="p-5">
                   <div className="flex items-center gap-2 mb-4">
                      <div className="flex items-center gap-2 bg-white border border-slate-200 shadow-sm rounded-xl p-1 focus-within:border-emerald-400 transition-colors">
                        <div className="flex items-center gap-2 pl-3">
                          <Search size={14} className="text-slate-400" />
                          <input type="text" placeholder="e.g db:table" value={targetDbStr} onChange={e => setTargetDbStr(e.target.value)} className="bg-transparent border-none text-sm text-slate-700 placeholder-slate-400 focus:ring-0 w-48 outline-none" />
                        </div>
                        <button onClick={handleRetrieveTarget} disabled={isRetrievingTarget} className="bg-slate-100 hover:bg-slate-200 text-slate-600 text-xs px-4 py-2 rounded-lg transition font-medium border border-slate-200">
                          {isRetrievingTarget ? '检索中...' : '检索关联字段'}
                        </button>
                      </div>
                      <button
                        onClick={() => setShowTargetBrowser(true)}
                        title="打开数据库浏览器"
                        className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-xl shadow-sm hover:border-emerald-300 hover:bg-emerald-50 hover:text-emerald-600 transition-all"
                      >
                        <LayoutGrid size={14} /> 浏览
                      </button>
                   </div>
                   
                   {/* Target Items Table */}
                   <div className="overflow-x-auto bg-white rounded-xl border border-slate-100">
                     <table className="w-full text-left text-xs min-w-[500px]">
                        <thead className="text-slate-500 border-b border-slate-100 bg-slate-50/80">
                           <tr>
                              <th className="font-semibold py-3 px-2 w-10"></th>
                              <th className="font-semibold py-3 w-1/4">关联映射主键</th>
                              <th className="font-semibold py-3 w-20">类型</th>
                              <th className="font-semibold py-3 w-16">长度</th>
                              <th className="font-semibold py-3 px-2">字段注释</th>
                           </tr>
                        </thead>
                        <tbody className="text-slate-600 divide-y divide-slate-50">
                           {targetColumns.length === 0 ? (
                              <tr><td colSpan={5} className="text-center py-12 text-slate-400">输入关联表名并检索字段</td></tr>
                           ) : targetColumns.map((col, idx) => (
                              <tr key={idx} className={`transition cursor-pointer ${targetCol?.name === col.name ? 'bg-emerald-50/50' : 'hover:bg-slate-50'}`} onClick={() => setTargetCol(col)}>
                                 <td className="py-3 px-4">
                                    <input type="radio" name="targetCol" checked={targetCol?.name === col.name} readOnly className="text-emerald-500 focus:ring-emerald-500 border-slate-300 bg-white" />
                                 </td>
                                 <td className={`py-3 font-medium ${targetCol?.name === col.name ? 'text-emerald-600' : 'text-slate-700'}`}>{col.name}</td>
                                 <td className="py-3"><span className="px-2 py-0.5 rounded border border-emerald-200 text-emerald-600 bg-emerald-50 font-mono text-[11px]">{col.columnType || 'unknown'}</span></td>
                                 <td className="py-3 text-slate-400">{col.length || '-'}</td>
                                 <td className="py-3 px-2 text-slate-500 truncate max-w-[150px]" title={col.description}>{col.description}</td>
                              </tr>
                           ))}
                        </tbody>
                     </table>
                   </div>
                </div>
             </div>
          </div>

          {/* Submit Action footer */}
          <div className="mt-8 pt-6 border-t border-slate-100 flex flex-col sm:flex-row items-center justify-between gap-4">
             <div className="flex items-center gap-3 text-sm">
                <span className="text-slate-500">主源表：</span>
                <span className={`font-mono font-medium ${sourceCol ? 'text-indigo-600' : 'text-slate-400'}`}>{sourceCol ? `${sourceDbStr}.${sourceCol.name}` : '未选择主体'}</span>
                <span className="text-slate-300 px-2">→</span>
                <span className="text-slate-500">外键目标：</span>
                <span className={`font-mono font-medium ${targetCol ? 'text-emerald-600' : 'text-slate-400'}`}>{targetCol ? `${targetDbStr}.${targetCol.name}` : '(仅独立记录表)'}</span>
             </div>
             <button 
               onClick={handleSubmitRelate}
               className="flex items-center gap-2 px-6 py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-xl shadow-md hover:shadow-lg transition-all"
             >
               <GitMerge size={16} /> 签发关系绑定任务
             </button>
          </div>
      </div>

      <div className="mt-10 px-2">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-bold text-slate-800 flex items-center gap-2"><GitMerge size={20} className="text-indigo-500" /> 已存生血缘图谱 (Saved Lineage Graphs)</h3>
          <div className="flex items-center gap-2 bg-white border border-slate-200 rounded-xl px-3 py-2 shadow-sm focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-100 transition-all w-72">
            <Search size={14} className="text-slate-400 shrink-0" />
            <input
              type="text"
              placeholder="搜索源表或映射目标..."
              value={lineageFilter}
              onChange={e => setLineageFilter(e.target.value)}
              className="bg-transparent border-none text-sm text-slate-700 placeholder-slate-400 outline-none flex-1"
            />
          </div>
        </div>
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {loading ? (
            <div className="col-span-full rounded-2xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">
              加载血缘链路...
            </div>
          ) : (() => {
            const lf = lineageFilter.toLowerCase();
            const filtered = lf ? relates.filter(r =>
              r.tableName?.toLowerCase().includes(lf) ||
              r.columnName?.toLowerCase().includes(lf) ||
              r.relateTableName?.toLowerCase().includes(lf) ||
              r.relateColumnName?.toLowerCase().includes(lf) ||
              r.databaseName?.toLowerCase().includes(lf) ||
              r.relateDatabaseName?.toLowerCase().includes(lf)
            ) : relates;
            return filtered.length === 0 ? (
            <div className="col-span-full rounded-2xl border border-dashed border-slate-200 bg-slate-50 py-12 text-center text-sm text-slate-500">
              {lineageFilter ? `未找到与「${lineageFilter}」匹配的血缘记录` : '暂无配置数据，请通过上方配置项检索并签发'}
            </div>
          ) : filtered.map((rel) => (
            <article key={rel.ID} className="group rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-indigo-200 hover:shadow-md relative overflow-hidden">
               <div className="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-indigo-50 to-transparent pointer-events-none"></div>
               <div className="flex flex-col gap-6 relative z-10">
                   {/* Source */}
                   <div className="flex items-center gap-3">
                       <div className="h-10 w-10 rounded-xl bg-indigo-50 flex items-center justify-center text-indigo-500"><GitMerge size={18} className="-rotate-90"/></div>
                       <div className="flex-1 overflow-hidden">
                           <div className="text-[11px] text-slate-400 font-medium tracking-wide">源表节点</div>
                           <div className="text-sm font-bold truncate text-slate-700 tracking-wide">{rel.tableName}.<span className="text-indigo-600">{rel.columnName}</span></div>
                            {tableComments[`${rel.databaseName}:${rel.tableName}`] && (
                              <div className="text-[11px] text-slate-400 mt-0.5 truncate" title={tableComments[`${rel.databaseName}:${rel.tableName}`]}>{tableComments[`${rel.databaseName}:${rel.tableName}`]}</div>
                            )}
                       </div>
                   </div>

                   {/* Direction Indicator */}
                   <div className="flex flex-col items-center">
                       <div className="w-px h-8 bg-slate-200 relative z-0">
                         <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 px-2 py-0.5 rounded border border-slate-200 bg-white text-[10px] font-mono whitespace-nowrap text-slate-400 z-10 shadow-sm">REFERENCES</div>
                       </div>
                   </div>

                   {/* Target */}
                   <div className="flex items-center gap-3">
                       <div className="h-10 w-10 rounded-xl bg-emerald-50 flex items-center justify-center text-emerald-500"><GitMerge size={18} /></div>
                       <div className="flex-1 overflow-hidden">
                           <div className="text-[11px] text-slate-400 font-medium tracking-wide">映射目标</div>
                           <div className="text-sm font-bold truncate text-slate-700 tracking-wide">{rel.relateTableName}.<span className="text-emerald-600">{rel.relateColumnName}</span></div>
                            {tableComments[`${rel.relateDatabaseName}:${rel.relateTableName}`] && (
                              <div className="text-[11px] text-slate-400 mt-0.5 truncate" title={tableComments[`${rel.relateDatabaseName}:${rel.relateTableName}`]}>{tableComments[`${rel.relateDatabaseName}:${rel.relateTableName}`]}</div>
                            )}
                       </div>
                   </div>
               </div>

              <div className="mt-5 pt-4 border-t border-slate-100 flex justify-between items-center relative z-10">
                <span className="text-[10px] text-slate-400 font-medium bg-slate-50 px-2 py-1 rounded">操作人: {rel.userName || '-'}</span>
                <button onClick={() => handleDelete(rel)} className="text-xs font-medium text-rose-500 hover:text-rose-600 bg-rose-50 hover:bg-rose-100 px-2.5 py-1 rounded transition flex items-center gap-1"><Trash2 size={12}/> 移除</button>
              </div>
            </article>
          ));
          })()}
        </div>
      </div>
      <ConfirmDialog {...dialogProps} />

      {/* Source DB Browser */}
      <DatabaseBrowser
        open={showSourceBrowser}
        onClose={() => setShowSourceBrowser(false)}
        environment={environment}
        environments={envOptions}
        onEnvironmentChange={() => undefined}
        projectId={activeProjectId || 0}
        focusedConnectionId={selectedConnection?.ID}
        onTableSelect={(value) => setSourceDbStr(value)}
      />

      {/* Target DB Browser */}
      <DatabaseBrowser
        open={showTargetBrowser}
        onClose={() => setShowTargetBrowser(false)}
        environment={environment}
        environments={envOptions}
        onEnvironmentChange={() => undefined}
        projectId={activeProjectId || 0}
        focusedConnectionId={selectedConnection?.ID}
        onTableSelect={(value) => setTargetDbStr(value)}
      />
    </div>
  );
}
