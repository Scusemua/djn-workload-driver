package driver

import "net/http"

type WorkloadDriver interface {
	http.Handler

	// Start the HTTP server.
	// Should be called within its own goroutine.
	StartHttpServer()

	// Start the WorkloadDriver
	Start()
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}
