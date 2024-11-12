package cache

import (
	"context"
	"fmt"
	"nearby-friends/types"

	"go.uber.org/zap"
)

type ConnInfo struct {
	Host     string
	Port     string
	Username string
	Password string
	DB       int
}

func (i ConnInfo) Addr() string {
	return fmt.Sprintf("%v:%v", i.Host, i.Port)
}

type CacheFlavor int

const (
	RedisCache CacheFlavor = iota
)

type PubSubFlavor int

const (
	RedisPubSub PubSubFlavor = iota
)

type CacheHandlerable interface {
	SetUserLocation(context.Context, types.UserLocation) error
	GetUserLocations(context.Context, []types.User) ([]types.UserLocation, error)
}

type PubSubHandlerable interface {
	BroadcastLocation(context.Context, types.UserLocation) error
	SubscribeToFriends(context.Context, []types.User, func(types.UserLocation)) error
}

func NewCacheHandler(ctx context.Context, flavor CacheFlavor, info ConnInfo, log *zap.Logger) (CacheHandlerable, error) {
	switch flavor {
	case RedisCache:
		handler, err := NewRedisCacheHandler(ctx, info, log)
		if err != nil {
			return nil, fmt.Errorf("error creating new cache handler for flavor %v: %v", flavor, err)
		}
		return handler, nil
	default:
		return nil, fmt.Errorf("unhandled cache flavor %v", flavor)
	}
}

func NewPubSubHandler(ctx context.Context, flavor PubSubFlavor, info ConnInfo, log *zap.Logger) (PubSubHandlerable, error) {
	switch flavor {
	case RedisPubSub:
		handler, err := NewRedisCacheHandler(ctx, info, log)
		if err != nil {
			return nil, fmt.Errorf("error creating new cache handler for flavor %v: %v", flavor, err)
		}
		return &PubSubHandler{CacheHandler: handler.(*CacheHandler)}, nil
	default:
		return nil, fmt.Errorf("unhandled cache flavor %v", flavor)
	}
}
