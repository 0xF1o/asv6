package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"asv6/internal/index"
)

type searchInput struct {
	Q        string  `query:"q" maxLength:"256" example:"1110 Wien, Gasometer A/1 3.Stock 22b" doc:"Free-text address; trailing house/stair/door numbers are split off into street_number"`
	Limit    int     `query:"limit" default:"10" minimum:"1" maximum:"100" doc:"Maximum number of results"`
	MinScore float64 `query:"min_score" default:"0.15" minimum:"0" maximum:"1" doc:"Minimum score (matching trigrams / unique query trigrams)"`
}

type searchOutput struct {
	SearchTime string `header:"X-Search-Time" doc:"Server-side search duration"`
	Body       struct {
		Results []index.Result `json:"results"`
	}
}

type statusOutput struct {
	Body struct {
		Status string `json:"status" example:"rebuild started"`
	}
}

func (s *Server) registerAPI(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "search",
		Method:      http.MethodGet,
		Path:        "/search",
		Summary:     "Fuzzy address search",
		Errors:      []int{http.StatusServiceUnavailable},
	}, s.search)

	huma.Register(api, huma.Operation{
		OperationID:   "rebuild-index",
		Method:        http.MethodPost,
		Path:          "/rebuild-index",
		Summary:       "Re-download the street list and rebuild the index in the background",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusConflict},
	}, s.rebuild)
}

func (s *Server) search(_ context.Context, in *searchInput) (*searchOutput, error) {
	ix := s.idx.Load()
	if ix == nil {
		return nil, huma.Error503ServiceUnavailable("index is not ready yet")
	}
	t0 := time.Now()
	out := &searchOutput{}
	out.Body.Results = ix.Search(in.Q, in.Limit, in.MinScore)
	// ASCII only: browsers decode header values as Latin-1, which garbles Go's "µs".
	out.SearchTime = strconv.FormatFloat(time.Since(t0).Seconds()*1000, 'f', 3, 64) + "ms"
	return out, nil
}

func (s *Server) rebuild(_ context.Context, _ *struct{}) (*statusOutput, error) {
	if !s.StartRebuild() {
		return nil, huma.Error409Conflict("rebuild already in progress")
	}
	out := &statusOutput{}
	out.Body.Status = "rebuild started"
	return out, nil
}
