# AI Table Relations Import

Use this endpoint when an AI agent has already analyzed another project and needs to import table-field relations into `tb_table_relate`.

## Endpoint

```http
POST /ai-tools/table-relations/import
```

The endpoint uses the same JWT authentication as the rest of the private API. Send the login token in `x-token`.

## Request

```json
{
  "projectConfigId": 1,
  "userName": "ai",
  "relations": [
    {
      "source": {
        "databaseName": "order_db",
        "tableName": "orders",
        "columnName": "user_id",
        "columnType": "bigint"
      },
      "target": {
        "databaseName": "user_db",
        "tableName": "users",
        "columnName": "id",
        "columnType": "bigint"
      }
    }
  ]
}
```

## Fields

| Field | Required | Meaning |
| --- | --- | --- |
| `projectConfigId` | Yes | Project config ID that owns these relations. |
| `userName` | No | Import actor. Defaults to the current user or `ai`. |
| `relations` | Yes | Batch of table-field relations to import. |
| `relations[].source.databaseName` | Yes | Source table database name. |
| `relations[].source.tableName` | Yes | Source table name. |
| `relations[].source.columnName` | Yes | Source table column name. |
| `relations[].source.columnType` | No | Source column type, such as `bigint` or `varchar`. |
| `relations[].target.databaseName` | Yes | Target table database name. |
| `relations[].target.tableName` | Yes | Target table name. |
| `relations[].target.columnName` | Yes | Target table column name. |
| `relations[].target.columnType` | No | Target column type, such as `bigint` or `varchar`. |

`source` is written to `database_name`, `table_name`, `column_name`, and `column_type`.
`target` is written to `relate_database_name`, `relate_table_name`, `relate_column_name`, and `relate_column_type`.

## Response

```json
{
  "code": 0,
  "data": {
    "projectConfigId": 1,
    "created": 1,
    "skipped": 0,
    "failed": [],
    "items": []
  },
  "msg": "导入成功"
}
```

Existing identical relations are skipped instead of duplicated.
