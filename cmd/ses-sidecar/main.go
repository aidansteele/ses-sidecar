package main

import (
	"context"
	"fmt"
	"os"

	"github.com/QuentinBtd/ses-sidecar/internal/app/sessidecar"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout))
	slog.SetDefault(logger)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:1025"
	}

	if err := sessidecar.Run(ctx, logger, ses.NewFromConfig(cfg), addr); err != nil {
		logger.Error("starting server", err)
		os.Exit(1)
	}
}
