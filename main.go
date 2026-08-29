package main

import (
	"log/slog"
	"os"
	"recon/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		slog.Error("root 커맨드 에러", slog.Any("error", err))
		os.Exit(1)
	}
}
