package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type BaseHandler struct {
	http.Handler

	Logger *zap.Logger
	opts   *config.Configuration

	BackendHttpHandler domain.BackendHttpHandler
}

func NewBaseHandler(opts *config.Configuration) *BaseHandler {
	handler := &BaseHandler{
		opts: opts,
	}

	var err error
	handler.Logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.BackendHttpHandler = handler

	return handler
}

// Write an error back to the client.
func (h *BaseHandler) WriteError(c *websocket.Conn, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	msgJSON, _ := json.Marshal(msg)

	err := c.Write(context.Background(), websocket.MessageBinary, msgJSON)
	if err != nil {
		h.Logger.Error("Error while writing error message back to front-end.", zap.String("original-error-message", errorMessage), zap.Error(err))
	}
}

func (h *BaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Trying to accept a websocket connection now.")

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.Logger.Error("Failed to accept websocket connection.", zap.Error(err))
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	h.Logger.Info("Accepted websockets connection.", zap.Any("connection", c))

	var payload map[string]interface{}
	err = wsjson.Read(ctx, c, &payload)
	if err != nil {
		h.Logger.Error("Failed to read data from websocket connection.", zap.Error(err))
		h.WriteError(c, "Failed to read any data.")
		return
	}

	// The payload will likely be a dict with a single entry "payload" containing the dict that the client sent.
	if _, ok := payload["payload"]; ok {
		payload = payload["payload"].(map[string]interface{})
	}

	h.HandleRequest(c, r, payload)

	c.Close(websocket.StatusNormalClosure, "")
}

// It would make sense to add some sort of security/verification here so that we only respond to requests from the front-end.
// But that doesn't really matter for debugging and development purposes.
func (h *BaseHandler) HandleRequest(c *websocket.Conn, r *http.Request, payload map[string]interface{}) {
	h.BackendHttpHandler.HandleRequest(c, r, payload)
}
