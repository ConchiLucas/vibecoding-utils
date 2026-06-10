package system

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

type TbGenerateProjectPathService struct{}

var (
	generatePathServiceSegmentPattern = regexp.MustCompile(`(?i)(^|[-_])(api|biz|service|server|ui|web|admin|gateway|job|task|client|frontend|backend)$`)
	generatePathServiceRootPattern    = regexp.MustCompile(`(?i)(^|[-_])service$`)
)

type GenerateProjectPromptSummaryResult struct {
	ProjectInstanceId  int                       `json:"projectInstanceId"`
	ProjectName        string                    `json:"projectName"`
	DiskPath           string                    `json:"diskPath"`
	PathSet            int                       `json:"pathSet"`
	Module             string                    `json:"module"`
	TableName          string                    `json:"tableName"`
	Prompt             string                    `json:"prompt"`
	ModifyInstructions string                    `json:"modifyInstructions"`
	TargetPaths        []string                  `json:"targetPaths"`
	Files              []GenerateProjectCodeFile `json:"files"`
}

func normalizeStoredRelativeDir(value string) string {
	next := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	for strings.Contains(next, "//") {
		next = strings.ReplaceAll(next, "//", "/")
	}
	next = strings.TrimLeft(next, "/")
	next = strings.TrimRight(next, "/")
	return strings.TrimSpace(next)
}

func normalizeStoredFileName(value string) string {
	next := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	for strings.Contains(next, "//") {
		next = strings.ReplaceAll(next, "//", "/")
	}
	next = strings.TrimLeft(next, "/")
	return strings.TrimSpace(next)
}

func replaceGeneratePathPrefix(value string, oldPrefix string, nextPrefix string) string {
	currentValue := normalizeStoredRelativeDir(value)
	currentPrefix := normalizeStoredRelativeDir(oldPrefix)
	targetPrefix := normalizeStoredRelativeDir(nextPrefix)
	if currentValue == "" || currentPrefix == "" || currentPrefix == "." {
		return currentValue
	}
	if currentValue == currentPrefix {
		return targetPrefix
	}
	if strings.HasPrefix(currentValue, currentPrefix+"/") {
		suffix := strings.TrimPrefix(currentValue, currentPrefix+"/")
		if targetPrefix == "" {
			return suffix
		}
		return normalizeStoredRelativeDir(targetPrefix + "/" + suffix)
	}
	return currentValue
}

func inferGenerateProjectServiceBasePath(pathObj system.TbGenerateProjectPath) string {
	dir := normalizeStoredRelativeDir(pathObj.FileUrl)
	fileName := normalizeStoredFileName(pathObj.FileName)
	combined := strings.TrimRight(normalizeStoredRelativeDir(dir+"/"+fileName), "/")
	rawParts := strings.Split(combined, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return "."
	}
	if parts[0] == ".." || parts[0] == "." {
		return parts[0]
	}
	if len(parts) >= 3 && generatePathServiceRootPattern.MatchString(parts[0]) && generatePathServiceSegmentPattern.MatchString(parts[1]) && parts[2] == "src" {
		return strings.Join(parts[:3], "/")
	}
	if len(parts) >= 2 && generatePathServiceRootPattern.MatchString(parts[0]) && parts[1] == "src" {
		for index := 2; index+1 < len(parts); index++ {
			if parts[index] == "module" {
				return strings.Join(parts[:index+2], "/")
			}
		}
		for index := 2; index+1 < len(parts); index++ {
			if parts[index] == "sql-ext" {
				return strings.Join(parts[:index+2], "/")
			}
		}
		return strings.Join(parts[:2], "/")
	}
	if len(parts) >= 2 && generatePathServiceRootPattern.MatchString(parts[0]) && generatePathServiceSegmentPattern.MatchString(parts[1]) {
		return strings.Join(parts[:2], "/")
	}
	for index := 1; index < len(parts) && index <= 3; index++ {
		if generatePathServiceSegmentPattern.MatchString(parts[index]) {
			return strings.Join(parts[:index+1], "/")
		}
	}
	if parts[0] == "src" && len(parts) >= 3 {
		switch parts[1] {
		case "api", "views", "pages", "components":
			return strings.Join(parts[:3], "/")
		}
	}
	if parts[0] == "src" && len(parts) >= 2 {
		return strings.Join(parts[:2], "/")
	}
	return parts[0]
}

