package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	leaderboardKey   = "leaderboard:main"
	playerInfoPrefix = "player:info:"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

// UpdateScore 更新玩家分数到有序集合
func (c *RedisCache) UpdateScore(ctx context.Context, playerID string, score int64, timestamp int64) error {
	// 计算Redis排序值：负分确保降序，时间戳确保同分按时间排序
	redisScore := -score*1000000000 - timestamp
	return c.client.ZAdd(ctx, leaderboardKey, &redis.Z{
		Score:  float64(redisScore),
		Member: playerID,
	}).Err()
}

// SetPlayerInfo 存储玩家信息到哈希表
func (c *RedisCache) SetPlayerInfo(ctx context.Context, playerID string, score int64, timestamp int64) error {
	key := fmt.Sprintf("%s%s", playerInfoPrefix, playerID)
	return c.client.HSet(ctx, key, map[string]interface{}{
		"score":       score,
		"update_time": timestamp,
	}).Err()
}

// GetRank 获取玩家排名（0开始）
func (c *RedisCache) GetRank(ctx context.Context, playerID string) (int64, error) {
	return c.client.ZRank(ctx, leaderboardKey, playerID).Result()
}

// GetTopN 获取前N名玩家
func (c *RedisCache) GetTopN(ctx context.Context, n int64) ([]string, error) {
	return c.client.ZRange(ctx, leaderboardKey, 0, n-1).Result()
}

// GetPlayerScore 获取玩家分数
func (c *RedisCache) GetPlayerScore(ctx context.Context, playerID string) (int64, error) {
	key := fmt.Sprintf("%s%s", playerInfoPrefix, playerID)
	val, err := c.client.HGet(ctx, key, "score").Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// GetRangePlayers 获取玩家周围的排名
func (c *RedisCache) GetRangePlayers(ctx context.Context, start, end int64) ([]string, error) {
	return c.client.ZRange(ctx, leaderboardKey, start, end).Result()
}

// RemovePlayer 从缓存中移除玩家（用于软删除场景）
func (c *RedisCache) RemovePlayer(ctx context.Context, playerID string) error {
	// 1. 从有序集合中删除
	if err := c.client.ZRem(ctx, leaderboardKey, playerID).Err(); err != nil {
		return err
	}

	// 2. 从哈希表中删除
	key := fmt.Sprintf("%s%s", playerInfoPrefix, playerID)
	return c.client.Del(ctx, key).Err()
}
