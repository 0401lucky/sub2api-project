package main

import (
	"log"
	_ "time/tzdata"

	"welfare-backend/internal/app"
)

func main() {
	r, cfg, err := app.Build()
	if err != nil {
		log.Fatalf("build app failed: %v", err)
	}
	log.Printf("welfare-backend listening on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("server exit: %v", err)
	}
}
