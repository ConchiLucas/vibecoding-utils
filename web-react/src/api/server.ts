import service from '../utils/request';

// 获取服务器列表（不分页，用于下拉）
export const getServerList = (data?: any) => {
  return service({
    url: '/server/list',
    method: 'post',
    data: data || {}
  });
};

// 分页获取服务器列表
export const getServerPage = (data: any) => {
  return service({
    url: '/server/page',
    method: 'post',
    data: data
  });
};

// 根据ID获取服务器
export const getServerById = (id: number | string) => {
  return service({
    url: '/server/getById/' + id,
    method: 'get'
  });
};

// 新增或修改服务器
export const saveOrUpdateServer = (data: any) => {
  return service({
    url: '/server/saveOrUpdate',
    method: 'post',
    data: data
  });
};

// 删除服务器
export const deleteServer = (ids: any) => {
  return service({
    url: '/server/delete/' + ids,
    method: 'delete'
  });
};
