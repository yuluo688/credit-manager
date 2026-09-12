// Command ops-release-stopped releases reservations interrupted by a host restart.
// It is an operations helper, not part of the plugin's request path.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

func run(database string) error {
	file, err := os.Stat(database)
	if err != nil || !file.Mode().IsRegular() {
		return fmt.Errorf("existing regular database file required")
	}
	const production = "/home/qiuhe/opt/CLIProxyAPI/data/credit-manager/credit-manager.db"
	if live, err := os.Stat(production); err == nil && os.SameFile(file, live) {
		out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", "cli-proxy-api").Output()
		if err != nil || strings.TrimSpace(string(out)) != "false" {
			return fmt.Errorf("production container must be stopped before releasing interrupted reservations")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s, err := store.Open(ctx, database, store.OpenOptions{BusyTimeout: time.Second})
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.DB().QueryContext(ctx, "SELECT id FROM reservations WHERE status='held' ORDER BY id")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.FinishExecution(ctx, id); err != nil {
			return err
		}
		if _, err := s.Release(ctx, id, "host_restart_cancelled"); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"released_count": len(ids), "released_ids": ids})
}

func main() {
	database := flag.String("database", "", "existing stopped-host database or an isolated snapshot")
	flag.Parse()
	if err := run(*database); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
