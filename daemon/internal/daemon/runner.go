package daemon

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Run boots the supervisor + IPC server. Blocks until shutdown.
func Run(ctx context.Context, serve func(ctx context.Context, sup *Supervisor) error) error {
	if err := WritePIDFile(); err != nil {
		return err
	}
	defer RemovePIDFile()

	sup := NewSupervisor()
	defer sup.Shutdown()

	if err := sup.Resurrect(); err != nil {
		log.Printf("resurrect: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("signal received, shutting down")
		cancel()
	}()

	return serve(ctx, sup)
}
