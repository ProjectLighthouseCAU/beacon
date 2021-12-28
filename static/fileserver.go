package static

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"lighthouse.uni-kiel.de/lighthouse-server/config"
)

var (
	webinterfaceRoute = config.GetString("WEBINTERFACE_ROUTE", "/")
	webinterfacePort  = config.GetInt("WEBINTERFACE_PORT", 3001)
)

func StartFileserver() {
	// serve static testing site (only works with websocket endpoint enabled)
	log.Println("Serving static files: " + "http://localhost:" + strconv.Itoa(webinterfacePort) + webinterfaceRoute)
	mux := http.NewServeMux()
	mux.Handle(webinterfaceRoute, http.FileServer(http.Dir("./static")))
	serv := &http.Server{
		Addr:    ":" + fmt.Sprint(webinterfacePort),
		Handler: mux,
	}
	go func() {
		log.Fatal(serv.ListenAndServe())
	}()
}
