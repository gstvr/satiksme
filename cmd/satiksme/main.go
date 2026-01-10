package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gstvr/satiksme"
)

//go:embed board.gohtml
var boardTemplateHTML string

const defaultPort = 8000

var (
	flagPort       = flag.Int("port", defaultPort, fmt.Sprintf("Web server port; defaults to %d", defaultPort))
	flagConfigPath = flag.String("config", "config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := loadConfig(*flagConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Boards) == 0 {
		return fmt.Errorf("no boards configured")
	}

	client, err := satiksme.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	boardTemplate := template.Must(template.
		New("board").
		Parse(boardTemplateHTML),
	)

	s := http.NewServeMux()
	s.HandleFunc("GET /", handleIndex(cfg))
	s.HandleFunc("GET /{board_slug}", handleGetBoard(cfg, client, boardTemplate))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *flagPort),
		Handler: s,
	}
	return srv.ListenAndServe()
}
