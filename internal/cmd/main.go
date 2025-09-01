package main

import (
	"context"
	"dice-sorensen-similarity-search/internal/middlewares"
	"log/slog"
)

func main() {
	token, _, err := middlewares.GenerateToken(context.Background(), []byte(middlewares.SigningKey), 0, "s79bb", []string{"admin"})
	if err != nil {
		slog.Error(err.Error())
	}

	slog.Info(token)
}
