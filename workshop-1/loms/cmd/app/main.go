package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	lomshandler "route256/loms/internal/handler"
	orderrepo "route256/loms/internal/repository/order"
	stockrepo "route256/loms/internal/repository/stock"
	lomssvc "route256/loms/internal/service/loms"
	"route256/loms/internal/worker"
	"syscall"
	"time"
)

func main() {
	orders := orderrepo.New()
	stocks := stockrepo.New(map[uint32]uint64{
		1001: 100,
		1002: 50,
		1003: 10,
	})
	service := lomssvc.New(orders, stocks)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go worker.New(orders, service).Run(ctx)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           lomshandler.New(service),
		ReadHeaderTimeout: 2 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shut down LOMS server: %v", err)
		}
	}()

	log.Printf("LOMS listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
