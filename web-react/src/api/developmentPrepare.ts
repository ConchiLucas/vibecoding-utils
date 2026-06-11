import service from '../utils/request';

export type DevelopmentPrepareType = 'script' | 'code' | 'checklist';

export interface DevelopmentPrepareItem {
  ID: number;
  CreatedAt?: string;
  UpdatedAt?: string;
  projectConfigId?: number;
  projectConfigName?: string;
  businessGroup?: string;
  title: string;
  itemType: DevelopmentPrepareType;
  language?: string;
  tags?: string;
  summary?: string;
  content?: string;
  isPinned?: boolean;
  sort?: number;
  userId?: number;
}

export interface DevelopmentPrepareSearch {
  page?: number;
  pageSize?: number;
  keyword?: string;
  projectConfigId?: number | null;
  projectConfigName?: string;
  businessGroup?: string;
  itemType?: DevelopmentPrepareType;
}

export const getDevelopmentPreparePage = (data: DevelopmentPrepareSearch) => {
  return service({
    url: '/developmentPrepare/page',
    method: 'post',
    data,
    donNotShowLoading: true,
  });
};

export const getDevelopmentPrepareDetail = (id: number | string) => {
  return service({
    url: `/developmentPrepare/detail/${id}`,
    method: 'get',
    donNotShowLoading: true,
  });
};

export const saveOrUpdateDevelopmentPrepare = (data: Partial<DevelopmentPrepareItem>) => {
  return service({
    url: '/developmentPrepare/saveOrUpdate',
    method: 'post',
    data,
  });
};

export const deleteDevelopmentPrepare = (ids: number | string) => {
  return service({
    url: `/developmentPrepare/delete/${ids}`,
    method: 'delete',
  });
};
