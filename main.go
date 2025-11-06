package main

import (
	"context"
	"fmt"
	"leader_board/cache"
	"leader_board/model"
	"leader_board/repository"
	"leader_board/service"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 初始化MySQL连接（GORM）
	// 正式业务场景 接入配置中心 配置化，笔试简略
	dsn := "root:password@tcp(localhost:3306)/game_leaderboard?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL连接失败: %v", err)
	}

	// 自动迁移数据表
	err = db.AutoMigrate(
		&model.PlayerScore{},
		&model.ScoreChangeLog{},
		&model.LeaderboardSnapshot{},
	)
	if err != nil {
		log.Fatalf("数据表迁移失败: %v", err)
	}

	// 初始化Redis缓存
	redisCache := cache.NewRedisCache("localhost:6379")

	// 初始化仓储层
	playerRepo := repository.NewPlayerRepository(db)

	// 初始化服务
	// 正式业务场景按要求使用各开发框架，笔试简略
	leaderboardService := service.NewLeaderboardService(playerRepo, redisCache, db)

	// 示例操作
	// 正式业务场景包装接口给上游/前端使用。笔试简略

	ctx := context.Background()

	// 1. 更新玩家分数
	now := time.Now().Unix()
	err = leaderboardService.UpdateScore(ctx, "player1", 150, now)
	if err != nil {
		log.Printf("更新分数失败: %v", err)
	} else {
		fmt.Println("player1 分数更新成功")
	}

	// 2. 获取玩家排名
	rank, err := leaderboardService.GetPlayerRank(ctx, "player1")
	if err != nil {
		log.Printf("获取排名失败: %v", err)
	} else {
		fmt.Printf("player1 排名: %+v\n", rank)
	}

	// 3. 获取前3名
	top3, err := leaderboardService.GetTopN(ctx, 3)
	if err != nil {
		log.Printf("获取前3名失败: %v", err)
	} else {
		fmt.Println("前3名:")
		for _, info := range top3 {
			fmt.Printf("%+v\n", info)
		}
	}

	// 4. 获取玩家周边5名
	rangePlayers, err := leaderboardService.GetPlayerRankRange(ctx, "player1", 5)
	if err != nil {
		log.Printf("获取周边玩家失败: %v", err)
	} else {
		fmt.Println("周边玩家:")
		for _, info := range rangePlayers {
			fmt.Printf("%+v\n", info)
		}
	}
}
