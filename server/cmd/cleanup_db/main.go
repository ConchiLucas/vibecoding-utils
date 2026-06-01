package main

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 数据库配置（从 config.yaml 读取）
	dsn := "host=127.0.0.1 user=conchi password=conchi123456 dbname=vibe_project port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("=== 数据库清理脚本（移除权限系统后） ===")
	fmt.Println()

	// 需要保留的表
	keepTables := map[string]bool{
		"tb_user":              true,
	}

	// 查询数据库中所有表
	var tables []string
	err = db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename").Scan(&tables).Error
	if err != nil {
		log.Fatalf("查询表列表失败: %v", err)
	}

	fmt.Printf("数据库中共有 %d 张表\n\n", len(tables))

	var toKeep []string
	var toDrop []string
	for _, t := range tables {
		if keepTables[t] {
			toKeep = append(toKeep, t)
		} else {
			toDrop = append(toDrop, t)
		}
	}

	fmt.Printf("保留 %d 张表:\n", len(toKeep))
	for _, t := range toKeep {
		fmt.Printf("  ✅ %s\n", t)
	}

	fmt.Printf("\n删除 %d 张表:\n", len(toDrop))
	if len(toDrop) == 0 {
		fmt.Println("  无需删除的表")
		return
	}
	for _, t := range toDrop {
		fmt.Printf("  ❌ %s\n", t)
	}

	// 执行删除
	fmt.Println("\n开始删除...")
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", strings.Join(quoteAll(toDrop), ", "))
	if err := db.Exec(dropSQL).Error; err != nil {
		log.Printf("删除表失败: %v", err)
	} else {
		fmt.Printf("  ✅ 成功删除 %d 张表\n", len(toDrop))
	}

	// 验证
	var remaining []string
	db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename").Scan(&remaining)
	fmt.Printf("\n清理完成，当前剩余 %d 张表:\n", len(remaining))
	for _, t := range remaining {
		fmt.Printf("  📋 %s\n", t)
	}
}

func quoteAll(names []string) []string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = fmt.Sprintf(`"%s"`, n)
	}
	return quoted
}
