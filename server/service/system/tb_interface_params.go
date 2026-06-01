package system

import (
	"encoding/json"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbInterfaceParamsService struct{}

func (s *TbInterfaceParamsService) CreateTbInterfaceParams(params system.TbInterfaceParams) (err error) {
	err = global.GVA_DB.Create(&params).Error
	return err
}

func (s *TbInterfaceParamsService) DeleteTbInterfaceParams(params system.TbInterfaceParams) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&params).Error
	return err
}

func (s *TbInterfaceParamsService) DeleteTbInterfaceParamsByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbInterfaceParams{}, "id in ?", ids).Error
	return err
}

func (s *TbInterfaceParamsService) UpdateTbInterfaceParams(params *system.TbInterfaceParams) (err error) {
	err = global.GVA_DB.Updates(params).Error
	return err
}

func (s *TbInterfaceParamsService) GetTbInterfaceParams(id uint) (params system.TbInterfaceParams, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&params).Error
	return
}

func (s *TbInterfaceParamsService) GetTbInterfaceParamsInfoList(info systemReq.InterfaceParamsSearch) (list []system.TbInterfaceParams, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterfaceParams{})
	if info.InterfacePaths != "" {
		db = db.Where("interface_paths LIKE ?", "%"+info.InterfacePaths+"%")
	}
	if info.Environment != "" {
		db = db.Where("environment = ?", info.Environment)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

// ParamsEntityResult is returned by GetParamsEntity to the frontend
type ParamsEntityResult struct {
	ID              uint   `json:"id"`
	ParamsID        uint   `json:"paramsId"`
	Environment     string `json:"environment"`
	Identity        string `json:"identity"`
	InterfaceParams string `json:"interfaceParams"`
	ResponseParams  string `json:"responseParams"`
}

// GetParamsEntity mirrors Python params/entity:
//  1. Find the latest tb_interface_params record for this interface path & user.
//  2. If found → return its saved request + response params (with env/user).
//  3. If not found → build an empty JSON skeleton from tb_column for the interface's
//     requestParam entity, then carry the last-known env/identity from any record for this user.
//     Do not carry the fallback params ID: it belongs to a different interface path.
func (s *TbInterfaceParamsService) GetParamsEntity(interfaceID uint, interfacePaths, userName string) (ParamsEntityResult, error) {
	var result ParamsEntityResult

	// Step 1: Find the latest params record for this specific interface path
	var latest system.TbInterfaceParams
	err := global.GVA_DB.
		Where("interface_paths = ? AND user_name = ?", interfacePaths, userName).
		Order("id DESC").
		First(&latest).Error

	if err == nil {
		// Found — return it directly with request + response
		result = ParamsEntityResult{
			ID:              interfaceID,
			ParamsID:        latest.ID,
			Environment:     latest.Environment,
			Identity:        latest.Identity,
			InterfaceParams: latest.InterfaceParams,
			ResponseParams:  latest.ResponseParams,
		}
		return result, nil
	}

	// Step 2: Not found for this path — build skeleton from tb_column
	var iface system.TbInterface
	global.GVA_DB.Where("id = ?", interfaceID).First(&iface)

	jsonStr := "{}"
	if iface.RequestParam != "" {
		var columns []system.TbColumn
		global.GVA_DB.
			Where("entity_name = ? AND server_name = ?", iface.RequestParam, iface.ServerName).
			Find(&columns)

		if len(columns) > 0 {
			skeleton := make(map[string]interface{})
			for _, col := range columns {
				skeleton[col.ColumnName] = ""
			}
			b, _ := json.MarshalIndent(skeleton, "", "  ")
			jsonStr = string(b)
		}
	}

	// Step 3: Carry forward last-known env/identity from any record for this user
	var fallback system.TbInterfaceParams
	global.GVA_DB.
		Where("user_name = ?", userName).
		Order("id DESC").
		First(&fallback)

	result = ParamsEntityResult{
		ID:              interfaceID,
		ParamsID:        0,
		Environment:     fallback.Environment,
		Identity:        fallback.Identity,
		InterfaceParams: jsonStr,
		ResponseParams:  "",
	}
	return result, nil
}
