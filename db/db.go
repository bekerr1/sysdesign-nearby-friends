package db

import (
	"fmt"
	"nearby-friends/types"

	"go.uber.org/zap"
)

type ConnInfo struct {
	Hostname string
	Username string
	Password string
	DBName   string
}

type Flavor int

const (
	MySQL Flavor = iota
)

type DBHandler interface {
	// User mgmt
	Login(name string) (*types.User, error)
	CreateUser(user *types.User) error

	// Friendships
	ListUserFriends(userID int) ([]types.User, error)
	ListPossibleFriends(userID int) ([]types.User, error)
	EstablishFriendship(request types.FriendRequest) error
}

func NewDBHandler(dbFlavor Flavor, dbInfo ConnInfo, log *zap.Logger) (DBHandler, error) {
	switch dbFlavor {
	case MySQL:
		db, err := NewMySQLDB(dbInfo.Hostname, dbInfo.Username, dbInfo.Password, dbInfo.DBName)
		if err != nil {
			return nil, fmt.Errorf("error connecting to db for flavor '%v': %v", dbFlavor, err)
		}
		return NewMySQLDBHandler(db, log)
	default:
		return nil, fmt.Errorf("unhandled db flavor: %v", dbFlavor)
	}
}
