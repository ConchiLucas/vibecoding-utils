import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbConnection {
  ID: number;
  connectionName: string;
  connectionType: string;
  connectionUrl: string;
  connectionGroup: string;
  databaseName: string;
  port: number;
  dbLoginName: string;
  dbLoginPassword?: string;
  userName: string;
  envName?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface RemoteDatabase {
  connectionId: number;
  connectionName: string;
  connectionType: string;
  databaseName: string;
  envName?: string;
}

export const createTbConnection = (data: Partial<TbConnection>) => {
  return service.post<any, ApiResponse>('/connection/createTbConnection', data);
};

export const deleteTbConnection = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/connection/deleteTbConnection', { data });
};

export const updateTbConnection = (data: Partial<TbConnection>) => {
  return service.put<any, ApiResponse>('/connection/updateTbConnection', data);
};

export const getTbConnectionList = (params: { page: number; pageSize: number; connectionName?: string; connectionType?: string; connectionGroup?: string; envName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbConnection>>>('/connection/getTbConnectionList', { params });
};

export const testConnection = (params: { ID: number }) => {
  return service.get<any, ApiResponse>('/connection/testConnection', { params });
};

export const testConnectionPayload = (data: Partial<TbConnection>) => {
  return service.post<any, ApiResponse>('/connection/testConnectionPayload', data);
};

export const initConnection = (params: { ID: number }) => {
  return service.get<any, ApiResponse>('/connection/initConnection', { params });
};

export const getRemoteDatabases = (params: { connectionGroup?: string; envName?: string; ID?: number }) => {
  return service.get<any, ApiResponse<RemoteDatabase[]>>('/connection/getRemoteDatabases', { params });
};

export const getRemoteTables = (params: { ID: number; databaseName?: string }) => {
  return service.get<any, ApiResponse<string[]>>('/connection/getRemoteTables', { params });
};

export const getRemoteTableComments = (params: { ID: number; databaseName?: string }) => {
  return service.get<any, ApiResponse<Record<string, string>>>('/connection/getRemoteTableComments', { params });
};

export interface RemoteTableDDL {
  databaseName: string;
  tableName: string;
  sql: string;
}

export const getRemoteTableDDL = (params: { ID: number; databaseName?: string; tableName: string }) => {
  return service.get<any, ApiResponse<RemoteTableDDL>>('/connection/getRemoteTableDDL', { params });
};

export interface ColumnPreview {
  name: string;
  value: string;
  description: string;
  isNull?: boolean;
  primaryKey?: boolean;
}

export interface TableRecordPreview {
  columns: ColumnPreview[];
  total: number;
  offset: number;
}

export const getRemoteTablePreview = (params: { ID: number; databaseName?: string; tableName: string; offset?: number; filterColumn?: string; filterValue?: string }) => {
  return service.get<any, ApiResponse<TableRecordPreview>>('/connection/getRemoteTablePreview', { params });
};

export interface TableDataColumn {
  name: string;
  description: string;
  primaryKey?: boolean;
}

export interface TableDataCell {
  value: string;
  isNull?: boolean;
}

export interface TableDataRow {
  offset: number;
  cells: TableDataCell[];
}

export interface TableDataPage {
  columns: TableDataColumn[];
  rows: TableDataRow[];
  total: number;
  page: number;
  pageSize: number;
}

export const getRemoteTablePage = (params: { ID: number; databaseName?: string; tableName: string; page?: number; pageSize?: number; filterColumn?: string; filterValue?: string }) => {
  return service.get<any, ApiResponse<TableDataPage>>('/connection/getRemoteTablePage', { params });
};

export interface RemoteTableGenerateResult {
  requested: number;
  inserted: number;
  provider: string;
  model: string;
}

export const generateRemoteTableData = (data: { ID: number; databaseName?: string; tableName: string; count: number }) => {
  return service.post<any, ApiResponse<RemoteTableGenerateResult>>('/connection/generateRemoteTableData', data);
};

export interface RemoteSQLQueryResult {
  columns: string[];
  rows: unknown[][];
  limit: number;
  returned: number;
  truncated: boolean;
  elapsedMs: number;
  databaseName?: string;
}

export const queryRemoteSQL = (data: { ID: number; databaseName?: string; sql: string; limit?: number }) => {
  return service.post<any, ApiResponse<RemoteSQLQueryResult>>('/connection/queryRemoteSQL', data);
};

export interface RemoteSQLHistoryRecord {
  id: number;
  projectConfigId: number;
  envName: string;
  connectionId: number;
  connectionName: string;
  connectionType: string;
  databaseName: string;
  sql: string;
  createdAt: string;
}

export interface RemoteSQLHistoryScope {
  projectConfigId: number;
  envName?: string;
  connectionId?: number;
  databaseName?: string;
}

export const getRemoteSQLHistory = (params: RemoteSQLHistoryScope & { limit?: number }) => {
  return service.get<any, ApiResponse<RemoteSQLHistoryRecord[]>>('/connection/getRemoteSQLHistory', { params });
};

export const saveRemoteSQLHistory = (data: RemoteSQLHistoryScope & { sql: string }) => {
  return service.post<any, ApiResponse<RemoteSQLHistoryRecord[]>>('/connection/saveRemoteSQLHistory', data);
};

export const deleteRemoteSQLHistory = (data: RemoteSQLHistoryScope & { id: number }) => {
  return service.delete<any, ApiResponse<RemoteSQLHistoryRecord[]>>('/connection/deleteRemoteSQLHistory', { data });
};

export const clearRemoteSQLHistory = (data: RemoteSQLHistoryScope) => {
  return service.delete<any, ApiResponse>('/connection/clearRemoteSQLHistory', { data });
};

export interface TableRecordUpdateChange {
  name: string;
  value: string;
}

export const updateRemoteTableRecord = (data: {
  ID: number;
  databaseName?: string;
  tableName: string;
  offset: number;
  filterColumn?: string;
  filterValue?: string;
  changes: TableRecordUpdateChange[];
}) => {
  return service.post<any, ApiResponse<TableRecordPreview>>('/connection/updateRemoteTableRecord', data);
};

export const deleteRemoteTableRecord = (data: {
  ID: number;
  databaseName?: string;
  tableName: string;
  offset: number;
  filterColumn?: string;
  filterValue?: string;
}) => {
  return service.delete<any, ApiResponse>('/connection/deleteRemoteTableRecord', { data });
};
