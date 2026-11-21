package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

// syncFixture stubs the host auth RPCs for SyncExcludedModels tests.
type syncFixture struct {
	files    []pluginapi.HostAuthFileEntry
	auths    map[string][]byte   // auth index -> current JSON
	getSeq   map[string][][]byte // auth index -> queued get responses (consumed in order)
	models   []pluginapi.HostAuthModels
	saved    map[string][]byte // file name -> written JSON
	modelsOK bool
	getCalls map[string]int
}

func (f *syncFixture) hostCall(method string, raw []byte) ([]byte, int, error) {
	switch method {
	case pluginabi.MethodHostAuthList:
		enc, err := okEnvelope(map[string]any{"files": f.files})
		return enc, 0, err
	case pluginabi.MethodHostAuthModels:
		if !f.modelsOK {
			return nil, 0, errors.New("host.auth.models unsupported")
		}
		enc, err := okEnvelope(pluginapi.HostAuthModelsResponse{Credentials: f.models})
		return enc, 0, err
	case pluginabi.MethodHostAuthGet:
		var req pluginapi.HostAuthGetRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, 0, err
		}
		f.getCalls[req.AuthIndex]++
		if queue := f.getSeq[req.AuthIndex]; len(queue) > 0 {
			body := queue[0]
			f.getSeq[req.AuthIndex] = queue[1:]
			enc, err := okEnvelope(pluginapi.HostAuthGetResponse{AuthIndex: req.AuthIndex, JSON: body})
			return enc, 0, err
		}
		body, ok := f.auths[req.AuthIndex]
		if !ok {
			return nil, 0, errors.New("unknown auth " + req.AuthIndex)
		}
		enc, err := okEnvelope(pluginapi.HostAuthGetResponse{AuthIndex: req.AuthIndex, JSON: body})
		return enc, 0, err
	case pluginabi.MethodHostAuthSave:
		var req pluginapi.HostAuthSaveRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, 0, err
		}
		f.saved[req.Name] = append([]byte(nil), req.JSON...)
		enc, err := okEnvelope(pluginapi.HostAuthSaveResponse{Name: req.Name})
		return enc, 0, err
	}
	return nil, 0, errors.New("unexpected host call " + method)
}

func newSyncFixture() *syncFixture {
	return &syncFixture{
		auths:    map[string][]byte{},
		saved:    map[string][]byte{},
		getCalls: map[string]int{},
		modelsOK: true,
	}
}

func TestSyncExcludedModelsScopesPerProvider(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
		{AuthIndex: "claude-1", Name: "claude-a.json", Provider: "claude"},
		{AuthIndex: "kimi-1", Name: "kimi-a.json", Provider: "kimi"},
	}
	fx.auths["devin-1"] = []byte(`{"type":"devin"}`)
	fx.auths["claude-1"] = []byte(`{"type":"claude"}`)
	fx.auths["kimi-1"] = []byte(`{"type":"kimi"}`)
	fx.models = []pluginapi.HostAuthModels{
		{ID: "devin-1", AuthIndex: "devin-1", Provider: "devin", Models: []string{"swe-2", "devin/swe-1-6-slow"}},
		{ID: "claude-1", AuthIndex: "claude-1", Provider: "claude", Models: []string{"claude-opus-5"}},
		{ID: "kimi-1", AuthIndex: "kimi-1", Provider: "kimi", Models: []string{"kimi-k2"}},
	}
	HostCall = fx.hostCall

	disabled := []string{"devin/swe-1-6-slow", "swe-2"}
	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), disabled); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, ok := fx.saved["claude-a.json"]; ok {
		t.Fatalf("claude file written: %s", fx.saved["claude-a.json"])
	}
	if _, ok := fx.saved["kimi-a.json"]; ok {
		t.Fatalf("kimi file written: %s", fx.saved["kimi-a.json"])
	}
	devin, ok := fx.saved["devin-a.json"]
	if !ok {
		t.Fatalf("devin file not written; saved=%v", fx.saved)
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(devin, &meta); err != nil {
		t.Fatal(err)
	}
	var excluded []string
	if err := json.Unmarshal(meta["excluded_models"], &excluded); err != nil {
		t.Fatalf("excluded_models decode: %v", err)
	}
	if len(excluded) != 2 {
		t.Fatalf("excluded_models = %#v", excluded)
	}
}

func TestSyncExcludedModelsSharedBareIDOnlyOnServingProviders(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
		{AuthIndex: "claude-1", Name: "claude-a.json", Provider: "claude"},
		{AuthIndex: "kimi-1", Name: "kimi-a.json", Provider: "kimi"},
	}
	fx.auths["devin-1"] = []byte(`{"type":"devin"}`)
	fx.auths["claude-1"] = []byte(`{"type":"claude"}`)
	fx.auths["kimi-1"] = []byte(`{"type":"kimi"}`)
	fx.models = []pluginapi.HostAuthModels{
		{ID: "devin-1", AuthIndex: "devin-1", Provider: "devin", Models: []string{"claude-opus-5", "swe-2"}},
		{ID: "claude-1", AuthIndex: "claude-1", Provider: "claude", Models: []string{"claude-opus-5"}},
		{ID: "kimi-1", AuthIndex: "kimi-1", Provider: "kimi", Models: []string{"kimi-k2"}},
	}
	HostCall = fx.hostCall

	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), []string{"claude-opus-5"}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, ok := fx.saved["kimi-a.json"]; ok {
		t.Fatalf("kimi file written: %s", fx.saved["kimi-a.json"])
	}
	for _, name := range []string{"devin-a.json", "claude-a.json"} {
		if _, ok := fx.saved[name]; !ok {
			t.Fatalf("%s not written", name)
		}
	}
}

