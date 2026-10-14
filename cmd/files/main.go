package main

import (
	"context"
	"log"
	"net"

	files_v1 "github.com/voice-chat-team/contracts/gen/go/files/v1"
	"github.com/voice-chat-team/services.files/internal/config"
	"github.com/voice-chat-team/services.files/internal/database"
	"github.com/voice-chat-team/services.files/internal/repository"
	"github.com/voice-chat-team/services.files/internal/service"
	"github.com/voice-chat-team/services.files/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database pool error: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	st, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	repo := repository.NewFileRepository(pool)

	s := grpc.NewServer()
	files_v1.RegisterFileServiceServer(s, service.NewFileService(repo, st))
	reflection.Register(s)

	log.Printf("files service listening on :%s", cfg.GRPCPort)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
