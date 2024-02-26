package components

import (
	"github.com/elliotchance/orderedmap/v2"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type AlertList struct {
	app.Compo

	Alerts *orderedmap.OrderedMap[string, *Alert]
}

func (l *AlertList) Render() app.UI {
	keys := l.Alerts.Keys()

	app.Logf("Rendering AlertsList with %d alert(s).", len(keys))

	return app.Ul().Class("pf-v5-c-alert-group pf-m-toast").Body(
		app.Range(keys).Slice(func(idx int) app.UI {
			alertId := keys[idx]
			val, ok := l.Alerts.Get(alertId)
			if ok {
				return val
			}
			return app.Div()
		}))
}
