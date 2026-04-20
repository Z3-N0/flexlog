//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Config
var (
	TAILWIND     = "tailwindcss"
	INPUT_CSS    = "web/static/input.css"
	OUTPUT_CSS   = "web/static/tailwind.min.css"
	TEMPLATE_DIR = "web/templates/**/*.html"

	GO     = "go"
	BINARY = "flexlog-viewer"
	CMD    = "./cmd/viewer"
)

// ==== Helpers ====

func run(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("binary not found: %s: %w", name, err)
	}
	cmd := exec.Command(path, args...) // nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func rm(path string) error {
	return os.RemoveAll(path)
}

// ==== Targets ====

// Prepare: clean, css, and tidy (skip build)
func Prepare() error {
	if err := Clean(); err != nil {
		return err
	}
	if err := Css(); err != nil {
		return err
	}
	return Gotidy()
}

// Dev: clean → css → build → tidy
func Dev() error {
	fmt.Println("|==> Starting viewer dev build ===|")
	if err := Clean(); err != nil {
		return err
	}
	if err := Css(); err != nil {
		return err
	}
	if err := Build(); err != nil {
		return err
	}
	if err := Gotidy(); err != nil {
		return err
	}
	fmt.Println("===> Dev build complete ===|")
	return nil
}

// Css: scan templates and generate minified tailwind.min.css
func Css() error {
	fmt.Println("|==> Tailwind build started ===>")
	err := run(TAILWIND, "-i", INPUT_CSS, "-o", OUTPUT_CSS, "--content", TEMPLATE_DIR, "--minify")
	if err != nil {
		return err
	}
	fmt.Println("===> Tailwind build complete ===|")
	return nil
}

// Build: compile the viewer binary
func Build() error {
	fmt.Println("|==> Building flexlog-viewer ===>")
	err := run(GO, "build", "-o", BINARY, CMD)
	if err != nil {
		return err
	}
	fmt.Println("===> Build complete ===|")
	return nil
}

// Gotidy: go mod tidy
func Gotidy() error {
	fmt.Println("|==> Running go mod tidy ===>")
	err := run(GO, "mod", "tidy")
	if err != nil {
		return err
	}
	fmt.Println("===> Go mod tidy complete ===|")
	return nil
}

// Clean: remove generated CSS and binary
func Clean() error {
	fmt.Println("|==> Cleaning generated files ===>")
	if err := rm(OUTPUT_CSS); err != nil {
		return err
	}
	if err := rm(BINARY); err != nil {
		return err
	}
	if err := run(GO, "clean"); err != nil {
		return err
	}
	fmt.Println("===> Clean complete ===|")
	return nil
}
