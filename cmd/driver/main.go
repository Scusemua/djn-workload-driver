package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/components"
	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/server"
)

func main() {
	conf := config.GetConfiguration()

	app.RouteFunc("/", func() app.Composer {
		mainWindow := components.NewMainWindow(components.DefaultGatewayAddress)
		// driver := driver.NewWorkloadDriver(mainWindow, conf)
		// mainWindow.SetWorkloadDriver(driver)
		return mainWindow
	})

	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// When executed on the server-side, RunWhenOnBrowser() does nothing, which
	// lets room for server implementation without the need for precompiling
	// instructions.
	app.RunWhenOnBrowser()

	// Finally, launching the server that serves the app is done by using the Go
	// standard HTTP package.
	//
	// The Handler is an HTTP handler that serves the client and all its
	// required resources to make it work into a web browser. Here it is
	// configured to handle requests with a path that starts with "/".
	http.Handle("/", &app.Handler{
		Name:               "Workload Driver",
		Title:              "Workload Driver",
		ShortName:          "Wrkld Drvr",
		LoadingLabel:       "Workload Driver for the Distributed Jupyter Notebook platform.",
		Author:             "Benjamin Carver",
		Description:        "Workload Driver for the Distributed Jupyter Notebook platform.",
		AutoUpdateInterval: time.Second * 2,
		Styles: []string{
			"/web/main.css",
			"/web/css/docs.css",
		},
		Icon: app.Icon{
			SVG: "/web/icon.svg",
		},
	})

	// Used to transfer data from the frontend to the backend.
	http.Handle("/api/k8s-nodes", server.NewKubeNodeHttpHandler(conf))

	// Used to return the config to the client.
	http.Handle("/api/config", server.NewConfigHttpHandler(conf))

	fmt.Printf("WorkloadDriver HTTP server is starting now.")

	// TODO(Ben): Make this port configurable.
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
