package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	lomsclient "route256/cart/internal/client/loms"
	"route256/cart/internal/client/product"
	"route256/cart/internal/config"
	carthandler "route256/cart/internal/handler"
	cartrepo "route256/cart/internal/repository/cart"
	cartsvc "route256/cart/internal/service/cart"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()
	repository := cartrepo.New()
	products := product.New()
	loms := lomsclient.New(cfg.LOMSURL, nil)
	service := cartsvc.New(repository, products, loms)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           carthandler.New(service),
		ReadHeaderTimeout: 2 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shut down cart server: %v", err)
		}
	}()

	log.Printf("cart listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
