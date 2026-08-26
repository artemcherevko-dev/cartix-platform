package verify

import (
	"context"
	"errors"
	"time"

	redisclient "redis"

	"github.com/redis/go-redis/v9"
)

const tokenPrefix = "auth:verify:email:"

var ErrInvalidToken = errors.New("invalid or expired verification token")

type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewStore(client *redisclient.Client, ttl time.Duration) *Store {
	return &Store{
		rdb: client.RDB,
		ttl: ttl,
	}
}

func (s *Store) SaveEmailToken(ctx context.Context, token, userID string) error {
	ok, err := s.rdb.SetNX(ctx, tokenPrefix+token, userID, s.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("verification token already exists")
	}
	return nil
}

func (s *Store) ConsumeEmailToken(ctx context.Context, token string) (string, error) {
	key := tokenPrefix + token

	userID, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", err
	}

	if _, err := s.rdb.Del(ctx, key).Result(); err != nil {
		return "", err
	}

	return userID, nil
}
