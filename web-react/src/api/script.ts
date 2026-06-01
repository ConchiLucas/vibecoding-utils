import service from '../utils/request';

// 分页获取脚本列表
export const getScriptPage = (data: any) => {
  return service({
    url: '/script/page',
    method: 'post',
    data: data
  });
};

// 根据ID获取脚本
export const getScriptById = (id: number | string) => {
  return service({
    url: '/script/getById/' + id,
    method: 'get'
  });
};

// 删除脚本
export const deleteScript = (ids: any) => {
  return service({
    url: '/script/delete/' + ids,
    method: 'delete'
  });
};

// 上传文件
export const uploadScriptFile = (data: FormData) => {
  return service({
    url: '/script/upload',
    method: 'post',
    data: data,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  });
};

// 下载文件
export const downloadScriptFile = (id: number | string) => {
  return service({
    url: '/script/download/' + id,
    method: 'get',
    responseType: 'blob'
  });
};

// 预览文件内容
export const previewScriptFile = (id: number | string) => {
  return service({
    url: '/script/preview/' + id,
    method: 'get'
  });
};

// 更新或新增文件
export const saveOrUpdateScript = (data: any) => {
  return service({
    url: '/script/saveOrUpdate',
    method: 'post',
    data: data
  });
};
