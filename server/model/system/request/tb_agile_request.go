package request

import "github.com/flipped-aurora/easy-deploy/server/model/common/request"

type AgileRequestSend struct {
	Method         string `json:"method"`
	URL            string `json:"url"`
	RequestHeaders string `json:"requestHeaders"`
	RequestBody    string `json:"requestBody"`
}

type AgileRequestSearch struct {
	request.PageInfo
	Method    string `json:"method" form:"method"`
	Keyword   string `json:"keyword" form:"keyword"`
	IsSuccess *int   `json:"isSuccess" form:"isSuccess"`
}
