package main

import (
	"fmt"
	"log"
)

const (
	// Version 项目版本
	Version = "0.1.0"
	// AppName 应用名称
	AppName = "Siriusx-API"
)

func main() {
	log.Printf("=== %s v%s ===\n", AppName, Version)
	log.Println("轻量级 AI 模型聚合网关")
	log.Println("项目骨架初始化成功！")

	fmt.Println("\n🎉 项目启动成功！")
	fmt.Println("📋 当前状态: 项目骨架阶段")
	fmt.Println("🔧 下一步: 添加数据库和业务逻辑")
}
