import type { TbTableRelate } from '../../api/sysRelate';

export type RelationDirection = 'outgoing' | 'incoming';

export interface TableRelationGroup {
  tableKey: string;
  databaseName: string;
  tableName: string;
  relations: TbTableRelate[];
}

export interface TableLineageNode {
  key: string;
  databaseName: string;
  tableName: string;
  outgoing: TableRelationGroup[];
  incoming: TableRelationGroup[];
  relatedTableCount: number;
  fieldRelationCount: number;
}

export interface TableLineageViewModel {
  tables: TableLineageNode[];
}

export function tableKey(databaseName?: string, tableName?: string) {
  const db = (databaseName || '').trim();
  const table = (tableName || '').trim();
  return db && table ? `${db}:${table}` : table || db || '-';
}

function createNode(databaseName: string, tableName: string): TableLineageNode {
  return {
    key: tableKey(databaseName, tableName),
    databaseName,
    tableName,
    outgoing: [],
    incoming: [],
    relatedTableCount: 0,
    fieldRelationCount: 0,
  };
}

function addRelationToGroup(groups: TableRelationGroup[], databaseName: string, tableName: string, relation: TbTableRelate) {
  const key = tableKey(databaseName, tableName);
  let group = groups.find(item => item.tableKey === key);
  if (!group) {
    group = {
      tableKey: key,
      databaseName,
      tableName,
      relations: [],
    };
    groups.push(group);
  }
  group.relations.push(relation);
}

function sortGroups(groups: TableRelationGroup[]) {
  groups.sort((a, b) => a.tableKey.localeCompare(b.tableKey));
  for (const group of groups) {
    group.relations.sort((a, b) =>
      `${a.columnName}:${a.relateColumnName}:${a.ID}`.localeCompare(`${b.columnName}:${b.relateColumnName}:${b.ID}`)
    );
  }
}

export function buildTableLineageViewModel(relations: TbTableRelate[]): TableLineageViewModel {
  const nodeMap = new Map<string, TableLineageNode>();

  const ensureNode = (databaseName: string, tableName: string) => {
    const key = tableKey(databaseName, tableName);
    let node = nodeMap.get(key);
    if (!node) {
      node = createNode(databaseName, tableName);
      nodeMap.set(key, node);
    }
    return node;
  };

  for (const relation of relations) {
    const sourceDb = relation.databaseName || '';
    const sourceTable = relation.tableName || '';
    const targetDb = relation.relateDatabaseName || '';
    const targetTable = relation.relateTableName || '';
    if (!sourceTable && !sourceDb) continue;
    if (!targetTable && !targetDb) continue;

    const sourceNode = ensureNode(sourceDb, sourceTable);
    const targetNode = ensureNode(targetDb, targetTable);

    addRelationToGroup(sourceNode.outgoing, targetDb, targetTable, relation);
    addRelationToGroup(targetNode.incoming, sourceDb, sourceTable, relation);
  }

  const tables = Array.from(nodeMap.values());
  for (const node of tables) {
    sortGroups(node.outgoing);
    sortGroups(node.incoming);
    const relatedKeys = new Set<string>();
    for (const group of node.outgoing) relatedKeys.add(group.tableKey);
    for (const group of node.incoming) relatedKeys.add(group.tableKey);
    node.relatedTableCount = relatedKeys.size;
    node.fieldRelationCount =
      node.outgoing.reduce((total, group) => total + group.relations.length, 0) +
      node.incoming.reduce((total, group) => total + group.relations.length, 0);
  }

  tables.sort((a, b) => {
    const countDiff = b.fieldRelationCount - a.fieldRelationCount;
    if (countDiff !== 0) return countDiff;
    return a.key.localeCompare(b.key);
  });

  return { tables };
}
