package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type ConfigHttpHandler struct {
	http.Handler
	logger *zap.Logger
	opts   *config.Configuration
}

func NewConfigHttpHandler(opts *config.Configuration) *ConfigHttpHandler {
	handler := &ConfigHttpHandler{
		opts: opts,
	}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.logger.Info("Creating server-side ConfigHttpHandler.", zap.String("options", opts.String()))

	return handler
}

// Write an error back to the client.
func (h *ConfigHttpHandler) writeError(c *websocket.Conn, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	msgJSON, _ := json.Marshal(msg)

	err := c.Write(context.Background(), websocket.MessageBinary, msgJSON)
	if err != nil {
		h.logger.Error("Error while writing error message back to front-end.", zap.String("original-error-message", errorMessage), zap.Error(err))
	}
}

func (h *ConfigHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Trying to accept a websocket connection now.")

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to accept websocket connection.", zap.Error(err))
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	h.logger.Info("Accepted websockets connection.", zap.Any("connection", c))

	var payload map[string]interface{}
	err = wsjson.Read(ctx, c, &payload)
	if err != nil {
		h.logger.Error("Failed to read data from websocket connection.", zap.Error(err))
		h.writeError(c, "Failed to read any data.")
		return
	}

	// The payload will likely be a dict with a single entry "payload" containing the dict that the client sent.
	if _, ok := payload["payload"]; ok {
		payload = payload["payload"].(map[string]interface{})
	}

	h.logger.Info("Received payload from client.", zap.Any("payload", payload))

	if payload["op"] != "request-config" {
		h.logger.Error(fmt.Sprintf("Unexpected operation requested from client: '%s'", payload["op"]), zap.String("op", payload["op"].(string)))
		h.writeError(c, fmt.Sprintf("Unexpected operation: %s", payload["op"].(string)))
		return
	}

	data, err := json.Marshal(h.opts)
	if err != nil {
		h.logger.Error("Failed to marshall configuration object to JSON.", zap.Error(err))

		// Write error back to front-end.
		h.writeError(c, "Failed to marshall configuration object to JSON.")

		return
	}

	h.logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	err = c.Write(context.Background(), websocket.MessageBinary, data)
	if err != nil {
		h.logger.Error("Error while writing configuration object back to front-end.", zap.Error(err))
	} else {
		h.logger.Info("Successfully sent config back to client.")
	}

	c.Close(websocket.StatusNormalClosure, "")
}