func TestSyncExcludedModelsHonorsOptOut(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
	}
	fx.auths["devin-1"] = []byte(`{"type":"devin","credit_manager_managed":false,"excluded_models":["mine"]}`)
	fx.models = []pluginapi.HostAuthModels{
		{ID: "devin-1", AuthIndex: "devin-1", Provider: "devin", Models: []string{"swe-2"}},
	}
	HostCall = fx.hostCall

	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), []string{"swe-2"}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(fx.saved) != 0 {
		t.Fatalf("opt-out file written: %v", fx.saved)
	}
}

func TestSyncExcludedModelsPreservesUserRemoval(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
	}
	fx.auths["devin-1"] = []byte(`{"type":"devin","excluded_models":["u1"],"credit_manager_removed_models":["swe-2"]}`)
	fx.models = []pluginapi.HostAuthModels{
		{ID: "devin-1", AuthIndex: "devin-1", Provider: "devin", Models: []string{"swe-2"}},
	}
	HostCall = fx.hostCall

	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), []string{"swe-2"}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(fx.saved) != 0 {
		t.Fatalf("user removal overwritten: %s", fx.saved["devin-a.json"])
	}
}

func TestSyncExcludedModelsFallsBackWithoutModelsAPI(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.modelsOK = false
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
		{AuthIndex: "claude-1", Name: "claude-a.json", Provider: "claude"},
	}
	fx.auths["devin-1"] = []byte(`{"type":"devin"}`)
	fx.auths["claude-1"] = []byte(`{"type":"claude"}`)
	HostCall = fx.hostCall

	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), []string{"devin/swe-1-6-slow", "swe-2"}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	// Without a catalog only the provider prefix applies: devin/x must not
	// reach the claude file; the bare ID still does (unscoped fallback).
	claudeWritten, ok := fx.saved["claude-a.json"]
	if !ok {
		t.Fatal("claude file not written in fallback mode")
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(claudeWritten, &meta); err != nil {
		t.Fatal(err)
	}
	var excluded []string
	if err := json.Unmarshal(meta["excluded_models"], &excluded); err != nil {
		t.Fatalf("excluded_models decode: %v", err)
	}
	if len(excluded) != 1 || excluded[0] != "swe-2" {
		t.Fatalf("claude excluded_models = %#v", excluded)
	}
	devinWritten, ok := fx.saved["devin-a.json"]
	if !ok {
		t.Fatal("devin file not written in fallback mode")
	}
	if err := json.Unmarshal(devinWritten, &meta); err != nil {
		t.Fatal(err)
	}
	excluded = nil
	if err := json.Unmarshal(meta["excluded_models"], &excluded); err != nil {
		t.Fatalf("excluded_models decode: %v", err)
	}
	if len(excluded) != 2 {
		t.Fatalf("devin excluded_models = %#v", excluded)
	}
}

func TestSyncExcludedModelsMergesConcurrentUserEdit(t *testing.T) {
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	fx := newSyncFixture()
	fx.files = []pluginapi.HostAuthFileEntry{
		{AuthIndex: "devin-1", Name: "devin-a.json", Provider: "devin"},
	}
	// First load: plugin-managed m1 still present; sync wants to add m2.
	// Before the save, the user removes m1 — the second load must pick that
	// edit up instead of overwriting it.
	fx.auths["devin-1"] = []byte(`{"type":"devin","excluded_models":["m1","u1"],"credit_manager_excluded_models":["m1"]}`)
	fx.getSeq = map[string][][]byte{
		"devin-1": {
			[]byte(`{"type":"devin","excluded_models":["m1","u1"],"credit_manager_excluded_models":["m1"]}`),
			[]byte(`{"type":"devin","excluded_models":["u1"],"credit_manager_excluded_models":["m1"]}`),
		},
	}
	fx.models = []pluginapi.HostAuthModels{
		{ID: "devin-1", AuthIndex: "devin-1", Provider: "devin", Models: []string{"m1", "m2", "u1"}},
	}
	HostCall = fx.hostCall

	if err := (hostModelDirectorySyncer{}).SyncExcludedModels(context.Background(), []string{"m1", "m2"}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	written, ok := fx.saved["devin-a.json"]
	if !ok {
		t.Fatal("devin file not written")
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(written, &meta); err != nil {
		t.Fatal(err)
	}
	var excluded []string
	if err := json.Unmarshal(meta["excluded_models"], &excluded); err != nil {
		t.Fatalf("excluded_models decode: %v", err)
	}
	if len(excluded) != 2 || excluded[0] != "m2" || excluded[1] != "u1" {
		t.Fatalf("concurrent removal lost: %#v", excluded)
	}
	var removed []string
	if err := json.Unmarshal(meta["credit_manager_removed_models"], &removed); err != nil {
		t.Fatalf("removed decode: %v", err)
	}
	if len(removed) != 1 || removed[0] != "m1" {
		t.Fatalf("removed = %#v", removed)
	}
	var managed []string
	if err := json.Unmarshal(meta["credit_manager_excluded_models"], &managed); err != nil {
		t.Fatalf("managed decode: %v", err)
	}
	if len(managed) != 1 || managed[0] != "m2" {
		t.Fatalf("managed = %#v", managed)
	}
}
