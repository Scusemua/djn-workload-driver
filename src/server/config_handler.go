package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type ConfigHttpHandler struct {
	*BaseHandler
}

func NewConfigHttpHandler(opts *config.Configuration) *ConfigHttpHandler {
	handler := &ConfigHttpHandler{
		BaseHandler: NewBaseHandler(opts),
	}
	handler.BackendHttpHandler = handler

	handler.Logger.Info(fmt.Sprintf("Creating server-side ConfigHttpHandler.\nOptions: %s", opts))

	return handler
}

func (h *ConfigHttpHandler) HandleRequest(c *websocket.Conn, r *http.Request, payload map[string]interface{}) {
	h.Logger.Info("Received payload from client.", zap.Any("payload", payload))

	if payload["op"] != "request-config" {
		h.Logger.Error(fmt.Sprintf("Unexpected operation requested from client: '%s'", payload["op"]), zap.String("op", payload["op"].(string)))
		h.WriteError(c, fmt.Sprintf("Unexpected operation: %s", payload["op"].(string)))
		return
	}

	data, err := json.Marshal(h.opts)
	if err != nil {
		h.Logger.Error("Failed to marshall configuration object to JSON.", zap.Error(err))

		// Write error back to front-end.
		h.WriteError(c, "Failed to marshall configuration object to JSON.")

		return
	}

	h.Logger.Info("Sending config back to client now.", zap.Any("config", h.opts))
	err = c.Write(context.Background(), websocket.MessageBinary, data)
	if err != nil {
		h.Logger.Error("Error while writing configuration object back to front-end.", zap.Error(err))
	} else {
		h.Logger.Info("Successfully sent config back to client.")
	}
}
