package response

import (
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbUserResponse struct {
	User system.TbUser `json:"user"`
}

type LoginResponse struct {
	User      system.TbUser `json:"user"`
	Token     string         `json:"token"`
	ExpiresAt int64          `json:"expiresAt"`
}
