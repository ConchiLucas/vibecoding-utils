import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbTable {
  ID: number;
  databaseName: string;
  tableName: string;
  description: string;
  connectionId: number;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbTable = (data: Partial<TbTable>) => {
  return service.post<any, ApiResponse>('/table/createTbTable', data);
};

export const deleteTbTable = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/table/deleteTbTable', { data });
};

export const updateTbTable = (data: Partial<TbTable>) => {
  return service.put<any, ApiResponse>('/table/updateTbTable', data);
};

export const getTbTableList = (params: { page: number; pageSize: number; tableName?: string; databaseName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbTable>>>('/table/getTbTableList', { params });
};
