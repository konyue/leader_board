package repository

import (
	"context"
	"leader_board/model"

	"gorm.io/gorm"
)

type PlayerRepository interface {
	GetPlayerScore(ctx context.Context, playerID string) (*model.PlayerScore, error)
	GetDeletedPlayerScore(ctx context.Context, playerID string) (*model.PlayerScore, error)
	UpdatePlayerScore(ctx context.Context, score *model.PlayerScore) error
	CreateScoreLog(ctx context.Context, log *model.ScoreChangeLog) error
	BatchCreateSnapshot(ctx context.Context, snapshots []*model.LeaderboardSnapshot) error
	SoftDeletePlayer(ctx context.Context, playerID string) error
	HardDeletePlayer(ctx context.Context, playerID string) error
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db: db}
}

// GetPlayerScore 获取玩家当前分数（自动过滤软删除）
func (r *playerRepository) GetPlayerScore(ctx context.Context, playerID string) (*model.PlayerScore, error) {
	var score model.PlayerScore
	result := r.db.WithContext(ctx).Where("player_id = ?", playerID).First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}

// GetDeletedPlayerScore 获取被软删除的玩家分数
func (r *playerRepository) GetDeletedPlayerScore(ctx context.Context, playerID string) (*model.PlayerScore, error) {
	var score model.PlayerScore
	result := r.db.WithContext(ctx).Unscoped().
		Where("player_id = ? AND deleted_at IS NOT NULL", playerID).
		First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}

// UpdatePlayerScore 更新玩家分数（支持恢复软删除）
func (r *playerRepository) UpdatePlayerScore(ctx context.Context, score *model.PlayerScore) error {
	// 1. 尝试查询记录
	var existing model.PlayerScore
	result := r.db.WithContext(ctx).Unscoped(). // 包含软删除记录
							Where("player_id = ?", score.PlayerID).
							First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// 2. 记录不存在，直接创建
		return r.db.WithContext(ctx).Create(score).Error
	} else if result.Error != nil {
		// 3. 其他错误
		return result.Error
	}

	// 4. 记录存在（可能被软删除），更新字段并恢复软删除
	return r.db.WithContext(ctx).Unscoped(). // 必须用Unscoped才能更新deleted_at
							Model(&model.PlayerScore{}).
							Where("player_id = ?", score.PlayerID).
							Updates(map[string]interface{}{
			"score":            score.Score,
			"last_update_time": score.LastUpdateTime,
			"deleted_at":       nil, // 恢复软删除
		}).Error
}

// CreateScoreLog 创建分数变更日志
func (r *playerRepository) CreateScoreLog(ctx context.Context, log *model.ScoreChangeLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreateSnapshot 批量创建排行榜快照
func (r *playerRepository) BatchCreateSnapshot(ctx context.Context, snapshots []*model.LeaderboardSnapshot) error {
	return r.db.WithContext(ctx).CreateInBatches(snapshots, 1000).Error
}

// SoftDeletePlayer 软删除玩家
func (r *playerRepository) SoftDeletePlayer(ctx context.Context, playerID string) error {
	return r.db.WithContext(ctx).
		Where("player_id = ?", playerID).
		Delete(&model.PlayerScore{}).Error
}

// HardDeletePlayer 物理删除玩家（谨慎使用）
func (r *playerRepository) HardDeletePlayer(ctx context.Context, playerID string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("player_id = ?", playerID).
		Delete(&model.PlayerScore{}).Error
}
