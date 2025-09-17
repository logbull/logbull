package config

import (
	"os"
	"os/signal"
	"syscall"
)

var isShutDownSignalReceived = false

func StartListeningForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		isShutDownSignalReceived = true
	}()
}

func IsShouldShutdown() bool {
	return isShutDownSignalReceived
}
