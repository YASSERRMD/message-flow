package realtime

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWS(w http.ResponseWriter, r *http.Request, hub *Hub, tenantID int64, userID int64) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{TenantID: tenantID, UserID: userID, Send: make(chan []byte, 16)}
	hub.Register(client)

	go writePump(conn, client, hub)
	readPump(conn, client, hub)
}

func readPump(conn *websocket.Conn, client *Client, hub *Hub) {
	defer func() {
		hub.Unregister(client)
		_ = conn.Close()
	}()
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var message map[string]any
		if err := json.Unmarshal(payload, &message); err != nil {
			continue
		}
		if msgType, ok := message["type"].(string); ok && msgType == "typing" {
			message["user_id"] = client.UserID
			hub.Broadcast(client.TenantID, message)
		}
	}
}

func writePump(conn *websocket.Conn, client *Client, hub *Hub) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		hub.Unregister(client)
		_ = conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
