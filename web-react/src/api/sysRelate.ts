import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbTableRelate {
  ID: number;
  projectConfigId?: number;
  databaseName: string;
  tableName: string;
  columnName: string;
  relateDatabaseName: string;
  relateTableName: string;
  relateColumnName: string;
  relateColumnType: string;
  columnType: string;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbTableRelate = (data: Partial<TbTableRelate>) => {
  return service.post<any, ApiResponse>('/tableRelate/createTbTableRelate', data);
};

export const deleteTbTableRelate = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/tableRelate/deleteTbTableRelate', { data });
};

export const updateTbTableRelate = (data: Partial<TbTableRelate>) => {
  return service.put<any, ApiResponse>('/tableRelate/updateTbTableRelate', data);
};

export const getTbTableRelateList = (params: { page: number; pageSize: number; projectConfigId?: number; tableName?: string; relateTableName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbTableRelate>>>('/tableRelate/getTbTableRelateList', { params });
};

export interface ClientColumnVO {
  name: string;
  description: string;
  value: string;
  columnType: string;
  length: string;
}

export const getRemoteColumns = (data: { environment: string; databaseStr: string; projectConfigId?: number; connectionId?: number }) => {
  return service.post<any, ApiResponse<ClientColumnVO[]>>('/tableRelate/getRemoteColumns', data);
};

export const getTableComments = (data: { projectConfigId: number; environment: string; connectionId?: number; tables: string[] }) => {
  return service.post<any, ApiResponse<Record<string, string>>>('/tableRelate/getTableComments', data);
};
