package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// handleQueueWebSocket upgrades the connection and streams QueueEvents to the client.
func (s *Server) handleQueueWebSocket(c *websocket.Conn) {
	id, ch := s.queueBroadcaster.Subscribe()
	defer s.queueBroadcaster.Unsubscribe(id)

	// Send initial snapshot so the client has current state immediately.
	snapshot := QueueEvent{
		Type: "snapshot",
		Jobs: s.queueBroadcaster.Snapshot(),
	}
	if err := c.WriteJSON(snapshot); err != nil {
		log.Printf("queue ws: snapshot write error: %v", err)
		return
	}

	// Forward events until the client disconnects.
	for event := range ch {
		if err := c.WriteJSON(event); err != nil {
			log.Printf("queue ws: write error: %v", err)
			return
		}
	}
}

// RegisterQueueRoutes attaches the /ws/queue endpoint to the given Fiber app.
//
// TODO: call RegisterQueueRoutes from server.go's setupRoutes()
func RegisterQueueRoutes(app *fiber.App, s *Server) {
	app.Use("/ws/queue", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/queue", websocket.New(s.handleQueueWebSocket))
}