func getGenerateProjectCommonPathPrefix(paths []system.TbGenerateProjectPath) string {
	dirs := make([]string, 0, len(paths))
	for _, pathObj := range paths {
		if dir := normalizeStoredRelativeDir(pathObj.FileUrl); dir != "" {
			dirs = append(dirs, dir)
		}
	}
	if len(dirs) == 0 {
		return "."
	}
	if len(dirs) == 1 {
		return dirs[0]
	}

	commonParts := strings.Split(dirs[0], "/")
	for _, dir := range dirs[1:] {
		parts := strings.Split(dir, "/")
		index := 0
		for index < len(commonParts) && index < len(parts) && commonParts[index] == parts[index] {
			index++
		}
		commonParts = commonParts[:index]
	}
	if len(commonParts) == 0 {
		return "."
	}
	return strings.Join(commonParts, "/")
}

func chooseGenerateProjectPathGroupBase(inferredKey string, paths []system.TbGenerateProjectPath) string {
	commonPrefix := getGenerateProjectCommonPathPrefix(paths)
	normalizedKey := normalizeStoredRelativeDir(inferredKey)
	if normalizedKey != "" && strings.Contains(normalizedKey, "/src") {
		if commonPrefix == normalizedKey || strings.HasPrefix(commonPrefix, normalizedKey+"/") {
			return normalizedKey
		}
	}
	return commonPrefix
}

func getGenerateProjectStoredPathSetName(paths []system.TbGenerateProjectPath) string {
	for _, pathObj := range paths {
		if name := strings.TrimSpace(pathObj.PathSetName); name != "" {
			return name
		}
	}
	return ""
}

func applyProjectPathScope(db *gorm.DB, projectId int, projectInstanceId int) *gorm.DB {
	if projectInstanceId > 0 {
		return db.Where("project_instance_id = ?", projectInstanceId)
	}
	if projectId > 0 {
		return db.Where("project_id = ? AND (project_instance_id = 0 OR project_instance_id IS NULL)", projectId)
	}
	return db
}

