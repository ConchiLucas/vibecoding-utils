import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbEntity {
  ID: number;
  entityName: string;
  requiredColumn: string;
  columnCount: number;
  containEntity: number;
  userName: string;
  serverName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbEntity = (data: Partial<TbEntity>) => {
  return service.post<any, ApiResponse>('/entity/createTbEntity', data);
};

export const deleteTbEntity = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/entity/deleteTbEntity', { data });
};

export const updateTbEntity = (data: Partial<TbEntity>) => {
  return service.put<any, ApiResponse>('/entity/updateTbEntity', data);
};

export const getTbEntityList = (params: { page: number; pageSize: number; entityName?: string; serverName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbEntity>>>('/entity/getTbEntityList', { params });
};
