package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbDictDataService struct{}

// CreateDictData 创建字典数据
func (s *TbDictDataService) CreateDictData(dictData system.TbDictData) (err error) {
	err = global.GVA_DB.Create(&dictData).Error
	return err
}

// DeleteDictData 删除字典数据
func (s *TbDictDataService) DeleteDictData(dictData system.TbDictData) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&dictData).Error
	return err
}

// DeleteDictDataByIds 批量删除字典数据
func (s *TbDictDataService) DeleteDictDataByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbDictData{}, "id in ?", ids).Error
	return err
}

// UpdateDictData 更新字典数据
func (s *TbDictDataService) UpdateDictData(dictData *system.TbDictData) (err error) {
	err = global.GVA_DB.Updates(dictData).Error
	return err
}

// GetDictData 根据id获取字典数据
func (s *TbDictDataService) GetDictData(id uint) (dictData system.TbDictData, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&dictData).Error
	return
}

// GetDictDataInfoList 分页获取字典数据列表
func (s *TbDictDataService) GetDictDataInfoList(info systemReq.DictDataSearch) (list []system.TbDictData, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbDictData{})
	if info.DictType != "" {
		db = db.Where("dict_type = ?", info.DictType)
	}
	if info.UserName != "" {
		db = db.Where("user_name = ?", info.UserName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

// GetAllDictData 获取所有字典数据
func (s *TbDictDataService) GetAllDictData() (list []system.TbDictData, err error) {
	err = global.GVA_DB.Find(&list).Error
	return
}