func (s *TbGenerateProjectPathService) CreateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	req.FileUrl = normalizeStoredRelativeDir(req.FileUrl)
	req.FileName = normalizeStoredFileName(req.FileName)
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPathService) CreatePathGroup(req *system.TbGenerateProjectPathGroup) error {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return errors.New("projectInstanceId is required")
	}
	req.BasePath = normalizeStoredRelativeDir(req.BasePath)
	req.PathSetName = strings.TrimSpace(req.PathSetName)
	if req.BasePath == "" {
		return errors.New("basePath is required")
	}

	var duplicated int64
	if err := global.GVA_DB.Model(&system.TbGenerateProjectPathGroup{}).
		Where("project_instance_id = ? AND path_set = ? AND base_path = ?", req.ProjectInstanceId, req.PathSet, req.BasePath).
		Where("id <> ?", req.ID).
		Count(&duplicated).Error; err != nil {
		return err
	}
	if duplicated > 0 {
		return errors.New("相对路径已存在")
	}

	if req.Sort == 0 {
		var maxSort int
		if err := global.GVA_DB.Model(&system.TbGenerateProjectPathGroup{}).
			Where("project_instance_id = ? AND path_set = ?", req.ProjectInstanceId, req.PathSet).
			Select("COALESCE(MAX(sort), 0)").
			Scan(&maxSort).Error; err != nil {
			return err
		}
		req.Sort = maxSort + 1
	}
	if req.ProjectId == 0 {
		req.ProjectId = req.ProjectInstanceId
	}
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPathService) UpdatePathGroup(req *system.TbGenerateProjectPathGroup) error {
	if req.ID == 0 {
		return errors.New("id is required")
	}
	nextBasePath := normalizeStoredRelativeDir(req.BasePath)
	if nextBasePath == "" {
		return errors.New("basePath is required")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	var current system.TbGenerateProjectPathGroup
	if err := tx.Where("id = ?", req.ID).First(&current).Error; err != nil {
		tx.Rollback()
		return err
	}

	var duplicated int64
	if err := tx.Model(&system.TbGenerateProjectPathGroup{}).
		Where("project_instance_id = ? AND path_set = ? AND base_path = ?", current.ProjectInstanceId, current.PathSet, nextBasePath).
		Where("id <> ?", current.ID).
		Count(&duplicated).Error; err != nil {
		tx.Rollback()
		return err
	}
	if duplicated > 0 {
		tx.Rollback()
		return errors.New("相对路径已存在")
	}

	oldBasePath := current.BasePath
	current.BasePath = nextBasePath
	if strings.TrimSpace(req.PathSetName) != "" {
		current.PathSetName = strings.TrimSpace(req.PathSetName)
	}
	if req.Sort > 0 {
		current.Sort = req.Sort
	}
	if err := tx.Save(&current).Error; err != nil {
		tx.Rollback()
		return err
	}

	var paths []system.TbGenerateProjectPath
	if err := tx.Where("path_group_id = ?", current.ID).Find(&paths).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, pathObj := range paths {
		pathObj.FileUrl = replaceGeneratePathPrefix(pathObj.FileUrl, oldBasePath, nextBasePath)
		pathObj.FileName = replaceGeneratePathPrefix(pathObj.FileName, oldBasePath, nextBasePath)
		if err := tx.Save(&pathObj).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *TbGenerateProjectPathService) DeletePathGroup(req system.TbGenerateProjectPathGroup) error {
	if req.ID == 0 {
		return errors.New("id is required")
	}

	var pathCount int64
	if err := global.GVA_DB.Model(&system.TbGenerateProjectPath{}).
		Where("path_group_id = ?", req.ID).
		Count(&pathCount).Error; err != nil {
		return err
	}
	if pathCount > 0 {
		return errors.New("该子目录下还有路径数据，不能删除")
	}
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateProjectPathService) GetPathGroupList(projectId int, projectInstanceId int) ([]system.TbGenerateProjectPathGroup, error) {
	if err := s.ensurePathGroupsForLegacyPaths(projectId, projectInstanceId); err != nil {
		return nil, err
	}

	var groups []system.TbGenerateProjectPathGroup
	db := applyProjectPathScope(global.GVA_DB.Model(&system.TbGenerateProjectPathGroup{}), projectId, projectInstanceId)
	if err := db.Order("path_set ASC, sort ASC, id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *TbGenerateProjectPathService) ensurePathGroupsForLegacyPaths(projectId int, projectInstanceId int) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	if err := s.ensurePathGroupsForLegacyPathsTx(tx, projectId, projectInstanceId); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateProjectPathService) ensurePathGroupsForLegacyPathsTx(tx *gorm.DB, projectId int, projectInstanceId int) error {
	var legacyPaths []system.TbGenerateProjectPath
	pathQuery := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPath{}), projectId, projectInstanceId).
		Where("(path_group_id = 0 OR path_group_id IS NULL)")
	if err := pathQuery.Order("path_set ASC, id ASC").Find(&legacyPaths).Error; err != nil {
		return err
	}
	if len(legacyPaths) == 0 {
		return s.repairStoredPathGroupsFromPathsTx(tx, projectId, projectInstanceId)
	}

	type legacyGroupBucket struct {
		pathSet int
		key     string
		paths   []system.TbGenerateProjectPath
	}
	buckets := make(map[string]*legacyGroupBucket)
	for _, pathObj := range legacyPaths {
		inferredKey := inferGenerateProjectServiceBasePath(pathObj)
		bucketKey := fmt.Sprintf("%d\x00%s", pathObj.PathSet, inferredKey)
		if _, ok := buckets[bucketKey]; !ok {
			buckets[bucketKey] = &legacyGroupBucket{pathSet: pathObj.PathSet, key: inferredKey}
		}
		buckets[bucketKey].paths = append(buckets[bucketKey].paths, pathObj)
	}

	var maxSortByPathSet = map[int]int{}
	var existingGroups []system.TbGenerateProjectPathGroup
	if err := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPathGroup{}), projectId, projectInstanceId).
		Find(&existingGroups).Error; err != nil {
		return err
	}
	for _, group := range existingGroups {
		if group.Sort > maxSortByPathSet[group.PathSet] {
			maxSortByPathSet[group.PathSet] = group.Sort
		}
	}

	for _, bucket := range buckets {
		basePath := chooseGenerateProjectPathGroupBase(bucket.key, bucket.paths)
		pathSetName := getGenerateProjectStoredPathSetName(bucket.paths)

		var group system.TbGenerateProjectPathGroup
		groupQuery := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPathGroup{}), projectId, projectInstanceId).
			Where("path_set = ? AND base_path = ?", bucket.pathSet, basePath)
		if err := groupQuery.First(&group).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			maxSortByPathSet[bucket.pathSet]++
			group = system.TbGenerateProjectPathGroup{
				ProjectId:         projectId,
				ProjectInstanceId: projectInstanceId,
				PathSet:           bucket.pathSet,
				PathSetName:       pathSetName,
				BasePath:          basePath,
				Sort:              maxSortByPathSet[bucket.pathSet],
			}
			if group.ProjectId == 0 {
				group.ProjectId = group.ProjectInstanceId
			}
			if err := tx.Create(&group).Error; err != nil {
				return err
			}
		}

		pathIds := make([]uint, 0, len(bucket.paths))
		for _, pathObj := range bucket.paths {
			pathIds = append(pathIds, pathObj.ID)
		}
		if err := tx.Model(&system.TbGenerateProjectPath{}).
			Where("id IN ?", pathIds).
			Update("path_group_id", group.ID).Error; err != nil {
			return err
		}
	}

	return s.repairStoredPathGroupsFromPathsTx(tx, projectId, projectInstanceId)
}

