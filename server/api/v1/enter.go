package v1

import (
	"github.com/flipped-aurora/easy-deploy/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	SystemApiGroup system.ApiGroup
}
