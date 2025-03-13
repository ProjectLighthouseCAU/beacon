package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/handler"
	"github.com/ProjectLighthouseCAU/beacon/network"
	"github.com/ProjectLighthouseCAU/beacon/types"

	"github.com/gorilla/websocket"
)

// Endpoint defines a websocket endpoint
type Endpoint struct { // extends BaseEndpoint implements network.Endpoint (reminder for Java-Dev)
	httpServer           *http.Server
	upgrader             websocket.Upgrader
	network.BaseEndpoint // extends
	connectedClients     map[*types.Client]*websocket.Conn
	clientsLock          sync.Mutex
}

var _ network.Endpoint = (*Endpoint)(nil) // implements

// CreateEndpoint initiates the websocket endpoint (blocking call)
func CreateEndpoint(host string, port int, auth auth.Auth, handler *handler.Handler) *Endpoint {

	defer func() { // recover from any panic during initialization and retry
		if r := recover(); r != nil {
			log.Println("Error while creating websocket endpoint: ", r)
			log.Println("Retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			CreateEndpoint(host, port, auth, handler)
		}
	}()

	ep := &Endpoint{
		BaseEndpoint: network.BaseEndpoint{
			Type:    network.Websocket,
			Auth:    auth,
			Handler: handler,
		},
		httpServer: &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.WebsocketReadBufferSize,
			WriteBufferSize: config.WebsocketWriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow websocket connections from all origin domains
			},
		},
		connectedClients: make(map[*types.Client]*websocket.Conn),
	}
	ep.httpServer.Handler = ep.getWebsocketHandler()
	go func() {
		if err := ep.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Panicf("ListenAndServe returned: %v", err)
		}
	}()

	log.Printf("WebSocket Endpoint created: ws://%s:%d", host, port)

	return ep
}

// Close closes the WebSocket Endpoint
func (ep *Endpoint) Close() {
	log.Println("Closing websocket endpoint")
	ep.clientsLock.Lock()
	for client, conn := range ep.connectedClients {
		log.Println("Disconnecting ", client.Ip())
		ep.disconnectClient(client, conn,
			&websocket.CloseError{Code: websocket.CloseServiceRestart,
				Text: closeCodeToCloseMsg[websocket.CloseServiceRestart]},
			true)
	}
	ep.connectedClients = make(map[*types.Client]*websocket.Conn)
	ep.clientsLock.Unlock()
	log.Println("Clients disconnected")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := ep.httpServer.Shutdown(ctx)
	if err != nil {
		log.Println("Failed to gracefully shutdown websocket endpoint:", err)
		ep.httpServer.Close()
		log.Println("Forcefully closed websocket endpoint")
	}
	log.Println("Websocket endpoint closed")
}

var closeCodeToCloseMsg = map[int]string{
	websocket.CloseNormalClosure:   "Bye!",
	websocket.CloseGoingAway:       "Bye!",
	websocket.CloseProtocolError:   "Received malformed websocket frame",
	websocket.CloseUnsupportedData: "Data type unsupported (must be binary frame)",
	// websocket.CloseNoStatusReceived:        "", // connection broken
	// websocket.CloseAbnormalClosure:         "", // connection broken
	websocket.CloseInvalidFramePayloadData: "Please use our MessagePack based protocol!",
	// websocket.ClosePolicyViolation:         "", // unused
	websocket.CloseMessageTooBig:      fmt.Sprintf("Your message is too big! Limit: %d bytes", config.WebsocketReadLimit),
	websocket.CloseMandatoryExtension: "This server does not support your requested websocket extension!",
	websocket.CloseInternalServerErr:  "Oops, something bad happened on our side... sorry",
	websocket.CloseServiceRestart:     "The server ist currently restarting, hang on tight and retry later!",
	websocket.CloseTryAgainLater:      "Sorry, the server is temporarily unavailable. Please try again later!",
	// websocket.CloseTLSHandshake:            "", // connection broken
}

