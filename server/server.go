package server

import (
	"context"
	"fmt"
	"strconv"

	//"io"
	"sync"
	"time"

	//"html/template"
	//"log"
	"encoding/json"
	"nearby-friends/cache"
	"nearby-friends/db"
	"nearby-friends/types"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{} // use default options

type Info struct {
	Host       string
	Port       string
	CACertPath string
	CAKeyPath  string
}

func (i Info) Addr() string {
	return fmt.Sprintf("%v:%v", i.Host, i.Port)
}

type RequestHandler struct {
	*mux.Router
	upgrader websocket.Upgrader
	wsConnMu sync.Mutex

	userDBHandler     db.DBHandler
	userCacheHandler  cache.CacheHandlerable
	userPubSubHandler cache.PubSubHandlerable

	userLocationByID *types.SafeMap

	log *zap.Logger
}

func NewRequestHandler(
	ctx context.Context,
	userDBHandler db.DBHandler,
	userCacheHandler cache.CacheHandlerable,
	userPubSubHandler cache.PubSubHandlerable,
	log *zap.Logger,
) *RequestHandler {
	handler := &RequestHandler{
		upgrader:          websocket.Upgrader{}, // use default options
		userDBHandler:     userDBHandler,
		userCacheHandler:  userCacheHandler,
		userPubSubHandler: userPubSubHandler,
		userLocationByID:  types.NewSafeMap(),
		log:               log,
	}
	router := mux.NewRouter()
	router.HandleFunc("/health", handler.health())
	userRoutes := router.PathPrefix("/user").Subrouter()
	userRoutes.Path("/register").Methods(http.MethodPost).HandlerFunc(handler.createUser())
	userRoutes.Path("/{id}/location").Methods(http.MethodPost).HandlerFunc(handler.updateUserLocation(ctx))
	// handler.HandleFunc("/user/{id}/location", handler.updateUserLocation(ctx))
	userRoutes.Path("/friendship").Methods(http.MethodPost).HandlerFunc(handler.createUserFriendship())
	// handler.HandleFunc("/user/friendship", handler.createUserFriendship())
	userRoutes.Path("/{id}/friends").Methods(http.MethodGet).HandlerFunc(handler.listUserFriends())
	// handler.HandleFunc("/user/{id}/friends", handler.listUserFriends())
	userRoutes.Path("/{id}/possible-friends").Methods(http.MethodGet).HandlerFunc(handler.listPossibleFriends())
	// handler.HandleFunc("/user/{id}/possible-friends", handler.listPossibleFriends())
	handler.Router = router
	return handler
}

func (wh *RequestHandler) WithMiddleware() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request
		wh.log.With(
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote-addr", r.RemoteAddr),
		).Info("Server recieved request")

		// Serve the request
		start := time.Now()
		wh.ServeHTTP(w, r)

		wh.log.With(
			zap.Duration("request-duration", time.Since(start)),
		).Info("Server issued response")
	})
}

func (wh *RequestHandler) health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/text")
		fmt.Fprintf(w, "Hello, Client!\n")
	}
}

func (wh *RequestHandler) createUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//body, err := io.ReadAll(r.Body)
		//if err != nil {
		//	http.Error(w,
		//		fmt.Sprintf("Error reading request body: %v", err),
		//		http.StatusBadRequest)
		//	return
		//}
		//wh.log.With(
		//	zap.String("body", string(body)),
		//).Info("Request body")
		//if err := json.Unmarshal(body, &user); err != nil {

		var user types.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w,
				fmt.Sprintf("Invalid request body: %v", err),
				http.StatusBadRequest)
			return
		}

		if err := wh.userDBHandler.CreateUser(&user); err != nil {
			http.Error(w,
				fmt.Sprintf("internal server error when creating user: %v", err),
				http.StatusInternalServerError)
			return
		}
		wh.log.With(
			zap.String("name", user.Name),
			zap.Int("id", user.ID),
		).Info("Created User")

		// Respond with the created user
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

