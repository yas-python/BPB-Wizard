package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	workerURL = "https://raw.githubusercontent.com/yas-python/zizifn/main/_worker.js"
)

func init() {
	// parse flags
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()
	if *showVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// initialize globals, paths, DNS, Android checks
	initGlobals()
	initPaths()
	setDNS()
	checkAndroid()
}

func main() {
	// render ASCII header
	renderHeader()

	var wg sync.WaitGroup
	wg.Add(1)

	// run the wizard in a goroutine
	go func() {
		defer wg.Done()
		runWizard()
	}()

	// start local HTTP server for OAuth callback
	server := &http.Server{Addr: ":8976"}
	http.HandleFunc("/oauth/callback", callback)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			failMessage("Error serving localhost.")
			log.Fatalln(err)
		}
	}()

	// wait for wizard to finish
	wg.Wait()

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
