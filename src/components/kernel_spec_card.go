package components

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

// Display the different kernel specs available on the server.
type KernelSpecCard struct {
	app.Compo

	id                      string
	KernelSpecProvider      domain.KernelSpecProvider
	KernelSpecs             []*domain.KernelSpec
	selectedKernelSpecIndex int
}

func NewKernelSpecCard(kernelSpecProvider domain.KernelSpecProvider) *KernelSpecCard {
	card := &KernelSpecCard{
		KernelSpecProvider:      kernelSpecProvider,
		id:                      uuid.New().String(),
		selectedKernelSpecIndex: 0,
	}

	card.KernelSpecs = card.KernelSpecProvider.Resources()
	return card
}

func (c *KernelSpecCard) handleKernelSpecsRefreshed(kernelSpecs []*domain.KernelSpec) bool {
	if !c.Mounted() {
		app.Logf("KernelSpecCard %s (%p) is not mounted; ignoring refresh.", c.id, c)
		return false
	}

	app.Logf("KernelSpecCard %s (%p) is handling a kernelSpecs refresh. Number of kernelSpecs: %d.", c.id, c, len(kernelSpecs))

	// We sort these in the provider.
	// sort.Slice(kernelSpecs, func(i int, j int) bool {
	// 	return kernelSpecs[i].Name < kernelSpecs[j].Name
	// })

	// refreshedExpanded := make(map[string]bool, len(kernelSpecs))

	// for _, node := range kernelSpecs {
	// 	var expanded, ok bool

	// 	if expanded, ok = c.expanded[node.NodeId]; ok {
	// 		refreshedExpanded[node.NodeId] = expanded
	// 	} else {
	// 		expanded = false
	// 		refreshedExpanded[node.NodeId] = false
	// 	}
	// }

	// c.expanded = refreshedExpanded

	c.KernelSpecs = kernelSpecs
	c.Update()

	return true
}

func (c *KernelSpecCard) OnMount(ctx app.Context) {
	c.KernelSpecProvider.SubscribeToRefreshes(c.id, c.handleKernelSpecsRefreshed)

	go c.KernelSpecProvider.RefreshResources()
}

func (c *KernelSpecCard) OnDismount(ctx app.Context) {
	c.KernelSpecProvider.UnsubscribeFromRefreshes(c.id)
}

