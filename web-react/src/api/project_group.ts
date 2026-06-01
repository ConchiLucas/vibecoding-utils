import service from '../utils/request';

// 获取项目组列表
export const getGroupList = (data?: any) => {
  return service({
    url: '/projectGroup/list',
    method: 'post',
    data: data || {},
  });
};

// 新增或修改项目组
export const saveOrUpdateGroup = (data: any) => {
  return service({
    url: '/projectGroup/saveOrUpdate',
    method: 'post',
    data: data,
  });
};

// 删除项目组
export const deleteGroup = (id: number) => {
  return service({
    url: '/projectGroup/delete/' + id,
    method: 'delete',
  });
};
