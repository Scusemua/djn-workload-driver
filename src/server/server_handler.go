package server

import (
	"net/http"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ServerHttpHandler struct {
	http.Handler

	clientset *kubernetes.Clientset
	logger    *zap.Logger
}

func NewServerHttpHandler(opts *config.Configuration) *ServerHttpHandler {
	handler := &ServerHttpHandler{}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.logger.Info("Creating server-side HTTP handler.", zap.String("options", opts.String()))

	if opts.InCluster {
		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		handler.clientset = clientset
	} else {
		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		handler.clientset = clientset
	}

	return handler
}

func (h *ServerHttpHandler) ServeHTTP(respWriter http.ResponseWriter, req *http.Request) {

}
