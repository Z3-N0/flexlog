package main

import (
	"context"

	"github.com/Z3-N0/flexlog"
)

func main() {
	ctx := context.Background()
	log := flexlog.New()
	log.Info(ctx, "hello from flexlog", "env", "dev")
}
