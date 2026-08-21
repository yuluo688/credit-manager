package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type hostModelDirectorySyncer struct{}

func (hostModelDirectorySyncer) SyncExcludedModels(_ context.Context, disabled []string) error {
	raw, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil {
		return err
	}
	var listed struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return err
	}
	var errs []string
	for _, file := range listed.Files {
		if file.RuntimeOnly || strings.TrimSpace(file.AuthIndex) == "" {
			continue
		}
		if err := syncAuthExcludedModels(file, disabled); err != nil {
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = strings.TrimSpace(file.AuthIndex)
			}
			errs = append(errs, name+": "+err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New("sync excluded models: " + strings.Join(errs, "; "))
	}
	return nil
}

func syncAuthExcludedModels(file pluginapi.HostAuthFileEntry, disabled []string) error {
	load := func() (pluginapi.HostAuthGetResponse, error) {
		got, err := callHost(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{AuthIndex: file.AuthIndex})
		if err != nil {
			return pluginapi.HostAuthGetResponse{}, err
		}
		var auth pluginapi.HostAuthGetResponse
		if err := json.Unmarshal(got, &auth); err != nil {
			return pluginapi.HostAuthGetResponse{}, err
		}
		if len(auth.JSON) == 0 {
			return pluginapi.HostAuthGetResponse{}, errors.New("empty auth json")
		}
		return auth, nil
	}
	auth, err := load()
	if err != nil {
		return err
	}
	next, changed, err := service.MergeAuthExcludedModels(auth.JSON, disabled)
	if err != nil || !changed {
		return err
	}
	fresh, err := load()
	if err != nil {
		return err
	}
	if !bytes.Equal(auth.JSON, fresh.JSON) {
		next, changed, err = service.MergeAuthExcludedModels(fresh.JSON, disabled)
		if err != nil || !changed {
			return err
		}
		auth = fresh
	}
	name := strings.TrimSpace(file.Name)
	if name == "" {
		name = strings.TrimSpace(auth.Name)
	}
	if name == "" || !strings.HasSuffix(strings.ToLower(name), ".json") {
		return errors.New("auth file name is required")
	}
	_, err = callHost(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{Name: name, JSON: next})
	return err
}
