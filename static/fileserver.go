package static

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ProjectLighthouseCAU/beacon/config"
)

var (
	webinterfaceHost  = config.GetString("WEBINTERFACE_HOST", "127.0.0.1")
	webinterfaceRoute = config.GetString("WEBINTERFACE_ROUTE", "/")
	webinterfacePort  = config.GetInt("WEBINTERFACE_PORT", 3001)
)

func StartFileserver() {
	// serve static testing site (only works with websocket endpoint enabled)
	log.Println("Serving static files: " + "http://localhost:" + strconv.Itoa(webinterfacePort) + webinterfaceRoute)
	mux := http.NewServeMux()
	mux.Handle(webinterfaceRoute, http.FileServer(http.Dir("./static")))
	serv := &http.Server{
		Addr:    webinterfaceHost + ":" + fmt.Sprint(webinterfacePort),
		Handler: mux,
	}
	go func() {
		log.Fatal(serv.ListenAndServe())
	}()
}
