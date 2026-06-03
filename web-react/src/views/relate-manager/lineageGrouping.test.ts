import assert from 'node:assert/strict';
import { test } from 'node:test';
import { buildTableLineageViewModel } from './lineageGrouping.ts';

const relations = [
  {
    ID: 1,
    projectConfigId: 1,
    databaseName: 'sales',
    tableName: 'orders',
    columnName: 'user_id',
    columnType: 'bigint',
    relateDatabaseName: 'core',
    relateTableName: 'users',
    relateColumnName: 'id',
    relateColumnType: 'bigint',
    userName: 'ai',
  },
  {
    ID: 2,
    projectConfigId: 1,
    databaseName: 'sales',
    tableName: 'orders',
    columnName: 'address_id',
    columnType: 'bigint',
    relateDatabaseName: 'core',
    relateTableName: 'addresses',
    relateColumnName: 'id',
    relateColumnType: 'bigint',
    userName: 'ai',
  },
  {
    ID: 3,
    projectConfigId: 1,
    databaseName: 'billing',
    tableName: 'invoices',
    columnName: 'order_id',
    columnType: 'bigint',
    relateDatabaseName: 'sales',
    relateTableName: 'orders',
    relateColumnName: 'id',
    relateColumnType: 'bigint',
    userName: 'ai',
  },
];

test('groups table lineage by table with outgoing and incoming relations', () => {
  const viewModel = buildTableLineageViewModel(relations);
  const orders = viewModel.tables.find(table => table.key === 'sales:orders');

  assert.ok(orders);
  assert.equal(orders.databaseName, 'sales');
  assert.equal(orders.tableName, 'orders');
  assert.equal(orders.outgoing.length, 2);
  assert.equal(orders.incoming.length, 1);
  assert.equal(orders.relatedTableCount, 3);
  assert.equal(orders.fieldRelationCount, 3);
  assert.deepEqual(
    orders.outgoing.map(group => group.tableKey),
    ['core:addresses', 'core:users']
  );
  assert.deepEqual(
    orders.incoming.map(group => group.tableKey),
    ['billing:invoices']
  );
});
