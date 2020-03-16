package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"os"
)

func main() {
	cmd := exec.Command("/usr/bin/java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("one")
		log.Fatal(err)
	}

	cmd.Dir = "tmp"

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, "help")
	}()

	sErr := cmd.Start()
	if sErr != nil {
		fmt.Println("twp")
		log.Fatal(sErr)
	}
	log.Printf("Waiting for command to finish...")
	err = cmd.Wait()
	log.Printf("Command finished with error: %v", err)
}