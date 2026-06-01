import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbTableColumn {
  ID: number;
  connectionId: number;
  tableId: string;
  columnName: string;
  columnType: string;
  columnSize: string;
  isEmpty: number;
  defaultValue: string;
  description: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbTableColumn = (data: Partial<TbTableColumn>) => {
  return service.post<any, ApiResponse>('/tableColumn/createTbTableColumn', data);
};

export const deleteTbTableColumn = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/tableColumn/deleteTbTableColumn', { data });
};

export const updateTbTableColumn = (data: Partial<TbTableColumn>) => {
  return service.put<any, ApiResponse>('/tableColumn/updateTbTableColumn', data);
};

export const getTbTableColumnList = (params: { page: number; pageSize: number; tableId?: string; columnName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbTableColumn>>>('/tableColumn/getTbTableColumnList', { params });
};
