package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"asv6/internal/server"
	"asv6/internal/source"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:38417", "listen address")
	csvURL := flag.String("csv-url", source.DefaultURL, "street list CSV URL")
	keepSonder := flag.Bool("keep-sonderadressen", false, "keep entries whose street name contains \"Sonderadresse\"")
	flag.Parse()

	srv := server.New(*csvURL, *keepSonder)
	srv.StartRebuild()

	hs := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Printf("listening on http://%s (index loading in background)", *addr)
	log.Fatal(hs.ListenAndServe())
}
