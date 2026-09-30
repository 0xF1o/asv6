// Package server exposes the search API, rebuild endpoint and web UI.
package server

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"asv6/internal/index"
	"asv6/internal/source"
)

// Everything in web/ is compiled into the binary and served from "/".
//
//go:embed web
var webFiles embed.FS

// Server holds the current index behind an atomic pointer for lock-free swaps.
type Server struct {
	idx        atomic.Pointer[index.Index]
	rebuilding atomic.Bool
	csvURL     string
	keepSonder bool
}

func New(csvURL string, keepSonderadressen bool) *Server {
	return &Server{csvURL: csvURL, keepSonder: keepSonderadressen}
}

// Handler returns the HTTP routes; the OpenAPI spec is generated from the Go types at /openapi.json with Swagger UI at /docs.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(webFiles, "web")
	// API routes are more specific than "GET /" and therefore take precedence.
	mux.Handle("GET /", http.FileServerFS(web))
	config := huma.DefaultConfig("Austrian Street Search", "1.0.0")
	config.Info.Description = "Datenquelle: Statistik Austria — data.statistik.gv.at (CC BY 4.0, https://creativecommons.org/licenses/by/4.0/). " +
		"Changes: records are split per municipality/street-name variant, normalized for search, postal codes zero-padded, and duplicates and \"Sonderadresse\" placeholder entries removed."
	config.DocsRenderer = huma.DocsRendererSwaggerUI
	// Drop the default hook that injects "$schema" into response bodies.
	config.CreateHooks = nil
	s.registerAPI(humago.New(mux, config))
	return mux
}

// StartRebuild launches a background rebuild; returns false if one is already running.
func (s *Server) StartRebuild() bool {
	if !s.rebuilding.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer s.rebuilding.Store(false)
		t0 := time.Now()
		entries, err := source.Fetch(context.Background(), s.csvURL)
		if err != nil {
			log.Printf("rebuild failed: %v", err)
			return
		}
		if !s.keepSonder {
			entries = source.WithoutSonderadressen(entries)
		}
		ix := index.Build(entries)
		s.idx.Store(ix)
		log.Printf("index rebuilt: %d rows -> %d addresses in %s", len(entries), ix.Len(), time.Since(t0).Round(time.Millisecond))
	}()
	return true
}
