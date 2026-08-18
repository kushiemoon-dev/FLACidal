package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func (s *Server) handleQueueWebSocket(c *websocket.Conn) {
	id, ch := s.queueBroadcaster.Subscribe()
	defer s.queueBroadcaster.Unsubscribe(id)

	// Push the current state right away so the client isn't left waiting.
	snapshot := QueueEvent{
		Type: "snapshot",
		Jobs: s.queueBroadcaster.Snapshot(),
	}
	if err := c.WriteJSON(snapshot); err != nil {
		log.Printf("queue ws: could not write snapshot: %v", err)
		return
	}

	for event := range ch {
		if err := c.WriteJSON(event); err != nil {
			log.Printf("queue ws: write failed: %v", err)
			return
		}
	}
}

func RegisterQueueRoutes(app *fiber.App, s *Server) {
	app.Use("/ws/queue", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/queue", websocket.New(s.handleQueueWebSocket))
}