func (ep *Endpoint) disconnectClient(client *types.Client, conn *websocket.Conn, err error, alreadyLocked bool) {
	if err != nil {
		closeErr := &websocket.CloseError{}
		var closeMsg []byte
		if errors.As(err, &closeErr) {
			if msg, ok := closeCodeToCloseMsg[closeErr.Code]; ok {
				log.Println("Sending close message:", closeErr.Code, closeErr.Text)
				closeMsg = websocket.FormatCloseMessage(closeErr.Code, closeErr.Text+": "+msg)
			} else {
				log.Println("Sending close message:", websocket.CloseInternalServerErr, err.Error())
				closeMsg = websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
			}
		} else {
			log.Println("Sending close message:", websocket.CloseInternalServerErr, err.Error())
			closeMsg = websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
		}
		deadline := time.Now().Add(time.Second)
		err = conn.WriteControl(websocket.CloseMessage, closeMsg, deadline)
		if err != nil {
			log.Println("cannot write closemessage:", err)
			// TODO: fix
		}
	}
	if !alreadyLocked {
		ep.clientsLock.Lock()
		delete(ep.connectedClients, client)
		ep.clientsLock.Unlock()
	}
	client.Disconnect(ep.Handler.GetDirectory())
	time.Sleep(1 * time.Second) //TODO: remove and find smarter way
	conn.Close()
	log.Println("Client disconnected: ", client.Ip())
}

// The websocket handler upgrades HTTP to WebSocket connections, creates a new Client for that connection
// and then reads and decodes received packets into a Request before forwarding to the handler package.
func (ep *Endpoint) getWebsocketHandler() http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Error while handling websocket connection: ", r)
				log.Println("Closing...")
			}
		}()

		clientIp := request.Header.Get("X-Real-Ip")
		if clientIp == "" {
			clientIp = request.Header.Get("X-Forwarded-For")
		}
		if clientIp == "" {
			clientIp = request.RemoteAddr
		}

		log.Printf("Incoming Connection from: %s\n", clientIp)

		conn, err := ep.upgrader.Upgrade(responseWriter, request, nil)
		if err != nil {
			log.Println(err)
			return
		}
		conn.SetReadLimit(int64(config.WebsocketReadLimit)) // set the maximum message size -> closes connection if exceeded

		client := types.NewClient(clientIp, getSendFunc(conn))
		ep.clientsLock.Lock()
		ep.connectedClients[client] = conn
		ep.clientsLock.Unlock()

		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				ep.disconnectClient(client, conn, err, false)
				return
			}

			if messageType != websocket.BinaryMessage {
				response := types.NewResponse().Reid([]byte{0}).Rnum(http.StatusBadRequest).Warning("Non binary-type message received, use websocket binary-type instead").Build()
				client.Send(response)
				closeCode := websocket.CloseUnsupportedData
				ep.disconnectClient(client, conn, &websocket.CloseError{Code: closeCode, Text: closeCodeToCloseMsg[closeCode]}, false)
				return
			}
			request := types.Request{}
			_, err = request.UnmarshalMsg(payload)
			if err != nil {
				response := types.NewResponse().Reid(request.REID).Rnum(http.StatusBadRequest).Warning("Could not deserialize request. Please make sure that you are using the Lighthouse-Protocol correctly").Build()
				client.Send(response)
				closeCode := websocket.CloseInvalidFramePayloadData
				ep.disconnectClient(client, conn, &websocket.CloseError{Code: closeCode, Text: closeCodeToCloseMsg[closeCode]}, false)
				return
			}

			// authentication and authorization
			if ok, code := ep.Auth.IsAuthorized(client, &request); !ok {
				response := types.NewResponse().Reid(request.REID).Rnum(code).Build()
				client.Send(response)
				// TODO: decide when to disconnect client connection (without any authentication after timeout?)
				// if code == http.StatusUnauthorized {
				// 	disconnectClient()
				// 	return
				// }
				continue
			}
			// call handler
			ep.Handler.HandleRequest(client, &request)
		}
	}
}

// This function wraps a reference to the websocket connection
// and a mutex lock for synchronous access to that connection into a closure
// and returns a function that takes a server.Response and writes it thread-safe to the websocket connection.
func getSendFunc(connection *websocket.Conn) func(*types.Response) error {

	var lock = &sync.Mutex{}

	return func(response *types.Response) error {
		data, err := response.MarshalMsg(nil)
		if err != nil {
			return err
		}

		lock.Lock()
		err = connection.WriteMessage(websocket.BinaryMessage, data)
		lock.Unlock()
		if err != nil {
			connection.Close()
		}
		return err
	}
}
