package im

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"reflect"
	"socket/internal/logs"
	"sync"
	"time"
)

// Define a map to hold the WebSocket connection
var connections sync.Map

// Config allow cors
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Web Socket upgrade
func upgrade(response http.ResponseWriter, request *http.Request) (string, string, *websocket.Conn, error) {
	id, version, platform, err := bind(request)
	if err != nil {
		return "", "", nil, err
	}

	// Upgrade initial GET request to a websocket
	connection, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Fatal(err, 2)
		return "", "", nil, err
	}

	fd := uuid.New().String()

	Online(id, fd, connection.RemoteAddr().String(), platform, version)
	return id, fd, connection, nil
}

// Bind user id to connection
func bind(request *http.Request) (id, version, platform string, err error) {
	params := request.URL.Query()
	id = params.Get("id")
	version = params.Get("version")
	if version == "" {
		version = "1.0.1"
	}
	platform = params.Get("platform")
	if platform == "" {
		platform = "iOS"
	}
	if id == "" || version == "" || platform == "" {
		return id, version, platform, errors.New("authentication failure")
	}
	return id, version, platform, nil
}

// Connections handler logic
func Handle(response http.ResponseWriter, request *http.Request) {
	id, fd, connection, err := upgrade(response, request)
	if err != nil {
		response.WriteHeader(422)
		return
	}
	// Make sure we close the connection when the function returns
	defer func() {
		if connection != nil {
			if err := connection.Close(); err != nil {
				log.Printf("close connection error %v", err)
			}
		}
	}()

	logs.Handler(&logs.Payload{
		Uid:        id,
		Fd:         fd,
		Type:       "connection",
		Body:       "Connected",
		CreateTime: time.Now().String(),
		CreateDate: time.Now().Format("2006-01-02"),
		Microtime:  time.Now().UnixNano() / 1000,
	})

	// Register our new client
	connections.Store(id, connection)

	// Set read dead line
	if err := connection.SetReadDeadline(time.Now().Add(120e9)); err != nil {
		// TODO handle err
	}
	connection.SetPingHandler(func(appData string) error {
		if err := connection.SetReadDeadline(time.Now().Add(120e9)); err != nil {
			// TODO handle err
		}
		return nil
	})

	for {
		var msg Payload
		// Read in a new message as JSON and map it to a Message object
		if err := connection.ReadJSON(&msg); err != nil {
			if err := connection.Close(); err != nil {
				log.Printf("close connection error %v", err)
			}
			go logs.Handler(&logs.Payload{
				Uid:        id,
				Fd:         fd,
				Type:       "connection",
				Body:       "Disconnected",
				CreateTime: time.Now().String(),
				CreateDate: time.Now().Format("2006-01-02"),
				Microtime:  time.Now().UnixNano() / 1000,
			})
			connections.Delete(id)
			Offline(id, fd)
			goto CLOSE
		}

		if err := connection.SetReadDeadline(time.Now().Add(time.Second * 120)); err != nil {
			log.Printf("set read dead line error %v", err)
		}

	}
CLOSE:
	connection.Close()
}

func SendMessage(id string, message Payload) {
	oringin, ok := connections.Load(id)

	if ok {
		contype := reflect.ValueOf(oringin)
		connection := contype.Interface().(*websocket.Conn)
		if err := connection.WriteJSON(message); err != nil {
			log.Printf("send message error %v", err)
		}

		if message.GetAction() != "reply" {
			pushToUnconfirmedQueue(message)
		}

	} else {
		// User offline logic
	}
}

// Push message to unconfirmed queue
func pushToUnconfirmedQueue(payload Payload) {
	//boom := time.After(3e9)
	//for {
	//	select {
	//	case <-boom:
	//		if err := queue.Push(payload); err != nil {
	//			log.Println(err)
	//		}
	//		return
	//	}
	//}
}
