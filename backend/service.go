package main

import (
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

type Service struct {
	
}

type ServiceContext struct {
	InboundMessage chan string
	OutboundMessage chan string
}

// checkSameOrigin returns true if the origin is not set or is equal to the request host.
func checkSameOrigin(r *http.Request) bool {
	return true
	/*origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	return equalASCIIFold(u.Host, r.Host)*/
}

var upgrader = websocket.Upgrader{
	CheckOrigin: checkSameOrigin,
} // use default options

var serviceContext = ServiceContext{
	InboundMessage: make(chan string),
	OutboundMessage: make(chan string),
}

var activeConnections = []*websocket.Conn{}

func connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	activeConnections = append(activeConnections, c)
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		if mt == websocket.TextMessage {
			serviceContext.InboundMessage <- string(message)
		}

		/*err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}*/
	}
}

func (l *Service) Run(addr string) (ServiceContext) {
	http.HandleFunc("/ws", connect)
	log.Println(addr)

	go func(){
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	go func(){
		for outboundMessage := range serviceContext.OutboundMessage {
			for _, c := range activeConnections {
				err := c.WriteMessage(websocket.TextMessage, []byte(outboundMessage))
				if err != nil {
					log.Println("write err:", err)
				}
			}
		}
	}()

	return serviceContext
}