func (s *TbGenerateProjectPathService) repairStoredPathGroupsFromPathsTx(tx *gorm.DB, projectId int, projectInstanceId int) error {
	var groups []system.TbGenerateProjectPathGroup
	if err := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPathGroup{}), projectId, projectInstanceId).
		Order("id ASC").
		Find(&groups).Error; err != nil {
		return err
	}

	for _, group := range groups {
		var paths []system.TbGenerateProjectPath
		if err := tx.Where("path_group_id = ?", group.ID).Order("id ASC").Find(&paths).Error; err != nil {
			return err
		}
		if len(paths) == 0 {
			continue
		}
		preferredBasePath := chooseGenerateProjectPathGroupBase(inferGenerateProjectServiceBasePath(paths[0]), paths)
		currentBasePath := normalizeStoredRelativeDir(group.BasePath)
		if preferredBasePath == "" || preferredBasePath == "." || currentBasePath == preferredBasePath {
			continue
		}
		if strings.HasPrefix(currentBasePath, preferredBasePath+"/") {
			if err := tx.Model(&system.TbGenerateProjectPathGroup{}).
				Where("id = ?", group.ID).
				Update("base_path", preferredBasePath).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TbGenerateProjectPathService) DeleteTbGenerateProjectPath(req system.TbGenerateProjectPath) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	if err := tx.Where("path_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateProjectPathModel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateProjectPathService) DeletePathSet(req systemReq.DeleteGenerateProjectPathSetReq) (int, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return 0, err
	}

	pathQuery := tx.Model(&system.TbGenerateProjectPath{}).Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		pathQuery = pathQuery.Where("id IN ?", req.PathIds)
	} else {
		pathQuery = pathQuery.Where("path_set = ?", req.PathSet)
	}
	var paths []system.TbGenerateProjectPath
	if err := pathQuery.Order("id ASC").Find(&paths).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	pathIds := make([]uint, 0, len(paths))
	for _, pathObj := range paths {
		if pathObj.ID > 0 {
			pathIds = append(pathIds, pathObj.ID)
		}
	}

	deletedCount := int64(0)
	if len(pathIds) > 0 {
		modelResult := tx.Where("path_id IN ?", pathIds).Unscoped().Delete(&system.TbGenerateProjectPathModel{})
		if modelResult.Error != nil {
			tx.Rollback()
			return 0, modelResult.Error
		}
		deletedCount += modelResult.RowsAffected

		pathResult := tx.Where("id IN ?", pathIds).Unscoped().Delete(&system.TbGenerateProjectPath{})
		if pathResult.Error != nil {
			tx.Rollback()
			return 0, pathResult.Error
		}
		deletedCount += pathResult.RowsAffected
	}

	groupQuery := tx.Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.GroupIds) > 0 {
		groupQuery = groupQuery.Where("id IN ?", req.GroupIds)
	} else {
		groupQuery = groupQuery.Where("path_set = ?", req.PathSet)
	}
	result := groupQuery.Unscoped().Delete(&system.TbGenerateProjectPathGroup{})
	if result.Error != nil {
		tx.Rollback()
		return 0, result.Error
	}
	deletedCount += result.RowsAffected

	return int(deletedCount), tx.Commit().Error
}

