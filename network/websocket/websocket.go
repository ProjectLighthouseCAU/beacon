package websocket

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lighthouse.uni-kiel.de/lighthouse-server/auth/jwt"
	"lighthouse.uni-kiel.de/lighthouse-server/config"
	"lighthouse.uni-kiel.de/lighthouse-server/network"
	"lighthouse.uni-kiel.de/lighthouse-server/types"

	"github.com/gorilla/websocket"
)

var (
	readBufferSize     = config.GetInt("WEBSOCKET_READ_BUFFER_SIZE", 0)
	writeBufferSize    = config.GetInt("WEBSOCKET_WRITE_BUFFER_SIZE", 0)
	readLimit          = config.GetInt("WEBSOCKET_READ_LIMIT", 2048)
	certificatePath    = config.GetString("TLS_CERT_PATH", "")
	privateKeyPath     = config.GetString("TLS_PRIVATE_KEY_PATH", "")
	enableEndpointAuth = config.GetBool("WEBSOCKET_ENDPOINT_AUTHENTICATION", false) // TODO: default to true after testing
)

// Endpoint defines a websocket endpoint
type Endpoint struct { // extends BaseEndpoint implements network.Endpoint (reminder for Java-Dev)
	httpServer           *http.Server
	upgrader             websocket.Upgrader
	network.BaseEndpoint // extends
}

var _ network.Endpoint = (*Endpoint)(nil) // implements

// CreateEndpoint initiates the websocket endpoint (blocking call)
func CreateEndpoint(port int, route string, handlers []network.RequestHandler) *Endpoint {

	defer func() { // recover from any panic during initialization and retry
		if r := recover(); r != nil {
			log.Println("Error while creating websocket endpoint: ", r)
			log.Println("Retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			CreateEndpoint(port, route, handlers)
		}
	}()

	var tlsEnabled bool = certificatePath != "" && privateKeyPath != ""

	ep := &Endpoint{
		BaseEndpoint: network.BaseEndpoint{
			Type:     network.Websocket,
			Handlers: handlers,
		},
		httpServer: &http.Server{Addr: ":" + strconv.Itoa(port)},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow websocket connections from all origin domains
			},
		},
	}
	ep.httpServer.Handler = getWebsocketHandler(ep)
	go func() {
		if tlsEnabled {
			if err := ep.httpServer.ListenAndServeTLS(certificatePath, privateKeyPath); err != http.ErrServerClosed {
				log.Panicf("ListenAndServe returned: %v", err)
			}
		} else {
			if err := ep.httpServer.ListenAndServe(); err != http.ErrServerClosed {
				log.Panicf("ListenAndServe returned: %v", err)
			}
		}
	}()

	url := "ws"
	if tlsEnabled {
		url = url + "s"
	}
	url = url + "://localhost:" + strconv.Itoa(port) + route
	log.Printf("WebSocket Endpoint created: " + url)

	return ep
}

// Close closes the WebSocket Endpoint
func (ep *Endpoint) Close() {
	ep.httpServer.Close()
	log.Println("Websocket Endpoint closed")
}

// The websocket handler upgrades HTTP to WebSocket connections, creates a new server.Client for that connection
// and then reads and decodes received packets into a server.Request before forwarding to the server package.
func getWebsocketHandler(ep *Endpoint) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Error while handling websocket connection: ", r)
				log.Println("Closing...")
			}
		}()

		log.Printf("[%s] Incoming Connection: %+v\n", time.Now().Format(time.UnixDate), *request)

		claims := make(map[string]interface{})
		if enableEndpointAuth {
			authHeader := request.Header.Get("Authorization")
			if strings.TrimSpace(authHeader) == "" {
				responseWriter.WriteHeader(401)
				return
			}
			jwtStr := strings.Split(authHeader, "Bearer ")[1]
			var err error
			claims, err = jwt.ValidateJWT(jwtStr)
			if err != nil {
				responseWriter.WriteHeader(401)
				return
			}
		}

		conn, err := ep.upgrader.Upgrade(responseWriter, request, nil)
		if err != nil {
			log.Println(err)
			return
		}
		conn.SetReadLimit(int64(readLimit)) // set the maximum message size -> closes connection if exceeded

		client := types.NewClient(getSendHandle(conn), claims)

		for {
			/*messageType*/ _, payload, err := conn.ReadMessage()
			if err != nil {
				// if strings.Contains(err.Error(), "normal") {
				// 	return
				// }
				for _, h := range ep.Handlers {
					h.Disconnect(client)
				}
				log.Println(err)
				conn.Close()
				return
			}
			// TODO: uncomment this when all clients are capable of binary messages
			// if messageType != websocket.BinaryMessage {
			// 	log.Println("Received non binary message")
			// 	conn.WriteMessage(websocket.TextMessage, []byte("Bad Request - accepting only binary data encoded with MsgPack in the Lighthouse-Protocol"))
			// 	continue
			// }

			request := types.Request{}
			_, err = request.UnmarshalMsg(payload)

			if err != nil {
				log.Println(err)
				response := types.NewResponse().Reid(request.REID).Rnum(http.StatusBadRequest).Warning("Could not deserialize request. Please make sure that you are using the Lighthouse-Protocol correctly").Build()
				client.Send(response)
				continue
			}
			if len(ep.Handlers) == 0 { // no handler registered -> wrong config or startup
				response := types.NewResponse().Reid(request.REID).Rnum(http.StatusServiceUnavailable).Build()
				client.Send(response)
				continue
			}
			for _, h := range ep.Handlers {
				h.HandleRequest(client, &request)
			}
		}
	}
}

// This function wraps a reference to the websocket connection
// and a mutex lock for synchronous access to that connection into a closure
// and returns a function that takes a server.Response and writes it thread-safe to the websocket connection.
func getSendHandle(connection *websocket.Conn /*, serializer serialization.Serializer*/) func(*types.Response) error {

	var lock = &sync.Mutex{}

	return func(response *types.Response) error {
		data, err := response.MarshalMsg(nil)
		if err != nil {
			log.Println(err)
			return err
		}

		lock.Lock()
		err = connection.WriteMessage(websocket.BinaryMessage, data)
		lock.Unlock()
		if err != nil {
			log.Println(err)
			connection.Close()
		}
		return err
	}
}
