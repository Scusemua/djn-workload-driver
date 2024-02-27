package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type KubeNodeHttpHandler struct {
	http.Handler

	metricsClient *metrics.Clientset
	clientset     *kubernetes.Clientset
	logger        *zap.Logger
}

func NewKubeNodeHttpHandler(opts *config.Configuration) *KubeNodeHttpHandler {
	handler := &KubeNodeHttpHandler{}

	var err error
	handler.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	handler.logger.Info("Creating server-side KubeNodeHttpHandler.", zap.String("options", opts.String()))

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

		metricsConfig, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		metricsClient, err := metrics.NewForConfig(metricsConfig)
		if err != nil {
			panic(err)
		}

		handler.clientset = clientset
		handler.metricsClient = metricsClient
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

		metricsConfig, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
		if err != nil {
			panic(err.Error())
		}

		metricsClient, err := metrics.NewForConfig(metricsConfig)
		if err != nil {
			panic(err)
		}

		handler.clientset = clientset
		handler.metricsClient = metricsClient
	}

	handler.logger.Info("Successfully created server-side HTTP handler.")

	return handler
}

// Write an error back to the client.
func (h *KubeNodeHttpHandler) writeError(c *websocket.Conn, errorMessage string) {
	// Write error back to front-end.
	msg := &domain.ErrorMessage{
		ErrorMessage: errorMessage,
		Valid:        true,
	}
	msgJSON, _ := json.Marshal(msg)

	h.logger.Info("Writing error message back to client.", zap.Any("error-message", msg))

	err := c.Write(context.Background(), websocket.MessageBinary, msgJSON)
	if err != nil {
		h.logger.Error("Error while writing error message back to front-end.", zap.String("original-error-message", errorMessage), zap.Error(err))
	}
}

func (h *KubeNodeHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if payload["op"] != "request-nodes" {
		h.logger.Error("Unexpected operation requested from client.", zap.String("op", payload["op"].(string)))
		h.writeError(c, fmt.Sprintf("Unexpected operation: %s", payload["op"].(string)))
		return
	}

	spoofNodes := payload["spoof-nodes"].(bool)

	h.logger.Info("", zap.Bool("spoof-nodes", spoofNodes))

	nodes, err := h.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Error("Failed to retrieve nodes from Kubernetes.", zap.Error(err))
		h.writeError(c, "Failed to retrieve nodes from Kubernetes.")
		return
	}

	nodeUsageMetrics, err := h.metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		h.logger.Error("Failed to retrieve node metrics from Kubernetes.", zap.Error(err))
		h.writeError(c, "Failed to retrieve node metrics from Kubernetes.")
		return
	}

	h.logger.Info(fmt.Sprintf("Sending a list of %d nodes back to the client.", len(nodes.Items)), zap.Int("num-nodes", len(nodes.Items)))

	var kubernetesNodes map[string]*domain.KubernetesNode = make(map[string]*domain.KubernetesNode, len(nodes.Items))
	val := nodes.Items[0].Status.Capacity[corev1.ResourceCPU]
	val.AsInt64()
	for _, node := range nodes.Items {
		allocatableCPU := node.Status.Capacity[corev1.ResourceCPU]
		allocatableMemory := node.Status.Capacity[corev1.ResourceMemory]

		allocCpu := allocatableCPU.AsApproximateFloat64()
		allocMem := allocatableMemory.AsApproximateFloat64()

		h.logger.Info("Memory as inf.Dec.", zap.String("node-id", node.Name), zap.Any("mem inf.Dec", allocatableMemory.AsDec().String()))

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		pods, err := h.clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + node.Name,
		})

		if err != nil {
			h.logger.Error("Could not retrieve Pods running on node.", zap.String("node", node.Name), zap.Error(err))
		}

		var kubePods []*domain.KubernetesPod
		if pods != nil {
			kubePods = make([]*domain.KubernetesPod, 0, len(pods.Items))

			for _, pod := range pods.Items {
				kubePod := &domain.KubernetesPod{
					PodName:  pod.ObjectMeta.Name,
					PodPhase: string(pod.Status.Phase),
					PodIP:    pod.Status.PodIP,
					PodAge:   time.Since(pod.GetCreationTimestamp().Time).Round(time.Second),
				}

				kubePods = append(kubePods, kubePod)
			}
		}

		sort.Slice(kubePods, func(i, j int) bool {
			return kubePods[i].PodName < kubePods[j].PodName
		})

		kubernetesNode := domain.KubernetesNode{
			NodeId:         node.Name,
			CapacityCPU:    allocCpu,
			CapacityMemory: allocMem / 976600.0, // Convert from Ki to GB.
			Pods:           kubePods,
			Age:            time.Since(node.GetCreationTimestamp().Time).Round(time.Second),
			IP:             node.Status.Addresses[0].Address,
			// CapacityGPUs:    0,
			// CapacityVGPUs:   0,
			// AllocatedCPU:    0,
			// AllocatedMemory: 0,
			// AllocatedGPUs:   0,
			// AllocatedVGPUs:  0,
		}

		kubernetesNodes[node.Name] = &kubernetesNode
	}

	for _, nodeMetric := range nodeUsageMetrics.Items {
		nodeName := nodeMetric.ObjectMeta.Name
		kubeNode := kubernetesNodes[nodeName]
		h.logger.Info("Node metric.", zap.String("node", nodeName), zap.Any("metric", nodeMetric))

		cpu := nodeMetric.Usage.Cpu().AsApproximateFloat64()
		// if !ok {
		// 	h.logger.Error("Could not convert CPU usage metric to Int64.", zap.Any("cpu-metric", nodeMetric.Usage.Cpu()))
		// }
		h.logger.Info("CPU metric.", zap.String("node-id", nodeName), zap.Float64("cpu", cpu))

		mem := nodeMetric.Usage.Memory().AsApproximateFloat64()
		// if !ok {
		// 	h.logger.Error("Could not convert 	memory usage metric to Int64.", zap.Any("mem-metric", nodeMetric.Usage.Memory()))
		// }
		h.logger.Info("Memory metric.", zap.String("node-id", nodeName), zap.Float64("memory", cpu))

		kubeNode.AllocatedCPU = cpu
		kubeNode.AllocatedMemory = mem / 976600.0 // Convert from Ki to GB.

		kubernetesNodes[nodeName] = kubeNode
	}

	for _, node := range kubernetesNodes {
		h.logger.Info("Kubernetes node.", zap.String(node.NodeId, node.String()))
	}

	data, err := json.Marshal(kubernetesNodes)
	if err != nil {
		h.logger.Error("Failed to marshall nodes from Kubernetes to JSON.", zap.Error(err))

		// Write error back to front-end.
		h.writeError(c, "Failed to marshall nodes to JSON.")

		return
	}

	h.logger.Info("Sending nodes back to client now.")
	err = c.Write(context.Background(), websocket.MessageBinary, data)
	if err != nil {
		h.logger.Error("Error while writing node list back to front-end.", zap.Error(err))
	} else {
		h.logger.Info("Successfully sent config back to client.")
	}

	c.Close(websocket.StatusNormalClosure, "")
}
