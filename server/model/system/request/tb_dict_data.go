package request

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type DictDataSearch struct {
	system.TbDictData
	request.PageInfo
}