func (c *KernelSpecCard) Render() app.UI {
	kernelSpecs := c.KernelSpecs
	app.Logf("Rendering KernelSpecCard (%p) with %d kernel spec(s). selectedKernelSpecIndex: %d.", c, len(kernelSpecs), c.selectedKernelSpecIndex)

	// If they've not loaded in yet, then just render the header of the card.
	if len(kernelSpecs) == 0 {
		return app.Div().Class("pf-v5-c-card").Body(
			// Header
			app.Div().Class("pf-v5-c-card__header").Body(
				app.Div().Class("pf-v5-c-card__header-main").Body(
					app.H2().Class("pf-v5-c-title pf-m-lg").Body().Text("Kernel Specs"),
				),
			))
	}

	return app.Div().Class("pf-v5-c-card").Body(
		// Header
		app.Div().Class("pf-v5-c-card__header").Body(
			app.Div().Class("pf-v5-c-card__header-main").Body(
				app.H2().Class("pf-v5-c-title pf-m-lg").Body().Text("Kernel Specs"),
			),
		),
		// Tabs
		app.Div().Class("pf-v5-c-card__body").Body(
			app.Div().Class("pf-v5-c-tabs pf-m-fill").Role("region").ID("spec-tabs").Body(
				app.Ul().Class("pf-v5-c-tabs__list").Role("tablist").Body(
					app.Range(kernelSpecs).Slice(func(i int) app.UI {
						if i == c.selectedKernelSpecIndex {
							return app.Li().Class("pf-v5-c-tabs__item pf-m-current").Role("presentation").Body(
								app.Button().Class("pf-v5-c-tabs__link").Type("button").Role("tab").ID(fmt.Sprintf("kernel-spec-tab-%s", kernelSpecs[i].Name)).Body(
									app.Span().Class("pf-v5-c-tabs__item-text").Text(kernelSpecs[i].Name),
								).OnClick(func(ctx app.Context, e app.Event) {
									app.Logf("Changing c.selectedKernelSpecIndex from %d to %d. (%p)", c.selectedKernelSpecIndex, i, c)
									c.selectedKernelSpecIndex = i
									c.Update()
								}, c.selectedKernelSpecIndex),
							)
						} else {
							return app.Li().Class("pf-v5-c-tabs__item").Role("presentation").Body(
								app.Button().Class("pf-v5-c-tabs__link").Type("button").Role("tab").ID(fmt.Sprintf("kernel-spec-tab-%s", kernelSpecs[i].Name)).Body(
									app.Span().Class("pf-v5-c-tabs__item-text").Text(kernelSpecs[i].Name),
								).OnClick(func(ctx app.Context, e app.Event) {
									app.Logf("Changing c.selectedKernelSpecIndex from %d to %d. (%p)", c.selectedKernelSpecIndex, i, c)
									c.selectedKernelSpecIndex = i
									c.Update()
								}, c.selectedKernelSpecIndex),
							)
						}
					}),
				),
			),
		),
		// Tab Body
		app.Div().Class("pf-v5-c-card__body").Body(
			app.Section().Class("pf-v5-c-tab-content").ID(fmt.Sprintf("kernel-spec-tab%d-panel", c.selectedKernelSpecIndex)).Role("tabpanel").TabIndex(c.selectedKernelSpecIndex).Body(
				app.Div().Class("pf-v5-c-tab-content__body").Body(
					app.Div().Class("pf-v5-c-description-list pf-m-compact pf-m-horizontal pf-m-2-col").Body(
						app.Raw(fmt.Sprintf(
							`<div class="pf-v5-c-description-list__group">
									<dt class="pf-v5-c-description-list__term">%s</dt>
									<dd class="pf-v5-c-description-list__description">
										<div class="pf-v5-c-description-list__text">
											<p>%s</p>
										</div>
									</dd>
								</div>`, "Name", kernelSpecs[c.selectedKernelSpecIndex].Name)),
						app.Raw(fmt.Sprintf(
							`<div class="pf-v5-c-description-list__group">
										<dt class="pf-v5-c-description-list__term">%s</dt>
										<dd class="pf-v5-c-description-list__description">
											<div class="pf-v5-c-description-list__text">
												<p>%s</p>
											</div>
										</dd>
									</div>`, "Display Name", kernelSpecs[c.selectedKernelSpecIndex].DisplayName)),
						app.Raw(fmt.Sprintf(
							`<div class="pf-v5-c-description-list__group">
											<dt class="pf-v5-c-description-list__term">%s</dt>
											<dd class="pf-v5-c-description-list__description">
												<div class="pf-v5-c-description-list__text">
													<p>%s</p>
												</div>
											</dd>
										</div>`, "Language", kernelSpecs[c.selectedKernelSpecIndex].Language)),
						app.Raw(fmt.Sprintf(
							`<div class="pf-v5-c-description-list__group">
												<dt class="pf-v5-c-description-list__term">%s</dt>
												<dd class="pf-v5-c-description-list__description">
													<div class="pf-v5-c-description-list__text">
														<p>%s</p>
													</div>
												</dd>
											</div>`, "Interrupt Mode", kernelSpecs[c.selectedKernelSpecIndex].InterruptMode)),
						// If this kernel spec has a KernelProvisioner, then we'll display the KernelProvisioner's name field here.
						// If not, then we'll omit the associated UI entirely.
						app.If(kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner != nil,
							app.Raw(
								fmt.Sprintf(
									`<div class="pf-v5-c-description-list__group">
												<dt class="pf-v5-c-description-list__term">%s</dt>
												<dd class="pf-v5-c-description-list__description">
													<div class="pf-v5-c-description-list__text">
														<p>%s</p>
													</div>
												</dd>
											</div>`, "Provisioner Name", func(p *domain.KernelProvisioner) string {
										// We use an anonymous function call here to get the name in the event that it's null.
										// We have to do this because app.If() still evaluates the UI element passed to it even when the condition is false.
										if kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner == nil {
											return ""
										} else {
											return kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner.Name
										}
									}(kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner)),
							),
							// If this kernel spec has a KernelProvisioner, then we'll display the KernelProvisioner's gateway field here.
							// If not, then we'll omit the associated UI entirely.
							app.If(kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner != nil,
								app.Raw(
									fmt.Sprintf(
										`<div class="pf-v5-c-description-list__group">
												<dt class="pf-v5-c-description-list__term">%s</dt>
												<dd class="pf-v5-c-description-list__description">
													<div class="pf-v5-c-description-list__text">
														<p>%s</p>
													</div>
												</dd>
											</div>`, "Provisioner Gateway", func(p *domain.KernelProvisioner) string {
											// We use an anonymous function call here to get the name in the event that it's null.
											// We have to do this because app.If() still evaluates the UI element passed to it even when the condition is false.
											if kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner == nil {
												return ""
											} else {
												return kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner.Gateway
											}
										}(kernelSpecs[c.selectedKernelSpecIndex].KernelProvisioner)),
								),
							),
						),
					),
				)),
		),
	)
}
