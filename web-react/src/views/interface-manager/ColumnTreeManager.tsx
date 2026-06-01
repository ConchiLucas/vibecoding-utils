import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { ChevronDown, ChevronRight, ListTree } from 'lucide-react';
import { getColumnTree, ColumnTreeVO } from '../../api/sysColumn';

// Internal flattening helper to convert tree into renderable list of rows with depth
interface FlattenedNode {
  node: ColumnTreeVO;
  depth: number;
  isExpanded: boolean;
  hasChildren: boolean;
}

export default function ColumnTreeManager({ interfaceId, type }: { interfaceId: number; type: 1 | 2 }) {
  const [treeData, setTreeData] = useState<ColumnTreeVO[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set());
  const [isExpandAll, setIsExpandAll] = useState(false);

  useEffect(() => {
    async function loadData() {
      setLoading(true);
      try {
        const res = await getColumnTree({ id: interfaceId, type });
        setTreeData(res.data || []);
      } catch (err) {
        console.error(err);
        toast.error(`加载${type === 1 ? '入参' : '出参'}树失败`);
      } finally {
        setLoading(false);
      }
    }
    if (interfaceId) {
      loadData();
    }
  }, [interfaceId, type]);

  // Compute flattened list based on expanded state
  const flattenTree = (nodes: ColumnTreeVO[], depth = 0, flattened: FlattenedNode[] = []): FlattenedNode[] => {
    for (const node of nodes) {
      const hasChildren = !!node.children && node.children.length > 0;
      const isExpanded = expandedIds.has(node.id) || isExpandAll;
      
      flattened.push({ node, depth, isExpanded, hasChildren });
      
      if (hasChildren && isExpanded) {
        flattenTree(node.children!, depth + 1, flattened);
      }
    }
    return flattened;
  };

  const rows = flattenTree(treeData);

  const toggleExpand = (id: number) => {
    const newSet = new Set(expandedIds);
    if (newSet.has(id)) {
      newSet.delete(id);
      setIsExpandAll(false); // Can't be 'expand all' if we manually collapse
    } else {
      newSet.add(id);
    }
    setExpandedIds(newSet);
  };

  const handleToggleExpandAll = () => {
    if (isExpandAll) {
      setIsExpandAll(false);
      setExpandedIds(new Set());
    } else {
      setIsExpandAll(true);
      // We don't need to put everything in expandedIds, flattenTree checks isExpandAll
    }
  };

  return (
    <div className="flex flex-col h-full space-y-4">
      <div className="flex justify-between items-end border-b border-slate-200 pb-4">
        <div>
           <h2 className="text-xl font-semibold text-slate-800 flex items-center gap-2">
              <ListTree className="text-indigo-600" size={24} />
              {type === 1 ? '入参对象结构树' : '出参对象结构树'}
           </h2>
           <p className="text-sm text-slate-500 mt-1">
              展示当前接口的{type === 1 ? '请求参数' : '返回参数'}嵌套结构定义。
           </p>
        </div>
        <button 
          onClick={handleToggleExpandAll}
          className="px-4 py-2 border border-slate-200 bg-white hover:bg-slate-50 text-slate-700 text-sm font-medium rounded-lg transition"
        >
          {isExpandAll ? "全部收起" : "全部展开"}
        </button>
      </div>

      <div className="flex-1 bg-white border border-slate-200 rounded-2xl overflow-hidden flex flex-col">
          {loading ? (
             <div className="flex-1 flex justify-center items-center text-slate-500 text-sm py-12">
               正在加载参数结构...
             </div>
          ) : rows.length === 0 ? (
             <div className="flex-1 flex justify-center items-center text-slate-400 text-sm py-12">
               暂无参配置数据
             </div>
          ) : (
            <div className="overflow-x-auto flex-1">
              <table className="w-full text-left text-sm text-slate-600 font-mono">
                <thead className="bg-slate-50 text-slate-500 border-b border-slate-200 text-xs uppercase tracking-wider sticky top-0 z-10 font-sans">
                  <tr>
                    <th className="px-6 py-4 font-bold min-w-[200px]">字段名称</th>
                    <th className="px-6 py-4 font-bold min-w-[150px]">描述</th>
                    <th className="px-6 py-4 font-bold w-32">字段类型</th>
                    <th className="px-6 py-4 font-bold w-32">默认值</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                   {rows.map((row) => (
                      <tr key={row.node.id} className="hover:bg-slate-50/50 transition whitespace-nowrap">
                         <td className="px-6 py-3 flex items-center gap-2" style={{ paddingLeft: `${1.5 + row.depth * 1.5}rem` }}>
                            {row.hasChildren ? (
                               <button onClick={() => toggleExpand(row.node.id)} className="p-0.5 rounded hover:bg-slate-200 text-slate-400 transition cursor-pointer">
                                  {row.isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                               </button>
                            ) : (
                               <span className="w-[18px]"></span> /* spacer */
                            )}
                            <span className="font-semibold text-indigo-700">{row.node.columnName || '-'}</span>
                         </td>
                         <td className="px-6 py-3 font-sans text-slate-600 truncate max-w-sm" title={row.node.description || ''}>
                           {row.node.description || '-'}
                         </td>
                         <td className="px-6 py-3">
                           <span className="px-2 py-0.5 bg-slate-100/50 text-slate-500 rounded text-xs border border-slate-200 shadow-sm">{row.node.columnType || '-'}</span>
                         </td>
                         <td className="px-6 py-3 text-emerald-600">
                           {row.node.defaultValue || '-'}
                         </td>
                      </tr>
                   ))}
                </tbody>
              </table>
            </div>
          )}
      </div>
    </div>
  );
}
