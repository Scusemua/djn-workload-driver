package cluster

import (
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
)

// Spoof a Gateway Cluster for testing.
type FakeCluster struct {
	Nodes   []*domain.KubernetesNode
	Kernels []*gateway.DistributedJupyterKernel

	logger *zap.Logger
}

func NewFakeCluster() *FakeCluster {
	cluster := &FakeCluster{
		Nodes:   make([]*domain.KubernetesNode, 0),
		Kernels: make([]*gateway.DistributedJupyterKernel, 0),
	}

	var err error
	cluster.logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	return cluster
}

// Start the FakeCluster.
func (c *FakeCluster) Start() {

}
