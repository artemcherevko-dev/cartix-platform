package store

import (
	"context"
	"time"

	redisclient "redis"

	"github.com/redis/go-redis/v9"
)

const receiptPrefix = "notification:receipt:processed:"

// DedupeStore remembers which notifications were already handled so that
// redelivered NATS messages or repeated Stripe webhooks do not produce
// duplicate emails.
type DedupeStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewDedupeStore(client *redisclient.Client, ttl time.Duration) *DedupeStore {
	return &DedupeStore{
		rdb: client.RDB,
		ttl: ttl,
	}
}

// MarkProcessed reserves key as "being processed". It returns true only for
// the first caller; concurrent or repeated deliveries get false.
func (s *DedupeStore) MarkProcessed(ctx context.Context, key string) (bool, error) {
	return s.rdb.SetNX(ctx, receiptPrefix+key, "1", s.ttl).Result()
}

// Release undoes MarkProcessed so a failed delivery can be retried on the
// next message redelivery.
func (s *DedupeStore) Release(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, receiptPrefix+key).Err()
}
