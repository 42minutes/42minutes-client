package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/42minutes/42minutes-server-api/models"
	"gopkg.in/alecthomas/kingpin.v1"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	app      = kingpin.New("42minutes", "CLI tool for syncing local folders with 42minutes.tv")
	apiUri   = app.Flag("api", "API URI.").Default("http://localhost:8000/").String()
	apiToken = app.Flag("token", "API User Token.").Required().String()

	scan     = app.Command("scan", "Scan tv shows folder and report to 42minutes.")
	scanPath = scan.Arg("path", "Tv show path.").Required().String()
)

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("/Users/geoah/Workspace/Geoah/corn/tests/tv")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func PushShows(shows []*models.UserFile) {
	url := *apiUri + "files"
	jsonStr, err := json.Marshal(shows)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("X-API-TOKEN", *apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)
	ioutil.ReadAll(resp.Body)
	// fmt.Println("response Body:", string(body))
	fmt.Printf("Pushed %d filenames.\n", len(shows))
}

func ScanForShows(tvpath string) {
	showFiles := make([]*models.UserFile, 0)
	filepath.Walk(tvpath, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			// TODO Log Error
			return err
		}
		if !f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			userFile := models.UserFile{}
			userFile.UserID = *apiToken
			userFile.RelativePath = filePath // TODO Remove tvpath from fileAPth
			showFiles = append(showFiles, &userFile)
			// fmt.Printf("Got %s\n", filePath)

			if len(showFiles) > 100 {
				showFilesForPush := make([]*models.UserFile, 0)
				for _, show := range showFiles {
					showFilesForPush = append(showFilesForPush, show)
				}
				PushShows(showFilesForPush)
				showFiles = make([]*models.UserFile, 0)
			}
		}
		return nil
	})
	if len(showFiles) > 0 {
		PushShows(showFiles)
	}
	fmt.Printf("Done sending filenames.\n")
}

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// Scan
	case scan.FullCommand():
		ScanForShows(*scanPath)
	}
}
