package redis

import (
	"time"

	"github.com/go-redis/redis"
)

// RClient ...
type RClient struct {
	client *redis.Client
}

// NewClient ....
func NewClient(addr string) (*RClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, err
	}

	return &RClient{client: client}, nil
}

// Set ...
func (rc *RClient) Set(key string, val string, dur time.Duration) error {
	return rc.client.Set(key, val, dur).Err()
}

// Get ...
func (rc *RClient) Get(key string) (string, error) {
	return rc.client.Get(key).Result()
}
