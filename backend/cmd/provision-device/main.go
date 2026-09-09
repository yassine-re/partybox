package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"partybox/backend/internal/database"
	"partybox/backend/internal/deviceauth"
	"partybox/backend/internal/repositories"
	"partybox/backend/internal/services"
)

func main() {
	boxID := flag.String("box-id", "PB001", "identifiant de la PartyBox à provisionner")
	tokenFlag := flag.String("token", "", "token base64url de 32 octets (préférer DEVICE_TOKEN)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("connexion PostgreSQL impossible", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	token := strings.TrimSpace(os.Getenv("DEVICE_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(*tokenFlag)
	}
	generated := token == ""
	if generated {
		token, err = deviceauth.GenerateToken()
		if err != nil {
			slog.Error("génération du token impossible", "error", err)
			os.Exit(1)
		}
	}
	service := &services.Service{Repo: &repositories.Repository{Pool: pool}}
	if err = service.ProvisionDevice(ctx, *boxID, token); err != nil {
		slog.Error("provisionnement impossible", "box_id", *boxID, "error", err)
		os.Exit(1)
	}
	fmt.Printf("Device %s provisionné et activé.\n", *boxID)
	if generated {
		fmt.Printf("DEVICE_TOKEN=%s\nConservez ce token maintenant : il ne sera plus affiché.\n", token)
	} else {
		fmt.Println("Le token fourni a été remplacé sans être réaffiché.")
	}
}
