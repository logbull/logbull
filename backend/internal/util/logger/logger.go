package logger

import (
	"log/slog"
	"os"
	"sync"
	"time"
)

var (
	loggerInstance *slog.Logger
	once           sync.Once
)

// GetLogger returns a singleton slog.Logger that logs to the console
func GetLogger() *slog.Logger {
	once.Do(func() {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					a.Value = slog.StringValue(time.Now().Format("2006/01/02 15:04:05"))
				}

				if a.Key == slog.MessageKey {
					// Format the message to match the desired output format
					return slog.Attr{
						Key:   slog.MessageKey,
						Value: slog.StringValue(a.Value.String()),
					}
				}

				// Remove level and other attributes to get clean output
				if a.Key == slog.LevelKey {
					return slog.Attr{}
				}

				return a
			},
		})

		loggerInstance = slog.New(handler)

		loggerInstance.Info("Text structured logger initialized")
	})

	return loggerInstance
}
