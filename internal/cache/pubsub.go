package cache

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

const invalidationChannel = "cm:cache:invalidate"

// invalidationMsg is the message format published on the invalidation channel.
type invalidationMsg struct {
	Action string   `json:"a"` // "delete" or "flush"
	Keys   []string `json:"k,omitempty"`
}

// startSubscription listens for cache invalidation messages from other pods
// and applies them to the local in-memory cache. It runs until stopCh is closed.
func (s *Service) startSubscription() {
	s.pubsub = s.redis.Subscribe(context.Background(), invalidationChannel)

	go func() {
		ch := s.pubsub.Channel()
		for {
			select {
			case <-s.stopCh:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				s.handleInvalidation(msg)
			}
		}
	}()
}

// handleInvalidation processes a single pub/sub message.
func (s *Service) handleInvalidation(msg *redis.Message) {
	var m invalidationMsg
	if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
		slog.Debug("cache: failed to unmarshal invalidation message", "error", err)
		return
	}

	switch m.Action {
	case "delete":
		for _, key := range m.Keys {
			s.c.Delete(key)
		}
	case "flush":
		s.c.Flush()
	}
}

// publishInvalidation publishes a cache invalidation message to all pods.
func (s *Service) publishInvalidation(msg invalidationMsg) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Debug("cache: failed to marshal invalidation message", "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	if err := s.redis.Publish(ctx, invalidationChannel, data).Err(); err != nil {
		slog.Debug("cache: failed to publish invalidation", "error", err)
	}
}
