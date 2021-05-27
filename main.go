package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"

	"golang.org/x/sys/windows/registry"
)

const (
	SHADOWPLAY_PATH = `Software\NVIDIA CORPORATION\Global\ShadowPlay\NVSPCAPS`
	STEAM_PATH      = `Software\Valve\Steam\Apps`
	TARGET_SIZE_MB  = 8
)

var currentFilename, WEBHOOK_URL string

func main() {
	file, err := os.Open("webhook.txt")
	if err != nil {
		log.Fatal(err)
	}
	fileContents := make([]byte, 135)
	file.Read(fileContents)
	WEBHOOK_URL = string(fileContents)
	tf2Directory := (getClipPath() + `\Team Fortress 2`)
	startWatching(tf2Directory)
}

func startWatching(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Watching for changes %s\n", path)

	watcher.Add(path)
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("Event: ", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Modified file: ", event.Name)
					fileCreated(event, path)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error", err)
			}
		}
	}()
	<-done
}

func fileCreated(event fsnotify.Event, path string) {
	filename := strings.SplitAfter(event.Name, `"`)[0]
	if currentFilename != filename {
		// shadowplay event always triggers twice, this ensures only the second event is handled
		currentFilename = filename
	} else {
		cmd := exec.Command("ffmpeg", `-i`, filename, "-vf", "scale=-1:720", "-c:v", "libx264", "-crf", "17", "-preset", "veryslow", "-c:a", "copy", "output.mp4", "-y")
		fmt.Println(cmd.Args)
		var out bytes.Buffer
		var outErr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &outErr
		err := cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + outErr.String())
			return
		}
		sendFile("output.mp4")
	}
}

func sendFile(file string) {
	readFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer readFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("mp4", file)

	if err != nil {
		log.Fatal(err)
	}

	io.Copy(part, readFile)
	writer.Close()
	request, err := http.NewRequest("POST", WEBHOOK_URL, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	wd, err := os.Getwd()
	os.Remove(fmt.Sprintf("%s%s", wd, "output.mp4"))

	fmt.Println(string(content))
}

func getClipPath() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, SHADOWPLAY_PATH, registry.QUERY_VALUE)
	val, _, err := k.GetBinaryValue("DefaultPathW")
	if err != nil {
		log.Fatal(err)
	}
	final := bytes.ReplaceAll(val, []byte("\x00"), []byte(""))
	defer k.Close()
	return string(final)
}

func getClipLengthSeconds() (int64, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, SHADOWPLAY_PATH, registry.QUERY_VALUE)
	defer k.Close()
	val, _, err := k.GetBinaryValue("DVRBufferLen")
	if err != nil {
		log.Fatal(err)
	}
	final := bytes.ReplaceAll(val, []byte("\x00"), []byte(""))
	intStr := hex.EncodeToString(final)
	return strconv.ParseInt(intStr, 10, 64)
}