func (wh *RequestHandler) createUserFriendship() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Two usernames constitue a friend request
		var friendRequest types.FriendRequest

		err := json.NewDecoder(r.Body).Decode(&friendRequest)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Invalid request body: %v", err),
				http.StatusBadRequest)
			return
		}

		if err := wh.userDBHandler.EstablishFriendship(friendRequest); err != nil {
			// fmt.Println(err)
			http.Error(w,
				fmt.Sprintf("Internal server error when creating friendship '%v': %v", friendRequest, err),
				http.StatusInternalServerError)
			return
		}

		// Respond with the created user
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(friendRequest)
	}
}

func (wh *RequestHandler) listUserFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paramUserID := "id"
		params := mux.Vars(r)
		if _, ok := params[paramUserID]; !ok {
			http.Error(w,
				fmt.Sprintf("Invalid request: missing query parameter '%v'", paramUserID),
				http.StatusBadRequest)
			return
		}

		userIDStr := params[paramUserID]
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Query param key %v with value %v is not int convertable: %v",
					paramUserID, userIDStr, err),
				http.StatusBadRequest)
			return
		}

		possibleFriends, err := wh.userDBHandler.ListUserFriends(userID)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("error getting possible friends for user %v: %v",
					userID, err),
				http.StatusInternalServerError)
			return
		}

		// Respond with the possible friend entries
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(possibleFriends)
	}
}

func (wh *RequestHandler) listPossibleFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paramUserID := "id"
		params := mux.Vars(r)
		if _, ok := params[paramUserID]; !ok {
			http.Error(w,
				fmt.Sprintf("Invalid request: missing query parameter '%v'", paramUserID),
				http.StatusBadRequest)
			return
		}

		userIDStr := params[paramUserID]
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Query param key %v with value %v is not int convertable: %v",
					paramUserID, userIDStr, err),
				http.StatusBadRequest)
			return
		}

		possibleFriends, err := wh.userDBHandler.ListPossibleFriends(userID)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("error getting possible friends for user %v: %v",
					userID, err),
				http.StatusInternalServerError)
			return
		}
		if len(possibleFriends) > 0 {
			// Respond with the possible friend entries
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(possibleFriends)
		}
	}
}

func (wh *RequestHandler) updateUserLocation(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wh.upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("Internal server error when upgrading to web socket protocol: %v", err),
				http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// Read initial message from the client.
		// This should be the first user location. We will setup the initial
		// UI with this location/userID. After, we will simply listen to further
		// user location messages and send updates on the pubsub for them.
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Unmarshal the JSON message (assuming it's a valid JSON message)
		var userLocation types.UserLocation
		if err := json.Unmarshal(p, &userLocation); err != nil {
			err = fmt.Errorf("error when marshaling user location from message '%v': %v", string(p), err)
			netErr := types.NewGenericError(err, http.StatusInternalServerError)
			jsonErr, _ := json.Marshal(netErr)
			conn.WriteMessage(websocket.TextMessage, jsonErr)
			return
		}

		// Update the user location on the cache so new users going thorugh
		// the current process can get the latest location
		if err := wh.userCacheHandler.SetUserLocation(ctx, userLocation); err != nil {
			err = fmt.Errorf("error when caching user location for user %v: %v", userLocation.ID, err)
			netErr := types.NewGenericError(err, http.StatusInternalServerError)
			jsonErr, _ := json.Marshal(netErr)
			conn.WriteMessage(websocket.TextMessage, jsonErr)
			return
		}
		wh.userLocationByID.Set(userLocation.ID, userLocation)

		// Process the initial user location.
		// This includes getting all firends, populating the initial UI,
		// and subscribing to all friend updates.
		if err := wh.processUserLocation(ctx, userLocation, conn); err != nil {
			err = fmt.Errorf("error when processing user location for user %v: %v", userLocation.ID, err)
			netErr := types.NewGenericError(err, http.StatusInternalServerError)
			jsonErr, _ := json.Marshal(netErr)
			conn.WriteMessage(websocket.TextMessage, jsonErr)
			return
		}

		// Process subsequent user locations.
		// This includes caching the updated location and boradcasting
		// the update to subscribers.
		if err := wh.readSubsequentMessages(ctx, conn); err != nil {
			err = fmt.Errorf("error when reading from web socket: %v", err)
			netErr := types.NewGenericError(err, http.StatusInternalServerError)
			jsonErr, _ := json.Marshal(netErr)
			conn.WriteMessage(websocket.TextMessage, jsonErr)
			return
		}
	}
}

