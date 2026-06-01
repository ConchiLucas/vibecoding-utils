package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbAgileRequestService struct{}

func (s *TbAgileRequestService) Send(req systemReq.AgileRequestSend, userName string) (system.TbAgileRequestLog, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	if !isAgileAllowedMethod(method) {
		return system.TbAgileRequestLog{}, fmt.Errorf("暂不支持的请求方法: %s", method)
	}

	requestURL := strings.TrimSpace(req.URL)
	if err := validateAgileURL(requestURL); err != nil {
		return system.TbAgileRequestLog{}, err
	}

	headerMap, normalizedHeaders, err := normalizeAgileHeaders(req.RequestHeaders)
	if err != nil {
		return system.TbAgileRequestLog{}, err
	}

	body := strings.TrimSpace(req.RequestBody)
	bodyBytes := []byte{}
	if body != "" {
		var jsonBody interface{}
		if err := json.Unmarshal([]byte(body), &jsonBody); err != nil {
			return system.TbAgileRequestLog{}, fmt.Errorf("JSON Body 格式不正确: %w", err)
		}
		bodyBytes, _ = json.Marshal(jsonBody)
		body = string(bodyBytes)
	}

	record := system.TbAgileRequestLog{
		UserName:       userName,
		Method:         method,
		URL:            requestURL,
		RequestHeaders: normalizedHeaders,
		RequestBody:    body,
	}

	start := time.Now()
	httpReq, err := http.NewRequest(method, requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return recordAgileFailure(record, start, fmt.Errorf("构建请求失败: %w", err))
	}

	if body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headerMap {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return recordAgileFailure(record, start, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respHeaders, _ := json.Marshal(resp.Header)
	record.DurationMs = time.Since(start).Milliseconds()
	record.ResponseStatus = resp.StatusCode
	record.ResponseHeaders = string(respHeaders)
	record.ResponseBody = formatAgileResponseBody(respBody)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		record.IsSuccess = 1
	}

	if err := createAgileRequestLog(&record); err != nil {
		return record, err
	}
	return record, nil
}

func (s *TbAgileRequestService) GetList(info systemReq.AgileRequestSearch, userName string) (list []system.TbAgileRequestLog, total int64, err error) {
	if err = ensureAgileRequestTable(); err != nil {
		return []system.TbAgileRequestLog{}, 0, nil
	}

	db := global.GVA_DB.Model(&system.TbAgileRequestLog{}).Where("user_name = ?", userName)
	if info.Method != "" {
		db = db.Where("method = ?", strings.ToUpper(info.Method))
	}
	if strings.TrimSpace(info.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(info.Keyword) + "%"
		db = db.Where("url LIKE ? OR request_body LIKE ?", keyword, keyword)
	}
	if info.IsSuccess != nil {
		db = db.Where("is_success = ?", *info.IsSuccess)
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Scopes(info.Paginate()).Order("id desc").Find(&list).Error
	return
}

func (s *TbAgileRequestService) GetByID(id uint, userName string) (log system.TbAgileRequestLog, err error) {
	if err = ensureAgileRequestTable(); err != nil {
		return
	}
	err = global.GVA_DB.Where("id = ? AND user_name = ?", id, userName).First(&log).Error
	return
}

func (s *TbAgileRequestService) DeleteByID(id uint, userName string) error {
	if err := ensureAgileRequestTable(); err != nil {
		return nil
	}
	return global.GVA_DB.Unscoped().Where("id = ? AND user_name = ?", id, userName).Delete(&system.TbAgileRequestLog{}).Error
}

func (s *TbAgileRequestService) Clear(userName string) error {
	if err := ensureAgileRequestTable(); err != nil {
		return nil
	}
	return global.GVA_DB.Unscoped().Where("user_name = ?", userName).Delete(&system.TbAgileRequestLog{}).Error
}

func ensureAgileRequestTable() error {
	if global.GVA_DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	if global.GVA_DB.Migrator().HasTable(&system.TbAgileRequestLog{}) {
		return nil
	}
	return global.GVA_DB.AutoMigrate(&system.TbAgileRequestLog{})
}

func createAgileRequestLog(record *system.TbAgileRequestLog) error {
	if err := ensureAgileRequestTable(); err != nil {
		return err
	}
	return global.GVA_DB.Create(record).Error
}

func isAgileAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func validateAgileURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("请输入请求 URL")
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("URL 格式不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https URL")
	}
	return nil
}

func normalizeAgileHeaders(raw string) (map[string]string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, "{}", nil
	}
	var generic map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return nil, "", fmt.Errorf("Headers 必须是 JSON 对象: %w", err)
	}
	headerMap := make(map[string]string, len(generic))
	for k, v := range generic {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		headerMap[key] = fmt.Sprint(v)
	}
	normalized, _ := json.Marshal(headerMap)
	return headerMap, string(normalized), nil
}

func recordAgileFailure(record system.TbAgileRequestLog, start time.Time, reqErr error) (system.TbAgileRequestLog, error) {
	record.DurationMs = time.Since(start).Milliseconds()
	record.IsSuccess = 0
	record.ErrorMessage = reqErr.Error()
	record.ResponseBody = fmt.Sprintf(`{"success":false,"message":%q}`, reqErr.Error())
	if err := createAgileRequestLog(&record); err != nil {
		return record, err
	}
	return record, nil
}

func formatAgileResponseBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		formatted, _ := json.MarshalIndent(parsed, "", "  ")
		return string(formatted)
	}
	return string(body)
}
