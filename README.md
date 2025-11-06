# 游戏排行榜系统

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Language](https://img.shields.io/badge/language-Go-blue.svg)

一个基于Go语言、GORM和Redis实现的高性能游戏排行榜系统，支持实时更新玩家积分、查询排名及周边玩家信息，并采用软删除机制保证数据安全性。

## 项目简介

该系统专为多人在线游戏设计，具备以下特点：

- 支持百万级玩家规模的实时排行榜
- 采用Redis作为缓存层，提供高性能的排名计算和查询
- 使用MySQL进行数据持久化，通过GORM实现ORM操作
- 所有数据表均支持软删除，保障数据可恢复性

## 技术栈

- 编程语言：Go 1.20+
- 缓存：Redis 6.2+
- 数据库：MySQL 8.0+
- ORM：GORM v2
- 依赖管理：Go Modules

## 项目结构

```plaintext

leaderboard/
├── model/
│   └── models.go    # 数据模型定义，包含表结构和 DTO
├── cache/
│   └── redis.go     # Redis 缓存操作，实现排行榜核心功能
├── repository/
│   └── repo.go      # 数据访问层，封装 GORM 数据库操作
├── service/
│   └── service.go   # 业务逻辑层，实现核心功能接口
├── main.go          # 程序入口，包含初始化和示例代码
└── go.mod           # 项目依赖配置
```

## 核心功能

1. **玩家分数更新**：支持增量更新玩家分数，自动处理分数为负的情况
2. **排名查询**：查询指定玩家当前排名
3. **Top N查询**：获取排行榜前N名玩家信息
4. **周边玩家查询**：查询指定玩家排名前后的玩家信息
5. **软删除功能**：支持玩家数据软删除与恢复

## 数据模型设计

### 主要数据表

1. **PlayerScore（玩家当前分数表）**
   - 存储玩家当前分数及最后更新时间
   - 包含独立主键`id`和业务ID`player_id`
   - 支持软删除（`deleted_at`字段）

2. **ScoreChangeLog（分数变更日志表）**
   - 记录玩家每次分数变更的详细日志
   - 包含变更前后分数、变更时间等信息
   - 支持数据审计和回溯

3. **LeaderboardSnapshot（排行榜快照表）**
   - 定时存储排行榜快照，减轻实时计算压力
   - 包含快照时间、玩家ID、分数和排名

### 缓存设计

- 使用Redis Sorted Set存储实时排行榜数据
- 采用负分存储实现降序排序，时间戳解决同分排序问题
- 玩家详细信息使用Hash结构缓存

## 快速开始

### 环境准备

1. 安装MySQL和Redis并启动
2. 创建数据库`game_leaderboard`

### 配置修改

在`main.go`中修改数据库和Redis连接信息：

```go
// MySQL连接配置
dsn := "root:password@tcp(localhost:3306)/game_leaderboard?charset=utf8mb4&parseTime=True&loc=Local"

// Redis连接配置
redisCache := cache.NewRedisCache("localhost:6379")

启动程序
go mod tidy
go run main.go

程序会自动创建数据表并执行示例操作。
核心接口使用示例
1. 更新玩家分数
// 更新玩家分数（playerID, 增量分数, 时间戳）
err := leaderboardService.UpdateScore(ctx, "player1", 150, time.Now().Unix())

2. 查询玩家排名
rank, err := leaderboardService.GetPlayerRank(ctx, "player1")

3. 获取前 N 名玩家
top3, err := leaderboardService.GetTopN(ctx, 3)

4. 查询玩家周边排名
// 查询玩家前后共5名玩家（包含自己）
rangePlayers, err := leaderboardService.GetPlayerRankRange(ctx, "player1", 5)

```

### 性能优化策略

- 读写分离：热点数据操作通过 Redis，持久化操作通过 MySQL
- 异步写入：分数更新时异步同步到 MySQL，不阻塞主流程
- 索引优化：所有查询字段均添加索引，包括软删除字段
- 批量操作：快照数据采用批量插入，减少数据库交互
- 缓存策略：玩家信息缓存减少数据库查询压力

### 可靠性保障

- 数据持久化：所有操作记录日志，支持数据恢复
- 软删除机制：误操作可恢复，数据安全性高
- 双重存储：Redis 缓存 + MySQL 持久化，避免单点数据丢失
- 事务支持：关键操作保证原子性
