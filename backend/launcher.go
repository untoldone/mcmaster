package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"path"

	//"os"
	"bufio"
	"io/ioutil"
	"net/http"
	"os"
)

type Launcher struct {
	WorkingDirectory string
}

type LauncherContext struct {
	Stdout chan string
	Stderr chan string
	Stdin  chan string
}

var latestMinecraftUrl = "https://launcher.mojang.com/v1/objects/bb2b6b1aefcd70dfd1892149ac3a215f6c636b07/server.jar"

// From https://golangcode.com/download-a-file-from-a-url/
// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// From https://golangcode.com/check-if-a-file-exists/
// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (l *Launcher) Run() LauncherContext {
	if l.WorkingDirectory == "" {
		l.WorkingDirectory = "tmp"
	}

	jarPath := path.Join(l.WorkingDirectory, "server.jar")
	eulaPath := path.Join(l.WorkingDirectory, "eula.txt")

	// Does folder exist?
	if _, err := os.Stat(l.WorkingDirectory); os.IsNotExist(err) {
		err := os.MkdirAll(l.WorkingDirectory, 0755)
		if err != nil {
			fmt.Println(err)
		}
	}

	if !fileExists(jarPath) {
		dErr := downloadFile(jarPath, latestMinecraftUrl)
		if dErr != nil {
			fmt.Println(dErr)
		}
	}

	if !fileExists(eulaPath) {
		eErr := ioutil.WriteFile(eulaPath, []byte("eula=true"), 0644)
		if eErr != nil {
			fmt.Println(eErr)
		}
	}

	cmd := exec.Command("/usr/bin/java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")
	cmd.Dir = l.WorkingDirectory

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("one")
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("could not get stderr pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("could not get stdout pipe: %v", err)
	}

	launcherContext := LauncherContext{
		Stdin:  make(chan string),
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
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
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
		log.Fatal(sErr)
	}
	log.Printf("Waiting for command to finish...")

	return launcherContext
}
