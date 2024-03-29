package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	// Jupyter Server HTTP API endpoint for retrieving the Jupyter Server version.
	versionSpecJupyterServerEndpoint = "/api"

	// Jupyter Server HTTP API endpoint for retrieving the list of kernel specs.
	kernelSpecJupyterServerEndpoint = "/api/kernelspecs"
)

type KernelSpecHttpHandler struct {
	*BaseHandler

	jupyterServerAddress string // IP of the Jupyter Server.
	jupyterServerVersion string // We just obtain this when testing connectivity. It's not presently used for anything.
}

func NewKernelSpecHttpHandler(opts *config.Configuration) *KernelSpecHttpHandler {
	handler := &KernelSpecHttpHandler{
		BaseHandler:          NewBaseHandler(opts),
		jupyterServerAddress: opts.JupyterServerAddress,
	}
	handler.BackendHttpHandler = handler

	handler.Logger.Info(fmt.Sprintf("Creating server-side KernelSpecHttpHandler.\nOptions: %s", opts))

	connectivity := handler.testJupyterServerConnectivity()
	if !connectivity {
		handler.Logger.Error("Cannot connect to the Jupyter server.", zap.String("jupyter-server-ip", handler.jupyterServerAddress))
		panic("Could not connect to Jupyter server.")
	}

	return handler
}

func (h *KernelSpecHttpHandler) issueHttpRequest(target string) ([]byte, error) {
	resp, err := http.Get(target)
	if err != nil {
		h.Logger.Error("Failed to complete HTTP GET request.", zap.Error(err), zap.String("URL", target))
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.Logger.Error("Failed to read response from HTTP GET request.", zap.Error(err), zap.String("URL", target))
		return nil, err
	}

	return body, nil
}

func (h *KernelSpecHttpHandler) testJupyterServerConnectivity() bool {
	target := h.jupyterServerAddress + versionSpecJupyterServerEndpoint
	body, err := h.issueHttpRequest(target)
	if err != nil {
		return false
	}

	var version map[string]interface{}
	json.Unmarshal(body, &version)

	if versionString, ok := version["version"]; ok {
		h.jupyterServerVersion = versionString.(string)
		h.Logger.Debug("Successfully read Jupyter Server version. Connectivity established.", zap.String("version", h.jupyterServerVersion))
		return true
	} else {
		h.Logger.Error("Unexpected response from HTTP GET request to Jupyter Server /api/ endpoint.", zap.Any("response", version), zap.String("URL", target))
		return false
	}
}

func (h *KernelSpecHttpHandler) spoofKernelSpecs() []*domain.KernelSpec {
	// Distributed kernel.
	distributed_kernel := &domain.KernelSpec{
		Name:          "distributed",
		DisplayName:   "Distributed Python3",
		Language:      "python3",
		InterruptMode: "signal",
		ArgV:          []string{"/opt/conda/bin/python3", "-m", "distributed_notebook.kernel", "-f", "{connection_file}", "--debug", "--IPKernelApp.outstream_class=distributed_notebook.kernel.iostream.OutStream"},
		KernelProvisioner: &domain.KernelProvisioner{
			Name:    "gateway-provisioner",
			Gateway: "gateway:8080",
		},
	}

	// Standard Python3 kernel.
	python3_kernel := &domain.KernelSpec{
		Name:          "python3",
		DisplayName:   "Python 3 (ipykernel)",
		Language:      "python",
		InterruptMode: "signal",
		ArgV:          []string{"N/A"},
	}

	// Made-up kernel.
	ai_kernel := &domain.KernelSpec{
		Name:          "ai-kernel",
		DisplayName:   "AI-Powered Kernel",
		Language:      "all of them",
		InterruptMode: "impossible",
		ArgV:          []string{"N/A"},
	}

	return []*domain.KernelSpec{distributed_kernel, python3_kernel, ai_kernel}
}

// Retrieve the kernel specs by issuing an HTTP request to the Jupyter Server.
func (h *KernelSpecHttpHandler) getKernelSpecsFromJupyter(c *websocket.Conn) []*domain.KernelSpec {
	target := h.jupyterServerAddress + kernelSpecJupyterServerEndpoint

	body, err := h.issueHttpRequest(target)
	if err != nil {
		return nil
	}

	var kernelSpecs map[string]interface{}
	json.Unmarshal(body, &kernelSpecs)

	h.Logger.Info("Retrieved kernel specs from Jupyter Server.", zap.Any("kernel-specs", kernelSpecs))

	return make([]*domain.KernelSpec, 0)
}

func (h *KernelSpecHttpHandler) HandleRequest(c *websocket.Conn, r *http.Request, payload map[string]interface{}) {
	h.Logger.Info("Received payload from client.", zap.Any("payload", payload))

	if payload["op"] != "request-kernel-specs" {
		h.Logger.Error(fmt.Sprintf("Unexpected operation requested from client: '%s'", payload["op"]), zap.String("op", payload["op"].(string)))
		h.WriteError(c, fmt.Sprintf("Unexpected operation: %s", payload["op"].(string)))
		return
	}

	var kernelSpecs []*domain.KernelSpec

	// If we're spoofing the cluster, then just return some made up kernel specs for testing/debugging purposes.
	if h.opts.SpoofCluster {
		h.Logger.Info("Spoofing Jupyter kernel specs now.")
		kernelSpecs = h.spoofKernelSpecs()
	} else {
		h.Logger.Info("Retrieving Jupyter kernel specs from the Jupyter Server now.", zap.String("jupyter-server-ip", h.jupyterServerAddress))
		kernelSpecs = h.getKernelSpecsFromJupyter(c)

		if kernelSpecs == nil {
			// Write error back to front-end.
			h.Logger.Error("Failed to retrieve list of kernel specs from Jupyter Server.")
			h.WriteError(c, "Failed to retrieve list of kernel specs from Jupyter Server.")
			return
		}
	}

	data, err := json.Marshal(kernelSpecs)
	if err != nil {
		// Write error back to front-end.
		h.Logger.Error("Failed to marshall kernel spec objects to JSON.", zap.Error(err))
		h.WriteError(c, "Failed to marshall kernel spec objects to JSON.")
		return
	}

	h.Logger.Info("Sending kernel specs back to client now.", zap.Any("kernel-specs", kernelSpecs))
	err = c.Write(context.Background(), websocket.MessageBinary, data)
	if err != nil {
		h.Logger.Error("Error while writing kernel specs back to front-end.", zap.Error(err))
	} else {
		h.Logger.Info("Successfully sent kernel specs back to client.")
	}
}
