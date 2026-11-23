package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache предоставляет методы для кэширования аналитических данных
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCache создает новый экземпляр кэша
func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

// Set сохраняет значение в кэш
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// Get получает значение из кэша
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Delete удаляет значение из кэша
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// InvalidateUserCache инвалидирует все кэшированные данные пользователя
func (c *Cache) InvalidateUserCache(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("analytics:user:%s:*", userID)
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) > 0 {
		return c.Delete(ctx, keys...)
	}

	return nil
}

// MakeDailyReportKey формирует ключ для кэша дневного отчета
func MakeDailyReportKey(userID string, date time.Time) string {
	return fmt.Sprintf("analytics:user:%s:daily:%s", userID, date.Format("2006-01-02"))
}

// MakeCategoryStatsKey формирует ключ для кэша статистики по категориям
func MakeCategoryStatsKey(userID, txType string, startDate, endDate time.Time) string {
	return fmt.Sprintf("analytics:user:%s:categories:%s:%s:%s",
		userID, txType, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

// MakeTrendKey формирует ключ для кэша трендов
func MakeTrendKey(userID, txType, period string, startDate, endDate time.Time) string {
	return fmt.Sprintf("analytics:user:%s:trend:%s:%s:%s:%s",
		userID, txType, period, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}
