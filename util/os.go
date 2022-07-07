package util

import (
	"os"
	"os/signal"
	"syscall"
)

func WaitClose(signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGTSTP}
	}
	signal.Notify(c, signals...)
	for {
		<-c
		return
	}
}
