package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"message-flow/backend/internal/db"
	"message-flow/backend/internal/realtime"
)

type Queue struct {
	client *redis.Client
}

type QueueMessage struct {
	TenantID  int64     `json:"tenant_id"`
	MessageID int64     `json:"message_id"`
	Content   string    `json:"content"`
	Feature   string    `json:"feature"`
	CreatedAt time.Time `json:"created_at"`
}

func NewQueue(redisURL string) (*Queue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	return &Queue{client: client}, nil
}

func (q *Queue) Enqueue(ctx context.Context, message QueueMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, queueKey(message.TenantID), payload).Err()
}

func (q *Queue) DequeueBatch(ctx context.Context, tenantID int64, batchSize int) ([][]byte, error) {
	key := queueKey(tenantID)
	var items [][]byte
	for i := 0; i < batchSize; i++ {
		item, err := q.client.RPop(ctx, key).Bytes()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

func queueKey(tenantID int64) string {
	return "llm:queue:" + fmtInt(tenantID)
}

func fmtInt(value int64) string {
	return fmt.Sprintf("%d", value)
}

type Worker struct {
	Queue     *Queue
	Service   *Service
	DB        *db.Store
	Hub       *realtime.Hub
	BatchSize int
}

func (w *Worker) Start(ctx context.Context, tenantID int64) {
	batch := w.BatchSize
	if batch <= 0 {
		batch = 100
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		items, err := w.Queue.DequeueBatch(ctx, tenantID, batch)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if len(items) == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, raw := range items {
			var msg QueueMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
			result, err := w.Service.AnalyzeWithFallback(ctxTimeout, msg.TenantID, msg.Content, &msg.MessageID)
			cancel()
			if err == nil {
				_ = StoreAnalysis(ctx, w.DB, msg.TenantID, msg.MessageID, result)
				if w.Hub != nil {
					w.Hub.Broadcast(msg.TenantID, map[string]any{
						"type":       "message.analysis",
						"message_id": msg.MessageID,
					})
				}
			}
		}
	}
}