func (wh *RequestHandler) readSubsequentMessages(ctx context.Context, wsConn *websocket.Conn) error {
	for {
		_, p, err := wsConn.ReadMessage()
		if err != nil {
			fmt.Println("error reading web socket message: ", err)
			continue
		}

		var userLocation types.UserLocation
		if err := json.Unmarshal(p, &userLocation); err != nil {
			fmt.Println("error unmarshalling web socket message: ", err)
			continue
		}

		if err := wh.userCacheHandler.SetUserLocation(ctx, userLocation); err != nil {
			err = fmt.Errorf("error when caching user location for user %v: %v", userLocation.ID, err)
			netErr := types.NewGenericError(err, http.StatusInternalServerError)
			jsonErr, _ := json.Marshal(netErr)
			wsConn.WriteMessage(websocket.TextMessage, jsonErr)
			continue
		}
		wh.userLocationByID.Set(userLocation.ID, userLocation)

		if err := wh.userPubSubHandler.BroadcastLocation(ctx, userLocation); err != nil {
			fmt.Println("error broadcasting user location to pubsub: ", err)
			continue
		}
	}
}

func (wh *RequestHandler) processUserLocation(
	ctx context.Context,
	userLoc types.UserLocation,
	wsConn *websocket.Conn,
) error {
	userFriends, err := wh.userDBHandler.ListUserFriends(userLoc.ID)
	if err != nil {
		return err
	}

	userLocations, err := wh.userCacheHandler.GetUserLocations(ctx, userFriends)
	if err != nil {
		return err
	}

	for _, friendLocation := range userLocations {
		if userDistance := wh.userDistanceIfValid(userLoc, friendLocation); userDistance != nil {
			if err := writeUserDistance(wsConn, *userDistance); err != nil {
				return err
			}
		}
	}
	err = wh.userPubSubHandler.SubscribeToFriends(ctx, userFriends, func(subscribedLocation types.UserLocation) {
		userID := userLoc.ID
		if location, exists := wh.userLocationByID.Get(userID); exists {
			if userDistance := wh.userDistanceIfValid(location, subscribedLocation); userDistance != nil {
				wh.wsConnMu.Lock()
				defer wh.wsConnMu.Unlock()
				if err := writeUserDistance(wsConn, *userDistance); err != nil {
					fmt.Println("error writing user distance after subscription update: ", err)
				}
			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func writeUserDistance(wsConn *websocket.Conn, distance types.UserDistance) error {
	message, err := json.Marshal(distance)
	if err != nil {
		return fmt.Errorf("error marshaling user distance to JSON: %v", err)
	}

	if err := wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("error sending user distance message: %v", err)
	}
	return nil
}

func (wh *RequestHandler) userDistanceIfValid(userLocation, friendLocation types.UserLocation) *types.UserDistance {
	distance := types.DistanceBetweenUsers(userLocation, friendLocation)
	if distance <= types.MaxDistanceBetweenUsers {
		return &types.UserDistance{
			Primary:        userLocation.User,
			Remote:         friendLocation.User,
			Distance:       distance,
			LastUpdateTime: time.Now(),
		}
	}
	return nil
}
