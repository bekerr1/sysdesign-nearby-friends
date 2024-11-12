package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"nearby-friends/types"

	"github.com/go-redis/redis/v8"
)

type PubSubHandler struct {
	*CacheHandler
}

func (ch *PubSubHandler) SubscribeToFriends(
	ctx context.Context,
	friends []types.User,
	callback func(types.UserLocation),
) error {
	pubSubs := []*redis.PubSub{}
	for _, friend := range friends {
		pubSub, err := ch.subscribeToChannel(ctx, friend.ID)
		if err != nil {
			return err
		}
		pubSubs = append(pubSubs, pubSub)
	}

	for _, pubsub := range pubSubs {
		go ch.listenForUpdates(ctx, pubsub, callback)
	}

	return nil
}

func (ch *PubSubHandler) BroadcastLocation(ctx context.Context, userLocation types.UserLocation) error {
	message, err := json.Marshal(userLocation)
	if err != nil {
		return fmt.Errorf("error marshaling user location to JSON: %v", err)
	}

	channel := fmt.Sprintf("user_location:%v", userLocation.ID)
	if err := ch.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("error publishing user location to channel %v: %v", channel, err)
	}
	return nil
}

func (ch *PubSubHandler) subscribeToChannel(ctx context.Context, userID int) (*redis.PubSub, error) {
	channel := fmt.Sprintf("user_location:%v", userID)
	pubsub := ch.Subscribe(ctx, channel)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to channel '%v': %v", channel, err)
	}
	return pubsub, nil
}

func (ch *PubSubHandler) listenForUpdates(ctx context.Context, pubsub *redis.PubSub, callback func(types.UserLocation)) {
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			fmt.Println("Error receiving message:", err)
			return
		}
		var userLocationUpdate types.UserLocation
		if err := json.Unmarshal([]byte(msg.Payload), &userLocationUpdate); err != nil {
			fmt.Printf("error unmarshalling message payload '%v' recieved from pubsub: '%v'\n", msg.Payload, err)
			continue
		}

		callback(userLocationUpdate)
	}
}
