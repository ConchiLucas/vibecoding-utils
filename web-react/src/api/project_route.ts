import service from '@/utils/request';

// 保存或更新项目路由配置
export const saveOrUpdateRoute = (data: any) => {
  return service({
    url: '/projectRoute/saveOrUpdate',
    method: 'post',
    data,
  });
};

// 删除路由配置
export const deleteRoute = (data: { ID: number }) => {
  return service({
    url: '/projectRoute/deleteRoute',
    method: 'delete',
    data,
  });
};
