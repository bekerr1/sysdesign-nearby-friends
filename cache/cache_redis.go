package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"nearby-friends/types"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type CacheHandler struct {
	*redis.Client
	connInfo ConnInfo
	log      *zap.Logger
}

var _ CacheHandlerable = &CacheHandler{}

func NewRedisCacheHandler(ctx context.Context, info ConnInfo, log *zap.Logger) (CacheHandlerable, error) {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: info.Addr(),
		// Username: info.Username,
		// Password: info.Password,
		// DB:       info.DB,
	})

	// Check the connection to Redis
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	log.Info("Connected to Redis")

	return &CacheHandler{Client: client, connInfo: info, log: log}, nil
}

func (ch *CacheHandler) SetUserLocation(
	ctx context.Context,
	userLocation types.UserLocation,
) error {
	// Store the data in the cache with an expiration time of 10 minutes
	err := ch.Set(ctx, strconv.Itoa(userLocation.ID), userLocation, 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("error caching user location for user %v: %v", userLocation.ID, err)
	}
	return nil
}

func (ch *CacheHandler) GetUserLocations(
	ctx context.Context,
	users []types.User,
) ([]types.UserLocation, error) {
	userIDs := []string{}
	for _, user := range users {
		userIDs = append(userIDs, strconv.Itoa(user.ID))
	}

	result, err := ch.MGet(ctx, userIDs...).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting user locations for the set of user IDs provided: %v", err)
	}

	var locations []types.UserLocation
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &locations); err != nil {
		return nil, fmt.Errorf("error unmarshalling mget response for a set of users: %v", err)
	}

	return locations, nil
}
