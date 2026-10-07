package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const transferIdempotencyTTL = 24 * time.Hour

type transferIdempotencyRecord struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

func getTransferIdempotency(
	ctx context.Context,
	client *redis.Client,
	userID int64,
	idempotencyKey string,
) (*transferIdempotencyRecord, error) {
	value, err := client.Get(ctx, transferIdempotencyRedisKey(userID, idempotencyKey)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var record transferIdempotencyRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func storeTransferIdempotency(
	ctx context.Context,
	client *redis.Client,
	userID int64,
	idempotencyKey string,
	statusCode int,
	body []byte,
) error {
	record, err := json.Marshal(transferIdempotencyRecord{
		StatusCode: statusCode,
		Body:       string(body),
	})
	if err != nil {
		return err
	}

	return client.Set(
		ctx,
		transferIdempotencyRedisKey(userID, idempotencyKey),
		record,
		transferIdempotencyTTL,
	).Err()
}

func transferIdempotencyRedisKey(userID int64, idempotencyKey string) string {
	return fmt.Sprintf("idem:%d:%s", userID, idempotencyKey)
}
