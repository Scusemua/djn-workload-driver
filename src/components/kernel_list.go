package components

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type KernelList struct {
	app.Compo

	kernelProvider domain.KernelProvider
}

func NewKernelList(kernelProvider domain.KernelProvider) *KernelList {
	return &KernelList{
		kernelProvider: kernelProvider,
	}
}

func (kl *KernelList) Render() app.UI {
	return app.Div()
}
