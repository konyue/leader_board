package model

import (
	"time"

	"gorm.io/gorm"
)

// PlayerScore 玩家当前分数表
type PlayerScore struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID       string         `gorm:"uniqueIndex;size:64" json:"player_id"` // 业务唯一ID
	Score          int64          `gorm:"not null" json:"score"`
	LastUpdateTime int64          `gorm:"not null" json:"last_update_time"` // 时间戳
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除
}

// ScoreChangeLog 分数变更日志表
type ScoreChangeLog struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID    string         `gorm:"size:64;index:idx_player" json:"player_id"`
	IncrScore   int            `gorm:"not null" json:"incr_score"`
	Timestamp   int64          `gorm:"index:idx_timestamp" json:"timestamp"`
	BeforeScore int64          `gorm:"not null" json:"before_score"`
	AfterScore  int64          `gorm:"not null" json:"after_score"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除
}

// LeaderboardSnapshot 排行榜快照表
type LeaderboardSnapshot struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	SnapshotTime time.Time      `gorm:"index:idx_snapshot_time" json:"snapshot_time"`
	PlayerID     string         `gorm:"size:64" json:"player_id"`
	Score        int64          `gorm:"not null" json:"score"`
	Rank         int            `gorm:"not null" json:"rank"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除
	// 复合唯一索引
	UniqueSnapshotPlayer struct{} `gorm:"uniqueIndex:uk_snapshot_player:SnapshotTime,PlayerID"`
}

// RankInfo 排名信息DTO
type RankInfo struct {
	PlayerID string `json:"player_id"`
	Score    int64  `json:"score"`
	Rank     int    `json:"rank"`
}
