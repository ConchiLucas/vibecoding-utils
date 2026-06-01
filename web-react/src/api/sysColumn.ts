import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbColumn {
  ID: number;
  entityName: string;
  columnName: string;
  columnType: string;
  description: string;
  defaultValue: string;
  formatValue: string;
  maxLength: number;
  minLength: number;
  required: number;
  enumValue: string;
  columnRef: string;
  userName: string;
  serverName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbColumn = (data: Partial<TbColumn>) => {
  return service.post<any, ApiResponse>('/column/createTbColumn', data);
};

export const deleteTbColumn = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/column/deleteTbColumn', { data });
};

export const updateTbColumn = (data: Partial<TbColumn>) => {
  return service.put<any, ApiResponse>('/column/updateTbColumn', data);
};

export const getTbColumnList = (params: { page: number; pageSize: number; entityName?: string; columnName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbColumn>>>('/column/getTbColumnList', { params });
};

export interface ColumnTreeVO {
  id: number;
  pid: number;
  entityName: string;
  columnName: string;
  columnType: string;
  description: string;
  defaultValue: string;
  columnRef: string;
  userName: string;
  serverName: string;
  children?: ColumnTreeVO[];
}

export const getColumnTree = (data: { id: number; type: number; userName?: string }) => {
  return service.post<any, ApiResponse<ColumnTreeVO[]>>('/column/getColumnTree', data);
};
