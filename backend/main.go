package main

import (
	"flag"
	"log"
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
		  serviceContext.OutboundMessage <- processMessage
		case processError := <-launcherContext.Stderr:
		  log.Println("stderr received", processError)
		  serviceContext.OutboundMessage <- processError
		case inbound := <-serviceContext.InboundMessage:
		  log.Println("inbound message", inbound)
		  launcherContext.Stdin <- inbound
		}
	}

	log.Println("done")
}