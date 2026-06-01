import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbTablePrefer {
  ID: number;
  databaseName: string;
  tableName: string;
  columnValue: string;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbTablePrefer = (data: Partial<TbTablePrefer>) => {
  return service.post<any, ApiResponse>('/tablePrefer/createTbTablePrefer', data);
};

export const deleteTbTablePrefer = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/tablePrefer/deleteTbTablePrefer', { data });
};

export const updateTbTablePrefer = (data: Partial<TbTablePrefer>) => {
  return service.put<any, ApiResponse>('/tablePrefer/updateTbTablePrefer', data);
};

export const getTbTablePreferList = (params: { page: number; pageSize: number; tableName?: string; userName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbTablePrefer>>>('/tablePrefer/getTbTablePreferList', { params });
};
