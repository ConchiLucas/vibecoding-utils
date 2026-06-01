package router

import (
	"github.com/flipped-aurora/easy-deploy/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System system.RouterGroup
}