func (s *TbGenerateProjectPathService) UpdateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	req.FileUrl = normalizeStoredRelativeDir(req.FileUrl)
	req.FileName = normalizeStoredFileName(req.FileName)
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateProjectPathService) RenamePathSet(req systemReq.RenameGenerateProjectPathSetReq) (int64, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	query := global.GVA_DB.Model(&system.TbGenerateProjectPath{}).
		Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		query = query.Where("id IN ?", req.PathIds)
	} else {
		query = query.Where("path_set = ?", req.PathSet)
	}

	nextName := strings.TrimSpace(req.PathSetName)
	result := query.Update("path_set_name", nextName)
	if result.Error != nil {
		return 0, result.Error
	}

	groupQuery := global.GVA_DB.Model(&system.TbGenerateProjectPathGroup{}).
		Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.GroupIds) > 0 {
		groupQuery = groupQuery.Where("id IN ?", req.GroupIds)
	} else if len(req.PathIds) > 0 {
		var groupIds []int
		if err := global.GVA_DB.Model(&system.TbGenerateProjectPath{}).
			Where("id IN ?", req.PathIds).
			Where("path_group_id > 0").
			Distinct("path_group_id").
			Pluck("path_group_id", &groupIds).Error; err != nil {
			return result.RowsAffected, err
		}
		if len(groupIds) == 0 {
			return result.RowsAffected, nil
		}
		groupQuery = groupQuery.Where("id IN ?", groupIds)
	} else {
		groupQuery = groupQuery.Where("path_set = ?", req.PathSet)
	}
	groupResult := groupQuery.Update("path_set_name", nextName)
	return result.RowsAffected + groupResult.RowsAffected, groupResult.Error
}

