//go:build ignore

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	liveAuthID   = "xai-kingdiaodu@gmail.com.json"
	liveProvider = "xai"
	liveKeyFile  = `D:\UserData\Administrator\.temp\kilo\cm_live_key.txt`
)

func main() {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = `D:\CLIProxyAPI\data\credit-manager`
	cfg.Keys.PepperFile = "key-peppers"
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "setup-http":
		if err := setupHTTP(ctx, svc); err != nil {
			fmt.Fprintf(os.Stderr, "setup-http: %v\n", err)
			os.Exit(1)
		}
		return
	case "restore":
		if err := restoreLimit(ctx, svc); err != nil {
			fmt.Fprintf(os.Stderr, "restore: %v\n", err)
			os.Exit(1)
		}
		_ = os.Remove(liveKeyFile)
		fmt.Println("restored unlimited and removed temp key file")
		return
	}

	authID := liveAuthID
	provider := liveProvider
	item, err := svc.SetAuthConcurrencyLimit(ctx, provider, authID, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set limit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("set max_concurrent_requests=%d active=%d auth_id=%s provider=%s\n", item.MaxConcurrentRequests, item.ActiveRequests, item.AuthID, item.Provider)

	got, err := svc.Store().GetAuthConcurrencyLimit(ctx, provider, authID)
	if err != nil || got != 1 {
		fmt.Fprintf(os.Stderr, "get limit = %d %v\n", got, err)
		os.Exit(1)
	}

	candidates := []service.AuthPickCandidate{{ID: authID, Provider: provider}}
	id, handled, err := svc.PickAuth(ctx, candidates)
	if err != nil || !handled || id != authID {
		fmt.Fprintf(os.Stderr, "first pick = (%q, %t, %v)\n", id, handled, err)
		os.Exit(1)
	}
	fmt.Println("first pick bound the only xai credential")

	svc.TrackAuthCapture("live-res-held", "grok-3-mini")
	if err := svc.AdmitAuth(ctx, "live-res-held", store.AuthIdentity{AuthID: authID, Provider: provider}); err != nil {
		fmt.Fprintf(os.Stderr, "admit held: %v\n", err)
		os.Exit(1)
	}
	_, handled, err = svc.PickAuth(ctx, candidates)
	if !handled || !errors.Is(err, store.ErrConcurrentLimit) {
		fmt.Fprintf(os.Stderr, "busy pick = handled=%t err=%v\n", handled, err)
		os.Exit(1)
	}
	fmt.Println("second pick rejected: maximum concurrent requests reached")

	svc.CancelAuthCapture("live-res-held")
	id, handled, err = svc.PickAuth(ctx, candidates)
	if err != nil || !handled || id != authID {
		fmt.Fprintf(os.Stderr, "pick after release = (%q, %t, %v)\n", id, handled, err)
		os.Exit(1)
	}
	fmt.Println("pick after release succeeded")

	if _, err := svc.SetAuthConcurrencyLimit(ctx, provider, authID, 0); err != nil {
		fmt.Fprintf(os.Stderr, "restore unlimited: %v\n", err)
		os.Exit(1)
	}
	id, handled, err = svc.PickAuth(ctx, candidates)
	if err != nil || handled || id != "" {
		fmt.Fprintf(os.Stderr, "unlimited pick should delegate, got (%q, %t, %v)\n", id, handled, err)
		os.Exit(1)
	}
	fmt.Println("unlimited delegates to host scheduler")
	fmt.Printf("ok %s\n", time.Now().Format(time.RFC3339))
}

func setupHTTP(ctx context.Context, svc *service.Service) error {
	keysList, err := svc.Store().ListPluginKeys(ctx, 50)
	if err != nil {
		return err
	}
	var key store.PluginKey
	for _, item := range keysList {
		if item.Enabled && strings.EqualFold(item.Label, "fallingcliff") {
			key = item
			break
		}
	}
	if key.ID == "" {
		return errors.New("enabled fallingcliff key not found")
	}
	plaintext, err := keys.DecryptPlaintext(key.EncryptedKeyMaterial, svc.Peppers())
	if err != nil {
		return err
	}
	if err := os.WriteFile(liveKeyFile, []byte(plaintext), 0o600); err != nil {
		return err
	}
	item, err := svc.SetAuthConcurrencyLimit(ctx, liveProvider, liveAuthID, 1)
	if err != nil {
		return err
	}
	fmt.Printf("http setup key_label=%s models=%d max_concurrent=%d auth_id=%s\n", key.Label, len(key.AllowedModels), item.MaxConcurrentRequests, item.AuthID)
	return nil
}

func restoreLimit(ctx context.Context, svc *service.Service) error {
	_, err := svc.SetAuthConcurrencyLimit(ctx, liveProvider, liveAuthID, 0)
	return err
}
