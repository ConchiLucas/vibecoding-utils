import service from '../utils/request';

// 基础响应类型定义
export interface ApiResponse<T = any> {
  code: number;
  data: T;
  msg: string;
}

export interface PageResult<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DictData {
  ID: number;
  dictType: string;
  dictLabel: string;
  dictValue: string;
  labelClass: string;
  userName?: string;
  extendParams?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createDictData = (data: Partial<DictData>) => {
  return service.post<any, ApiResponse>('/dict/createDictData', data);
};

export const deleteDictData = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/dict/deleteDictData', { data });
};

export const updateDictData = (data: Partial<DictData>) => {
  return service.put<any, ApiResponse>('/dict/updateDictData', data);
};

export const getDictDataList = (params: { page: number; pageSize: number; dictType?: string }) => {
  return service.get<any, ApiResponse<PageResult<DictData>>>('/dict/getDictDataList', { params });
};
