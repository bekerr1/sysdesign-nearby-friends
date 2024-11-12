package db

import (
	"database/sql"
	"fmt"
	"nearby-friends/types"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type mySQLDBHandler struct {
	*sql.DB
	log *zap.Logger
}

var _ DBHandler = &mySQLDBHandler{}

func NewMySQLDB(hostname, username, password, dbName string) (*sql.DB, error) {
	return sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:3306)/%v", username, password, hostname, dbName))
}

func NewMySQLDBHandler(db *sql.DB, log *zap.Logger) (DBHandler, error) {
	var err error

	// Create the users table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			user_id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create the friendships table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS friendships (
			user INT,
			friend INT,
			PRIMARY KEY (user, friend),
			FOREIGN KEY (user) REFERENCES users(user_id),
			FOREIGN KEY (friend) REFERENCES users(user_id)
		)
	`)
	if err != nil {
		return nil, err
	}

	return &mySQLDBHandler{DB: db, log: log}, nil
}

// CreateUser is idempotent
func (dh *mySQLDBHandler) CreateUser(user *types.User) error {
	result, err := dh.Exec("INSERT INTO users (username) VALUES (?)", user.Name)
	if err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			switch e.Number {
			// Duplicate entry err code
			case 1062:
				dh.log.Sugar().Debugf("user with name '%v' already exists", user.Name)
				user.ID, err = dh.GetUserIDByUsername(user.Name)
				if err != nil {
					return fmt.Errorf("error getting userID for duplicate username %v: %v", user.Name, err)
				}
				return nil
			default:
				return err
			}
		default:
			return err
		}
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = int(userID)
	return nil
}

func (dh *mySQLDBHandler) Login(name string) (*types.User, error) {
	id, err := dh.GetUserIDByUsername(name)
	if err != nil {
		return nil, fmt.Errorf("unable to get user id for name %v: %v", name, err)
	}
	return &types.User{ID: id, Name: name}, nil
}

func (dh *mySQLDBHandler) GetUserIDByUsername(username string) (int, error) {
	var userID int
	err := dh.QueryRow("SELECT user_id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("error looking up userID for username %v: %v", username, err)
	}
	return userID, nil
}

func (dh *mySQLDBHandler) EstablishFriendship(request types.FriendRequest) error {
	// Get user IDs based on usernames
	var err error
	user, friend := request.User, request.Friend
	if user.ID == 0 {
		user.ID, err = dh.GetUserIDByUsername(user.Name)
		if err != nil {
			return err
		}
	}

	if friend.ID == 0 {
		friend.ID, err = dh.GetUserIDByUsername(friend.Name)
		if err != nil {
			return err
		}
	}

	// Insert new bi-directional friendship records
	_, err = dh.Exec("INSERT INTO friendships (user, friend) VALUES (?, ?), (?, ?)",
		user.ID, friend.ID, friend.ID, user.ID)
	if err != nil {
		return fmt.Errorf("error creating a friendship between users: [%v] <-> [%v]: %v ",
			user, friend, err)
	}
	return nil
}

// ListPossibleFriends lists the users that are not this user and are not already
// firends with this user
func (dh *mySQLDBHandler) ListPossibleFriends(userID int) ([]types.User, error) {
	query := `
		SELECT u.*
		FROM users u 
		WHERE u.user_id != ? AND u.user_id NOT IN (
			SELECT f.friend     
			FROM friendships f     
			WHERE f.user = ?
		)
	`

	rows, err := dh.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var user types.User
		err := rows.Scan(&user.ID, &user.Name)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// UserCount return the number of users registered to the system
// This is just a convinience for us to determine if a user is friends with all
// other users in the system. Until we start to scale we can use this to determine
// if a locust should stop running a task.
func (dh *mySQLDBHandler) UserCount() (int, error) {
	query := `SELECT COUNT(*) FROM users;`
	var userCount int
	if err := dh.QueryRow(query).Scan(&userCount); err != nil {
		return 0, err
	}
	return userCount, nil
}

// ListUserFriends queries to get friends of a specific user
func (dh *mySQLDBHandler) ListUserFriends(userID int) ([]types.User, error) {
	query := `
		SELECT u.*
		FROM friendships f
		LEFT JOIN users u on f.friend = u.userid
		WHERE f.user = ?
	`

	rows, err := dh.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []types.User

	for rows.Next() {
		var friend types.User
		err := rows.Scan(&friend.ID, &friend.Name)
		if err != nil {
			return nil, err
		}
		friends = append(friends, friend)
	}

	return friends, nil
}
