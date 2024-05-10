package main

import (
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"path/filepath"
	"time"
)

func assertOk(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Output FS events, etc")
	flag.Parse()
	cmdToRun := flag.Args()

	if len(cmdToRun) == 0 || cmdToRun[0] == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] ./run-and-watch-cmd args\nAvailable options:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	normalizedCmd, err := filepath.Abs(cmdToRun[0])
	assertOk(err)
	cmdToRun[0] = normalizedCmd
	dirToWatch := filepath.Dir(normalizedCmd)
	if debug {
		log.Printf("Watching dir: %s", dirToWatch)
		log.Println("Starting process:", cmdToRun)
	}
	proc, err := os.StartProcess(normalizedCmd, cmdToRun,
		&os.ProcAttr{Files: []*os.File{nil, os.Stdout, os.Stderr}})
	assertOk(err)

	watcher, err := fsnotify.NewWatcher()
	assertOk(err)
	defer watcher.Close()

	err = watcher.Add(dirToWatch)
	assertOk(err)

	lastReloadMilli := time.Now().UnixMilli()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Fatal("Error receiving FS events")
			}
			if debug {
				log.Println("event:", event)
			}
			if event.Has(fsnotify.Chmod) {
				timeMilli := time.Now().UnixMilli()
				if timeMilli-lastReloadMilli > 1000 && event.Name == normalizedCmd {
					lastReloadMilli = timeMilli
					log.Println("Reloading:", event.Name)
					err = proc.Signal(os.Interrupt)
					assertOk(err)
					_, err = proc.Wait()
					assertOk(err)
					proc, err = os.StartProcess(normalizedCmd, cmdToRun,
						&os.ProcAttr{Files: []*os.File{nil, os.Stdout, os.Stderr}})
					assertOk(err)
				}
			}
		case err, _ = <-watcher.Errors:
			assertOk(err)
		}
	}
}
