package components

import "github.com/maxence-charriere/go-app/v9/pkg/app"

// const (
// 	AlertTypeSuccess AlertType = "AlertType_Success"
// 	AlertTypeWarning AlertType = "AlertType_Warning"
// 	AlertTypeError   AlertType = "AlertType_Error"
// 	AlertTypeInfo    AlertType = "AlertType_Info"
// )

// type AlertType string

type Alert struct {
	app.Compo

	ID            string // Set this to uuid.New().String()
	Name          string
	Class         string
	IconClass     string
	Title         string
	Description   string
	ButtonText    string
	ButtonClass   string
	ButtonOnClick func(alertId string, ctx app.Context, e app.Event)
	OnClose       func(alertId string, ctx app.Context, e app.Event)
	HasButton     bool
}

func (a *Alert) Render() app.UI {
	return app.Li().Class("pf-v5-c-alert-group__item").Body(
		app.Div().Class(a.Class).Body(
			app.Div().Class(a.Class+"__icon").Body(
				app.I().Class(a.IconClass),
			),
			app.P().Class("pf-v5-c-alert__title").Body(
				app.Span().Class("pf-screen-reader").Text(a.Title),
			),
			app.Div().Class("pf-v5-c-alert__action").Body(
				app.Button().Class("pf-v5-c-button pf-m-plain").Type("button").Body(
					app.I().Class("fas fa-times"),
				).OnClick(func(ctx app.Context, e app.Event) {
					a.OnClose(a.ID, ctx, e)
				}),
			),
			app.Div().Class("pf-v5-c-alert__description").Body(
				app.P().Text(a.Description),
			),
			app.If(a.HasButton, app.Button().
				Class("pf-v5-c-button pf-m-primary "+a.ButtonClass).
				Type("button").
				Text(a.ButtonText).
				OnClick(func(ctx app.Context, e app.Event) {
					a.ButtonOnClick(a.ID, ctx, e)
				}),
			)),
	)
}
