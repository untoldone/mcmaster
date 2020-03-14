package main

import (
	"flag"
	"log"
	"encoding/json"
	"fmt"
	//"net/http"
	//"github.com/gorilla/websocket"
	//"github.com/untoldone/mcmaster/backend/service"
)

var addr = flag.String("addr", "localhost:5000", "http service address")

func main() {
	flag.Parse()

	launcher := Launcher{}
	service := Service{}

	launcherContext := launcher.Run()
	serviceContext := service.Run(*addr)

	log.Println("waiting...")

	for {
		select {
		case processMessage := <-launcherContext.Stdout:
		  log.Println("stdout received", processMessage)

		  outboundMessage := WSAction{
		  	Action: "TerminalStdouted",
		  	Body: WSTerminalStdoutedBody{
		  		Text: processMessage,
		  	},
		  }

		  bytes, err := json.Marshal(outboundMessage)
	    if err != nil {
	        fmt.Println("Can't serislize", outboundMessage)
	    }

		  serviceContext.OutboundMessage <- string(bytes)
		case processError := <-launcherContext.Stderr:
		  log.Println("stderr received", processError)

		  outboundMessage := WSAction{
		  	Action: "TerminalStderrored",
		  	Body: WSTerminalStderroredBody{
		  		Text: processError,
		  	},
		  }

		  bytes, err := json.Marshal(outboundMessage)
	    if err != nil {
	        fmt.Println("Can't serislize", outboundMessage)
	    }

		  serviceContext.OutboundMessage <- string(bytes)
		case inbound := <-serviceContext.InboundMessage:
		  log.Println("inbound message", inbound)
		  launcherContext.Stdin <- inbound
		}
	}

	log.Println("done")
}