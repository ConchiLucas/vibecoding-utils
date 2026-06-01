package system

import "github.com/flipped-aurora/easy-deploy/server/global"

type TbAgileRequestLog struct {
	global.GVA_MODEL_NO_SOFT_DELETE
	UserName        string `json:"userName" form:"userName" gorm:"column:user_name;type:varchar(255);comment:用户名称"`
	Method          string `json:"method" form:"method" gorm:"column:method;type:varchar(16);comment:请求方法"`
	URL             string `json:"url" form:"url" gorm:"column:url;type:text;comment:请求地址"`
	RequestHeaders  string `json:"requestHeaders" form:"requestHeaders" gorm:"column:request_headers;type:text;comment:请求头JSON"`
	RequestBody     string `json:"requestBody" form:"requestBody" gorm:"column:request_body;type:text;comment:请求体JSON"`
	ResponseStatus  int    `json:"responseStatus" form:"responseStatus" gorm:"column:response_status;type:int;comment:响应状态码"`
	ResponseHeaders string `json:"responseHeaders" form:"responseHeaders" gorm:"column:response_headers;type:text;comment:响应头JSON"`
	ResponseBody    string `json:"responseBody" form:"responseBody" gorm:"column:response_body;type:text;comment:响应体"`
	DurationMs      int64  `json:"durationMs" form:"durationMs" gorm:"column:duration_ms;type:bigint;comment:请求耗时毫秒"`
	IsSuccess       int    `json:"isSuccess" form:"isSuccess" gorm:"column:is_success;type:int;comment:是否成功(1成功 0失败)"`
	ErrorMessage    string `json:"errorMessage" form:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (TbAgileRequestLog) TableName() string {
	return "tb_agile_request_log"
}
