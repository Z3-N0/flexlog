package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Z3-N0/flexlog"
)

const defaultPort = 8080

type Params struct {
	Path string
	Port int
}

func parseArgs() (*Params, error) {
	path := flag.String("path", "", "path to a log file or directory of log files")
	port := flag.Int("port", defaultPort, "port to serve the viewer on")
	flag.Parse()

	if *path == "" {
		return nil, fmt.Errorf("--path is required")
	}
	if _, err := os.Stat(*path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if *port < 1 || *port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", *port)
	}

	return &Params{
		Path: *path,
		Port: *port,
	}, nil
}

func main() {
	params, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flexlog-viewer: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	logger := flexlog.New()
	defer logger.Close()

	if err := start(params, logger); err != nil {
		log.Fatalf("flexlog-viewer: %v", err)
	}
}