func (s *TbGenerateProjectPathService) CopyPathSet(req systemReq.CopyGenerateProjectPathSetReq) (int, error) {
	if req.ProjectInstanceId == 0 {
		req.ProjectInstanceId = req.ProjectId
	}
	if req.ProjectInstanceId == 0 {
		return 0, errors.New("projectInstanceId is required")
	}

	tx := global.GVA_DB.Begin()
	if err := tx.Error; err != nil {
		return 0, err
	}
	if err := s.ensurePathGroupsForLegacyPathsTx(tx, req.ProjectId, req.ProjectInstanceId); err != nil {
		tx.Rollback()
		return 0, err
	}

	var maxPathSet int
	if err := tx.Model(&system.TbGenerateProjectPath{}).
		Where("project_instance_id = ?", req.ProjectInstanceId).
		Select("COALESCE(MAX(path_set), 0)").
		Scan(&maxPathSet).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	var maxGroupPathSet int
	if err := tx.Model(&system.TbGenerateProjectPathGroup{}).
		Where("project_instance_id = ?", req.ProjectInstanceId).
		Select("COALESCE(MAX(path_set), 0)").
		Scan(&maxGroupPathSet).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	if maxGroupPathSet > maxPathSet {
		maxPathSet = maxGroupPathSet
	}
	nextPathSet := maxPathSet + 1

	var sourcePaths []system.TbGenerateProjectPath
	sourceQuery := tx.Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		sourceQuery = sourceQuery.Where("id IN ?", req.PathIds)
	} else {
		sourceQuery = sourceQuery.Where("path_set = ?", req.PathSet)
	}
	if err := sourceQuery.Order("id ASC").Find(&sourcePaths).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	groupIdSet := make(map[uint]struct{})
	for _, groupId := range req.GroupIds {
		if groupId > 0 {
			groupIdSet[groupId] = struct{}{}
		}
	}
	for _, sourcePath := range sourcePaths {
		if sourcePath.PathGroupId > 0 {
			groupIdSet[uint(sourcePath.PathGroupId)] = struct{}{}
		}
	}

	var sourceGroups []system.TbGenerateProjectPathGroup
	groupQuery := tx.Where("project_instance_id = ?", req.ProjectInstanceId)
	if len(groupIdSet) > 0 {
		groupIds := make([]uint, 0, len(groupIdSet))
		for groupId := range groupIdSet {
			groupIds = append(groupIds, groupId)
		}
		groupQuery = groupQuery.Where("id IN ?", groupIds)
	} else {
		groupQuery = groupQuery.Where("path_set = ?", req.PathSet)
	}
	if err := groupQuery.Order("sort ASC, id ASC").Find(&sourceGroups).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if len(sourcePaths) == 0 && len(sourceGroups) == 0 {
		tx.Rollback()
		return 0, errors.New("source paths are empty")
	}

	groupIdMap := make(map[int]int, len(sourceGroups))
	for index, sourceGroup := range sourceGroups {
		newGroup := system.TbGenerateProjectPathGroup{
			ProjectId:         sourceGroup.ProjectId,
			ProjectInstanceId: req.ProjectInstanceId,
			PathSet:           nextPathSet,
			PathSetName:       sourceGroup.PathSetName,
			BasePath:          sourceGroup.BasePath,
			Sort:              index + 1,
		}
		if newGroup.ProjectId == 0 {
			newGroup.ProjectId = req.ProjectId
		}
		if newGroup.ProjectId == 0 {
			newGroup.ProjectId = req.ProjectInstanceId
		}
		if err := tx.Create(&newGroup).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		groupIdMap[int(sourceGroup.ID)] = int(newGroup.ID)
	}

	for _, sourcePath := range sourcePaths {
		oldPathId := sourcePath.ID
		newPath := system.TbGenerateProjectPath{
			ProjectId:           sourcePath.ProjectId,
			ProjectInstanceId:   req.ProjectInstanceId,
			PathSet:             nextPathSet,
			PathSetName:         sourcePath.PathSetName,
			PathGroupId:         groupIdMap[sourcePath.PathGroupId],
			FileUrl:             sourcePath.FileUrl,
			FileName:            sourcePath.FileName,
			DynamicPlaceholders: sourcePath.DynamicPlaceholders,
			Enabled:             sourcePath.Enabled,
			Incremented:         sourcePath.Incremented,
		}
		if newPath.ProjectId == 0 {
			newPath.ProjectId = req.ProjectId
		}
		if newPath.ProjectId == 0 {
			newPath.ProjectId = req.ProjectInstanceId
		}
		if err := tx.Create(&newPath).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		var sourceModels []system.TbGenerateProjectPathModel
		if err := tx.Where("path_id = ?", oldPathId).Order("id ASC").Find(&sourceModels).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		for _, sourceModel := range sourceModels {
			newModel := system.TbGenerateProjectPathModel{
				PathId:  int(newPath.ID),
				Content: sourceModel.Content,
				Prompt:  sourceModel.Prompt,
			}
			if err := tx.Create(&newModel).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	return nextPathSet, tx.Commit().Error
}

func (s *TbGenerateProjectPathService) copyGenerateProjectPathScope(tx *gorm.DB, sourceProjectId int, sourceProjectInstanceId int, targetProjectId int, targetProjectInstanceId int) error {
	if err := s.ensurePathGroupsForLegacyPathsTx(tx, sourceProjectId, sourceProjectInstanceId); err != nil {
		return err
	}

	var sourceGroups []system.TbGenerateProjectPathGroup
	if err := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPathGroup{}), sourceProjectId, sourceProjectInstanceId).
		Order("path_set ASC, sort ASC, id ASC").
		Find(&sourceGroups).Error; err != nil {
		return err
	}

	groupIdMap := make(map[int]int, len(sourceGroups))
	for _, sourceGroup := range sourceGroups {
		newGroup := system.TbGenerateProjectPathGroup{
			ProjectId:         targetProjectId,
			ProjectInstanceId: targetProjectInstanceId,
			PathSet:           sourceGroup.PathSet,
			PathSetName:       sourceGroup.PathSetName,
			BasePath:          sourceGroup.BasePath,
			Sort:              sourceGroup.Sort,
		}
		if err := tx.Create(&newGroup).Error; err != nil {
			return err
		}
		groupIdMap[int(sourceGroup.ID)] = int(newGroup.ID)
	}

	var sourcePaths []system.TbGenerateProjectPath
	if err := applyProjectPathScope(tx.Model(&system.TbGenerateProjectPath{}), sourceProjectId, sourceProjectInstanceId).
		Order("id ASC").
		Find(&sourcePaths).Error; err != nil {
		return err
	}

	for _, sourcePath := range sourcePaths {
		oldPathId := sourcePath.ID
		newPath := system.TbGenerateProjectPath{
			ProjectId:           targetProjectId,
			ProjectInstanceId:   targetProjectInstanceId,
			PathSet:             sourcePath.PathSet,
			PathSetName:         sourcePath.PathSetName,
			PathGroupId:         groupIdMap[sourcePath.PathGroupId],
			FileUrl:             sourcePath.FileUrl,
			FileName:            sourcePath.FileName,
			DynamicPlaceholders: sourcePath.DynamicPlaceholders,
			Enabled:             sourcePath.Enabled,
			Incremented:         sourcePath.Incremented,
		}
		if err := tx.Create(&newPath).Error; err != nil {
			return err
		}

		var sourceModels []system.TbGenerateProjectPathModel
		if err := tx.Where("path_id = ?", oldPathId).Order("id ASC").Find(&sourceModels).Error; err != nil {
			return err
		}
		for _, sourceModel := range sourceModels {
			newModel := system.TbGenerateProjectPathModel{
				PathId:  int(newPath.ID),
				Content: sourceModel.Content,
				Prompt:  sourceModel.Prompt,
			}
			if err := tx.Create(&newModel).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPath(id string) (res system.TbGenerateProjectPath, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPathList(projectId int, projectInstanceId int) (res []system.TbGenerateProjectPath, err error) {
	if err = s.ensurePathGroupsForLegacyPaths(projectId, projectInstanceId); err != nil {
		return
	}
	db := global.GVA_DB.Model(&system.TbGenerateProjectPath{})
	if projectInstanceId > 0 {
		db = db.Where("project_instance_id = ?", projectInstanceId)
	} else if projectId > 0 {
		db = db.Where("project_id = ?", projectId)
	}
	err = db.Order("id ASC").Find(&res).Error
	return
}

func (s *TbGenerateProjectPathService) UpdateEnabled(id uint, enabled int) error {
	return global.GVA_DB.Model(&system.TbGenerateProjectPath{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *TbGenerateProjectPathService) BuildPromptSummary(req systemReq.BuildGenerateProjectPromptSummaryReq) (GenerateProjectPromptSummaryResult, error) {
	if req.ProjectInstanceId <= 0 {
		return GenerateProjectPromptSummaryResult{}, errors.New("projectInstanceId is required")
	}
	module := strings.TrimSpace(req.Module)
	tableName := strings.TrimSpace(req.TableName)
	if module == "" {
		return GenerateProjectPromptSummaryResult{}, errors.New("module 必填")
	}
	if tableName == "" {
		return GenerateProjectPromptSummaryResult{}, errors.New("TableName 必填")
	}

	var instance system.TbGenerateProjectInstance
	if err := global.GVA_DB.Where("id = ?", req.ProjectInstanceId).First(&instance).Error; err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	diskPathSource := instance.DiskPath
	if strings.TrimSpace(diskPathSource) == "" && instance.TemplateProjectId > 0 {
		var project system.TbGenerateProject
		if err := global.GVA_DB.Where("id = ?", instance.TemplateProjectId).First(&project).Error; err != nil {
			return GenerateProjectPromptSummaryResult{}, err
		}
		diskPathSource = project.DiskPath
	}
	diskPath, err := normalizeCodeGenerationRoot(diskPathSource)
	if err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	var paths []system.TbGenerateProjectPath
	query := global.GVA_DB.Where("project_instance_id = ? AND enabled = 1", req.ProjectInstanceId)
	if len(req.PathIds) > 0 {
		query = query.Where("id IN ?", req.PathIds)
	} else {
		query = query.Where("path_set = ?", req.PathSet)
	}
	if err := query.Order("id ASC").Find(&paths).Error; err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}
	if len(paths) == 0 {
		return GenerateProjectPromptSummaryResult{}, errors.New("当前相对路径没有可生成提示词的启用文件")
	}

	templates, err := loadGenerateProjectPathTemplates(paths)
	if err != nil {
		return GenerateProjectPromptSummaryResult{}, err
	}

	vars := mergeCodeGenerationPlaceholderValues(buildCodeGenerationVars(module, tableName), parseGeneratePlaceholderValues(instance.GeneratePlaceholderValues))
	module = vars["module"]
	tableName = vars["TableName"]
	result := GenerateProjectPromptSummaryResult{
		ProjectInstanceId:  int(instance.ID),
		ProjectName:        instance.ProjectName,
		DiskPath:           diskPath,
		PathSet:            req.PathSet,
		Module:             module,
		TableName:          tableName,
		ModifyInstructions: codeGenerationModifyInstructions,
		TargetPaths:        make([]string, 0, len(paths)),
		Files:              make([]GenerateProjectCodeFile, 0, len(paths)),
	}
	drafts := make([]generateProjectCodeDraft, 0, len(paths))

	for _, pathObj := range paths {
		relativePath, targetPath, err := renderGeneratedFileTarget(diskPath, pathObj.FileUrl, pathObj.FileName, vars)
		if err != nil {
			return GenerateProjectPromptSummaryResult{}, fmt.Errorf("路径 %d 无效: %w", pathObj.ID, err)
		}

		pathTemplate := templates[int(pathObj.ID)]
		content := renderCodeGenerationText(pathTemplate.Content, vars)
		filePrompt := renderCodeGenerationText(pathTemplate.Prompt, vars)
		file := GenerateProjectCodeFile{
			Path:         targetPath,
			AbsolutePath: targetPath,
			RelativePath: relativePath,
			PathId:       pathObj.ID,
			Status:       "ready",
			Bytes:        len([]byte(content)),
			Instruction:  buildGenerateCodeFileInstruction(relativePath, targetPath, pathObj.Incremented == 1),
		}
		result.TargetPaths = append(result.TargetPaths, targetPath)
		result.Files = append(result.Files, file)
		drafts = append(drafts, generateProjectCodeDraft{
			File:            file,
			TemplateContent: content,
			FilePrompt:      filePrompt,
			Incremented:     pathObj.Incremented == 1,
		})
	}

	result.Prompt = buildCodeGenerationTaskPromptContent(module, tableName, result.ProjectName, result.DiskPath, result.PathSet, drafts)
	return result, nil
}
