package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis connection with typed get/set helpers.
type Client struct {
	rdb *redis.Client
}

func NewClient(addr, password string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})
	return &Client{rdb: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetJSON deserialises a cached value into dst. Returns (false, nil) on cache miss.
func (c *Client) GetJSON(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache get %s: %w", key, err)
	}
	if err := json.Unmarshal(val, dst); err != nil {
		return false, fmt.Errorf("cache unmarshal %s: %w", key, err)
	}
	return true, nil
}

// SetJSON serialises src and stores it with the given TTL.
func (c *Client) SetJSON(ctx context.Context, key string, src interface{}, ttl time.Duration) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("cache marshal %s: %w", key, err)
	}
	return c.rdb.Set(ctx, key, b, ttl).Err()
}
