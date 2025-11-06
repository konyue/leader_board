package service

import (
	"context"
	"fmt"
	"leader_board/cache"
	"leader_board/model"
	"leader_board/repository"

	"gorm.io/gorm"
)

type LeaderboardService struct {
	repo  repository.PlayerRepository
	cache *cache.RedisCache
	db    *gorm.DB // 用于直接操作数据库（如查询软删除记录）
}

func NewLeaderboardService(repo repository.PlayerRepository, cache *cache.RedisCache, db *gorm.DB) *LeaderboardService {
	return &LeaderboardService{
		repo:  repo,
		cache: cache,
		db:    db,
	}
}

// UpdateScore 更新玩家分数
func (s *LeaderboardService) UpdateScore(ctx context.Context, playerID string, incrScore int, timestamp int64) error {
	// 1. 查询当前分数（处理软删除情况）
	currentScore, err := s.getPlayerCurrentScore(ctx, playerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 检查是否为软删除记录
			deletedScore, err := s.repo.GetDeletedPlayerScore(ctx, playerID)
			if err == nil {
				currentScore = deletedScore.Score
			} else {
				currentScore = 0 // 新玩家
			}
		} else {
			return fmt.Errorf("获取当前分数失败: %v", err)
		}
	}

	// 2. 计算新分数（防止负分）
	newScore := currentScore + int64(incrScore)
	if newScore < 0 {
		newScore = 0
	}

	// 3. 更新Redis缓存
	if err := s.cache.UpdateScore(ctx, playerID, newScore, timestamp); err != nil {
		return fmt.Errorf("更新Redis分数失败: %v", err)
	}

	// 4. 更新玩家信息缓存
	if err := s.cache.SetPlayerInfo(ctx, playerID, newScore, timestamp); err != nil {
		return fmt.Errorf("更新玩家信息缓存失败: %v", err)
	}

	// 5. 异步更新数据库
	go func() {
		ctx := context.Background()
		// 记录分数变更日志
		log := &model.ScoreChangeLog{
			PlayerID:    playerID,
			IncrScore:   incrScore,
			Timestamp:   timestamp,
			BeforeScore: currentScore,
			AfterScore:  newScore,
		}
		if err := s.repo.CreateScoreLog(ctx, log); err != nil {
			fmt.Printf("记录分数日志失败: %v\n", err)
		}

		// 更新玩家当前分数（恢复软删除）
		score := &model.PlayerScore{
			PlayerID:       playerID,
			Score:          newScore,
			LastUpdateTime: timestamp,
		}
		if err := s.repo.UpdatePlayerScore(ctx, score); err != nil {
			fmt.Printf("更新玩家分数失败: %v\n", err)
		}
	}()

	return nil
}

// GetPlayerRank 获取玩家当前排名
func (s *LeaderboardService) GetPlayerRank(ctx context.Context, playerID string) (*model.RankInfo, error) {
	// 获取排名（0开始）
	rank, err := s.cache.GetRank(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取排名失败: %v", err)
	}

	// 获取分数
	score, err := s.cache.GetPlayerScore(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取分数失败: %v", err)
	}

	return &model.RankInfo{
		PlayerID: playerID,
		Score:    score,
		Rank:     int(rank) + 1, // 转换为1开始的排名
	}, nil
}

// GetTopN 获取前N名玩家
func (s *LeaderboardService) GetTopN(ctx context.Context, n int) ([]*model.RankInfo, error) {
	if n <= 0 {
		return []*model.RankInfo{}, nil
	}

	// 获取前N名玩家ID
	playerIDs, err := s.cache.GetTopN(ctx, int64(n))
	if err != nil {
		return nil, fmt.Errorf("获取前N名失败: %v", err)
	}

	// 组装结果
	result := make([]*model.RankInfo, 0, len(playerIDs))
	for i, playerID := range playerIDs {
		score, err := s.cache.GetPlayerScore(ctx, playerID)
		if err != nil {
			continue // 忽略单个错误
		}
		result = append(result, &model.RankInfo{
			PlayerID: playerID,
			Score:    score,
			Rank:     i + 1,
		})
	}

	return result, nil
}

// GetPlayerRankRange 获取玩家周边排名
func (s *LeaderboardService) GetPlayerRankRange(ctx context.Context, playerID string, rangeNum int) ([]*model.RankInfo, error) {
	// 获取玩家当前排名（0开始）
	rank, err := s.cache.GetRank(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("获取玩家排名失败: %v", err)
	}

	// 计算查询范围
	half := rangeNum / 2
	start := rank - int64(half)
	if start < 0 {
		start = 0
	}
	end := rank + int64(half)

	// 获取范围内的玩家
	playerIDs, err := s.cache.GetRangePlayers(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("获取范围玩家失败: %v", err)
	}

	// 组装结果
	result := make([]*model.RankInfo, 0, len(playerIDs))
	for i, pid := range playerIDs {
		score, err := s.cache.GetPlayerScore(ctx, pid)
		if err != nil {
			continue
		}
		result = append(result, &model.RankInfo{
			PlayerID: pid,
			Score:    score,
			Rank:     int(start) + i + 1,
		})
	}

	return result, nil
}

// SoftDeletePlayer 软删除玩家（从排行榜移除）
func (s *LeaderboardService) SoftDeletePlayer(ctx context.Context, playerID string) error {
	// 1. 从Redis中移除
	if err := s.cache.RemovePlayer(ctx, playerID); err != nil {
		return fmt.Errorf("Redis移除玩家失败: %v", err)
	}

	// 2. 数据库软删除
	return s.repo.SoftDeletePlayer(ctx, playerID)
}

// 获取玩家当前分数（缓存优先）
func (s *LeaderboardService) getPlayerCurrentScore(ctx context.Context, playerID string) (int64, error) {
	// 先查缓存
	score, err := s.cache.GetPlayerScore(ctx, playerID)
	if err == nil {
		return score, nil
	}

	// 缓存未命中，查数据库
	playerScore, err := s.repo.GetPlayerScore(ctx, playerID)
	if err != nil {
		return 0, err
	}

	// 缓存查到的结果
	if err := s.cache.SetPlayerInfo(ctx, playerID, playerScore.Score, playerScore.LastUpdateTime); err != nil {
		fmt.Printf("缓存玩家信息失败: %v\n", err)
	}

	return playerScore.Score, nil
}
