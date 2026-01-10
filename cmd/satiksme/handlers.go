package main

import (
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"time"

	"github.com/gstvr/satiksme"
)

type handler func(http.ResponseWriter, *http.Request)

func handleIndex(cfg Config) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(cfg.Boards) == 0 {
			http.Error(w, "no boards configured", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/%s", cfg.Boards[0].Slug), http.StatusFound)
	}
}

type StopWithDepartures struct {
	Name       string
	Departures []satiksme.Departure
}

type BoardView struct {
	Boards           []BoardConfig
	CurrentBoardSlug string
	Stops            []StopWithDepartures
	Now              time.Time
}

func handleGetBoard(
	cfg Config,
	client satiksme.Client,
	boardTemplate *template.Template,
) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		boardSlug := r.PathValue("board_slug")
		board, ok := cfg.BoardBySlug(boardSlug)
		if !ok {
			http.NotFound(w, r)
			return
		}

		departures, err := client.GetStopDepartures(r.Context(), board.AllStopIDs())
		if err != nil {
			// todo: consider logging the error instead of exposing internals to the user
			http.Error(w, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
			return
		}

		stops := make([]StopWithDepartures, len(board.Stops))
		for _, stopCfg := range board.Stops {
			stops = append(stops, StopWithDepartures{
				Name:       stopCfg.Name,
				Departures: filterDeparturesByStopConfig(departures, stopCfg),
			})
		}

		if err = boardTemplate.Execute(w, BoardView{
			Boards:           cfg.Boards,
			CurrentBoardSlug: boardSlug,
			Stops:            stops,
			Now:              time.Now(),
		}); err != nil {
			// todo: consider logging the error instead of exposing internals to the user
			http.Error(w, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func filterDeparturesByStopConfig(stopDepartures []satiksme.StopDepartures, stopCfg StopConfig) []satiksme.Departure {
	var departures []satiksme.Departure
	for _, sd := range stopDepartures {
		if !slices.Contains(stopCfg.StopIDs, sd.StopID) {
			continue
		}

		for _, d := range sd.Departures {
			if len(stopCfg.LineFilter) == 0 || slices.Contains(stopCfg.LineFilter, d.Line.Name()) {
				departures = append(departures, d)
			}
		}
	}

	slices.SortFunc(departures, func(a, b satiksme.Departure) int {
		return a.DepartsAt.Compare(b.DepartsAt)
	})
	return departures
}
