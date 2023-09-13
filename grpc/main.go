package grpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/CyCoreSystems/ari/v5"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	grpc_engine "google.golang.org/grpc"
)

var addr = "0.0.0.0:8018"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"events"}, // <-- add this line
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func processEvents(c *websocket.Conn, clientId string, wsChan <-chan *ClientEvent, stopChan <-chan bool) {
	defer func() {
		helpers.Log(logrus.InfoLevel, "Closing event processor...")
		c.Close()
	}()

	for {
		select {
		case evt := <-wsChan:
			log.Println("Received client event...")
			if clientId != evt.ClientId {
				continue
			}

			mt := websocket.TextMessage
			b, err := json.MarshalIndent(&evt, "", "\t")
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "Error marshaling JSON:"+err.Error())
				continue // Skip sending this event on error
			}

			err = c.WriteMessage(mt, b)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "Error writing message:"+err.Error())
				return // Terminate the goroutine on error
			}

		case <-stopChan:
			helpers.Log(logrus.InfoLevel, "Closing event processor...")
			return
		}
	}
}

func ws(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	stopChan := make(chan bool)
	clientId := v.Get("clientId")
	wsChan := createWSChan(clientId)
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("got connection from: %s\r\n", clientId))
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("Req: %s %s\n", r.Host, r.URL.Path))
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		helpers.Log(logrus.InfoLevel, "upgrade:"+err.Error())
		return
	}
	go processEvents(c, clientId, wsChan, stopChan)
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			helpers.Log(logrus.InfoLevel, "read:"+err.Error())
			stopChan <- true
			c.Close()
			break
		}
		// helpers.Log(logrus.InfoLevel, fmt.Sprintf("recv: %s", message))
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "OK\n")
}

func startWebsocketServer() {
	http.HandleFunc("/", ws)
	http.HandleFunc("/healthz", healthz)
	helpers.Log(logrus.FatalLevel, http.ListenAndServe(addr, nil).Error())
}

func StartListener(cl ari.Client) {
	return
	wsChan := make(chan *ClientEvent)
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 9000))
	if err != nil {
		helpers.Log(logrus.FatalLevel, fmt.Sprintf("failed to listen: %v", err))
	}

	go startWebsocketServer()
	fmt.Println("GRPC is running!!")
	s := NewServer(cl, wsChan)

	grpcServer := grpc_engine.NewServer()

	RegisterLineblocsServer(grpcServer, s)

	if err := grpcServer.Serve(lis); err != nil {
		helpers.Log(logrus.FatalLevel, fmt.Sprintf("failed to serve: %s", err))
	}
}
