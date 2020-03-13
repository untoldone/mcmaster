package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	//"os"
	"bufio"
)

type Launcher struct {

}

type LauncherContext struct {
	Stdout chan string
	Stderr chan string
	Stdin chan string	
}

func (l *Launcher) Run() (LauncherContext) {
	cmd := exec.Command("/usr/bin/java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("one")
		log.Fatal(err)
	}

	cmd.Dir = "/Users/untoldone/Downloads/tmp"

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("could not get stderr pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("could not get stdout pipe: %v", err)
	}


	launcherContext := LauncherContext{
		Stdin: make(chan string),
		Stderr: make(chan string),
		Stdout: make(chan string),
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			msg := scanner.Text()
			launcherContext.Stderr <- msg
		}
	}()

	go func() {
		fmt.Println("one")
		scanner := bufio.NewScanner(stdout)
		fmt.Println("two")
		for scanner.Scan() {
			fmt.Println("three")
			msg := scanner.Text()
			fmt.Println(msg)
			launcherContext.Stdout <- msg
		}
	}()

	go func() {
		for inputLine := range launcherContext.Stdin {
			log.Println("Got for input:", inputLine)
			io.WriteString(stdin, inputLine)
			io.WriteString(stdin, "\n")
		}
	}()

	sErr := cmd.Start()
	if sErr != nil {
		fmt.Println("twp")
		log.Fatal(sErr)
	}
	log.Printf("Waiting for command to finish...")

	return launcherContext
}