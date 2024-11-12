package types

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type GenericError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewGenericError(err error, code int) error {
	return &GenericError{
		Message: err.Error(),
		Code:    code,
	}
}

// Error implements the error interface for HTTPError.
func (e *GenericError) Error() string {
	return fmt.Sprintf("%s [%d]", e.Message, e.Code)
}

// MarshalJSON marshals the error information into a JSON string
func (e *GenericError) MarshalJSON() ([]byte, error) {
	return json.Marshal(*e)
}

// User represents a user on the system
type User struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`
}

func (u *User) String() string {
	return fmt.Sprintf("%v - %v", u.ID, u.Name)
}

// FriendRequest is a request for a User
// to establish a firendship with a Friend
type FriendRequest struct {
	User   User `json:"user"`
	Friend User `json:"friend"`
}

// UserLocation represents a possible location a User is at.
// This struct is a composition of a User and said location metadata.
type UserLocation struct {
	*User
	Longitude      float64   `json:"longitude"`
	Latitude       float64   `json:"latitude"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

func (u UserLocation) radialLatitude() float64 {
	return float64(math.Pi * u.Latitude / 180)
}

func theta(user1, user2 UserLocation) float64 {
	return user1.Longitude - user2.Longitude
}

var MaxDistanceBetweenUsers float64 = 5

// DistanceBetweenUsers calculates the distance between the primary user
// (assuming thats 'our' location) and the secondary user (assuming that the 'remote' location)
func DistanceBetweenUsers(primary, secondary UserLocation) float64 {
	theta := theta(primary, secondary)
	radtheta := float64(math.Pi * theta / 180)

	dist := math.Sin(primary.radialLatitude())*
		math.Sin(secondary.radialLatitude()) +
		math.Cos(primary.radialLatitude())*
			math.Cos(secondary.radialLatitude())*
			math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515

	return dist
}

type UserDistance struct {
	Primary        *User     `json:"primary"`
	Remote         *User     `json:"remote"`
	Distance       float64   `json:"distance"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

type SafeMap struct {
	mu sync.RWMutex
	m  map[int]UserLocation
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		m: make(map[int]UserLocation),
	}
}

func (s *SafeMap) Set(key int, value UserLocation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *SafeMap) Get(key int) (UserLocation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.m[key]
	return value, ok
}
