// Stockyard Fence — API key vault for teams.
// Encrypt, store, rotate, and audit API keys. Self-hosted.
// Single binary, embedded SQLite, AES-256-GCM encryption at rest.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"
	"os"
	"strconv"

	"github.com/stockyard-dev/stockyard-fence/internal/server"
	"github.com/stockyard-dev/stockyard-fence/internal/store"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("fence %s\n", version)
			os.Exit(0)
		case "--health", "health":
			fmt.Println("ok")
			os.Exit(0)
		}
	}

	log.SetFlags(log.Ltime | log.Lshortfile)

	port := 8770
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}

	dataDir := "./data"
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}

	adminKey := os.Getenv("FENCE_ADMIN_KEY")
	if adminKey == "" {
		log.Fatalf("[fence] FENCE_ADMIN_KEY is required")
	}

	// Derive 32-byte AES key from FENCE_ENCRYPTION_KEY env var.
	// If not set, derive from admin key (not recommended for production).
	encKeyRaw := os.Getenv("FENCE_ENCRYPTION_KEY")
	var encKey []byte
	if encKeyRaw != "" {
		raw, err := hex.DecodeString(encKeyRaw)
		if err != nil || len(raw) != 32 {
			// Treat as passphrase — SHA-256 it
			h := sha256.Sum256([]byte(encKeyRaw))
			encKey = h[:]
		} else {
			encKey = raw
		}
	} else {
		log.Printf("[fence] FENCE_ENCRYPTION_KEY not set — deriving from admin key (set FENCE_ENCRYPTION_KEY for production)")
		h := sha256.Sum256([]byte(adminKey))
		encKey = h[:]
	}

		limits := server.DefaultLimits()
	db, err := store.Open(dataDir, encKey)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	log.Printf("")
	log.Printf("  Stockyard Fence %s", version)
	log.Printf("  API:      http://localhost:%d/api (requires FENCE_ADMIN_KEY)", port)
	log.Printf("  Resolve:  http://localhost:%d/api/resolve/{key_name} (requires fence token)", port)
	log.Printf("  Health:   http://localhost:%d/health", port)
	log.Printf("  Dashboard:  http://localhost:%d/ui", port)
	log.Printf("")

	// Expiration reminder goroutine — runs daily, logs keys expiring within 7 days
	if limits.ExpirationReminders {
		go func() {
			for {
				time.Sleep(24 * time.Hour)
				expiring := db.KeysExpiringWithin(7)
				for _, k := range expiring {
					log.Printf("[fence] expiry warning: key %q in vault %s expires %s",
						k.Name, k.VaultID, k.RotateAt)
				}
				if len(expiring) > 0 {
					log.Printf("[fence] %d key(s) expiring within 7 days", len(expiring))
				}
			}
		}()
	}

	srv := server.New(db, port, adminKey, limits)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
