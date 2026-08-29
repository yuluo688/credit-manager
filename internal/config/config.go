package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// UnknownPricingPolicy controls requests whose model has no matching price rule.
type UnknownPricingPolicy string

const (
	UnknownPricingDeny    UnknownPricingPolicy = "deny"
	UnknownPricingAllow   UnknownPricingPolicy = "allow"
	UnknownPricingDefault UnknownPricingPolicy = "default"
)

// MissingUsagePolicy controls settlement when upstream executed but returned no usable usage.
type MissingUsagePolicy string

const (
	MissingUsageSettleReserved MissingUsagePolicy = "settle_reserved"
	MissingUsageRelease        MissingUsagePolicy = "release"
)

// PerMTokPricing stores integer micro-USD prices per one million tokens.
type PerMTokPricing struct {
	Input         int64 `yaml:"input" json:"input"`
	Output        int64 `yaml:"output" json:"output"`
	Reasoning     int64 `yaml:"reasoning" json:"reasoning"`
	Cached        int64 `yaml:"cached" json:"cached"`
	CacheRead     int64 `yaml:"cache_read" json:"cache_read"`
	CacheCreation int64 `yaml:"cache_creation" json:"cache_creation"`
}

// RequestLimits defines the strict request-estimate boundary enforced before upstream.
type RequestLimits struct {
	MaxTokenEstimate     int64 `yaml:"max_token_estimate" json:"max_token_estimate"`
	DefaultOutputReserve int64 `yaml:"default_output_reserve" json:"default_output_reserve"`
	RequireEstimate      bool  `yaml:"require_estimate" json:"require_estimate"`
}

// PricingConfig defines behavior when no stored pricing rule matches a model.
type PricingConfig struct {
	UnknownPolicy UnknownPricingPolicy `yaml:"unknown_policy" json:"unknown_policy"`
	Default       *PerMTokPricing      `yaml:"default,omitempty" json:"default,omitempty"`
}

// StreamConfig controls streaming accounting and stale-hold recovery.
type StreamConfig struct {
	MaxBufferBytes          int           `yaml:"max_buffer_bytes" json:"max_buffer_bytes"`
	StaleReservationTimeout time.Duration `yaml:"stale_reservation_timeout" json:"stale_reservation_timeout"`
}

// SettlementConfig controls post-upstream accounting when usage is incomplete.
type SettlementConfig struct {
	MissingUsage MissingUsagePolicy `yaml:"missing_usage" json:"missing_usage"`
	// HostUsageWait is how long SettleFromUsage waits for usage.handle before
	// reserved_fallback. Zero disables the wait.
	HostUsageWait time.Duration `yaml:"host_usage_wait" json:"host_usage_wait"`
}

// KeyConfig holds external pepper material for plugin key verification.
// Peppers never enter SQLite, logs, or management API responses.
type KeyConfig struct {
	// PepperEnv is an optional environment variable that stores "id:hex,id:hex" peppers.
	// When set and non-empty, it takes precedence over PepperFile.
	PepperEnv string `yaml:"pepper_env" json:"pepper_env"`
	// PepperFile is a path to a pepper material file (same format as the env value).
	// Relative paths are resolved under DataDir. Empty defaults to "key-peppers".
	// On first use, if neither env nor file provides material, a random pepper is
	// generated and written here for reuse across restarts.
	PepperFile string `yaml:"pepper_file,omitempty" json:"pepper_file,omitempty"`
	// ActivePepperID selects the pepper used when minting new keys. Empty means first entry.
	ActivePepperID string `yaml:"active_pepper_id,omitempty" json:"active_pepper_id,omitempty"`
}

// ConfigFileEnv is an optional path to a plugin YAML file. Used when host
// config is empty or does not set config_file.
const ConfigFileEnv = "CREDIT_MANAGER_CONFIG_FILE"

// Config is the plugin YAML configuration loaded by the host and/or config_file.
type Config struct {
	// ConfigFile optionally points to an external YAML file. Host config can be
	// just `config_file: /path/to/credit-manager.yaml` to avoid inlining.
	// Host fields overlay the file. Nested config_file inside that file is ignored.
	ConfigFile string `yaml:"config_file,omitempty" json:"config_file,omitempty"`
	// DataDir is the plugin-managed directory for SQLite and lock files.
	DataDir string `yaml:"data_dir" json:"data_dir"`
	// DatabaseFile is the SQLite filename under DataDir. Defaults to credit-manager.db.
	DatabaseFile string           `yaml:"database_file,omitempty" json:"database_file,omitempty"`
	BusyTimeout  time.Duration    `yaml:"busy_timeout" json:"busy_timeout"`
	Keys         KeyConfig        `yaml:"keys" json:"keys"`
	Limits       RequestLimits    `yaml:"limits" json:"limits"`
	Pricing      PricingConfig    `yaml:"pricing" json:"pricing"`
	Settlement   SettlementConfig `yaml:"settlement" json:"settlement"`
	Stream       StreamConfig     `yaml:"stream" json:"stream"`
}

func Default() Config {
	return Config{
		DataDir:      "./data/credit-manager",
		DatabaseFile: "credit-manager.db",
		BusyTimeout:  5 * time.Second,
		Keys: KeyConfig{
			PepperEnv:  "CREDIT_MANAGER_KEY_PEPPERS",
			PepperFile: "key-peppers",
		},
		Limits: RequestLimits{
			MaxTokenEstimate:     1_000_000,
			DefaultOutputReserve: 4_096,
			RequireEstimate:      false,
		},
		Pricing: PricingConfig{
			// Zero-config default: unknown models are allowed at $0 until rules are customized.
			UnknownPolicy: UnknownPricingAllow,
		},
		Settlement: SettlementConfig{
			MissingUsage:  MissingUsageSettleReserved,
			HostUsageWait: 4 * time.Second,
		},
		Stream: StreamConfig{
			MaxBufferBytes:          4 << 20,
			StaleReservationTimeout: 2 * time.Hour,
		},
	}
}

func ParseYAML(raw []byte) (Config, error) {
	cfg, err := parseYAML(raw, true)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseYAML(raw []byte, allowExternalFile bool) (Config, error) {
	cfg := Default()
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		if allowExternalFile {
			if path := strings.TrimSpace(os.Getenv(ConfigFileEnv)); path != "" {
				return loadConfigFile(path)
			}
		}
		return cfg, nil
	}

	var meta struct {
		ConfigFile string `yaml:"config_file"`
	}
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return Config{}, fmt.Errorf("parse plugin config: %w", err)
	}

	filePath := strings.TrimSpace(meta.ConfigFile)
	if filePath == "" && allowExternalFile {
		filePath = strings.TrimSpace(os.Getenv(ConfigFileEnv))
	}
	if filePath != "" {
		if !allowExternalFile {
			return Config{}, errors.New("nested config_file is not supported")
		}
		fileCfg, err := loadConfigFile(filePath)
		if err != nil {
			return Config{}, err
		}
		cfg = fileCfg
	}

	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse plugin config: %w", err)
	}
	// Keep resolved path for diagnostics; host overlay may have rewritten it.
	if filePath != "" {
		cfg.ConfigFile = filePath
	}
	return cfg, nil
}

func loadConfigFile(path string) (Config, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return Config{}, errors.New("config_file path is empty")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config_file %q: %w", path, err)
	}
	// Disallow recursive config_file chains.
	cfg, err := parseYAML(raw, false)
	if err != nil {
		return Config{}, fmt.Errorf("parse config_file %q: %w", path, err)
	}
	cfg.ConfigFile = path
	return cfg, nil
}

func (c Config) DatabasePath() string {
	file := strings.TrimSpace(c.DatabaseFile)
	if file == "" {
		file = "credit-manager.db"
	}
	return filepath.Join(filepath.Clean(strings.TrimSpace(c.DataDir)), file)
}

// PepperFilePath resolves the on-disk pepper material path.
func (c Config) PepperFilePath() string {
	name := strings.TrimSpace(c.Keys.PepperFile)
	if name == "" {
		name = "key-peppers"
	}
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	return filepath.Join(filepath.Clean(strings.TrimSpace(c.DataDir)), name)
}

func (c Config) Validate() error {
	var errs []error
	if strings.TrimSpace(c.DataDir) == "" {
		errs = append(errs, errors.New("data_dir is required"))
	}
	if c.BusyTimeout <= 0 || c.BusyTimeout > 5*time.Minute {
		errs = append(errs, errors.New("busy_timeout must be greater than zero and at most 5 minutes"))
	}
	if c.Limits.MaxTokenEstimate <= 0 {
		errs = append(errs, errors.New("limits.max_token_estimate must be greater than zero"))
	}
	if c.Limits.DefaultOutputReserve <= 0 {
		errs = append(errs, errors.New("limits.default_output_reserve must be greater than zero"))
	}
	if c.Limits.DefaultOutputReserve > c.Limits.MaxTokenEstimate {
		errs = append(errs, errors.New("limits.default_output_reserve must not exceed max_token_estimate"))
	}
	if c.Stream.MaxBufferBytes <= 0 {
		errs = append(errs, errors.New("stream.max_buffer_bytes must be greater than zero"))
	}
	if c.Stream.StaleReservationTimeout < time.Minute || c.Stream.StaleReservationTimeout > 24*time.Hour {
		errs = append(errs, errors.New("stream.stale_reservation_timeout must be between 1 minute and 24 hours"))
	}

	switch c.Pricing.UnknownPolicy {
	case UnknownPricingDeny, UnknownPricingAllow:
		if c.Pricing.Default != nil {
			errs = append(errs, fmt.Errorf("pricing.default must be nil for unknown_policy %q", c.Pricing.UnknownPolicy))
		}
	case UnknownPricingDefault:
		if c.Pricing.Default == nil {
			errs = append(errs, errors.New("pricing.default is required for unknown_policy default"))
		}
	default:
		errs = append(errs, fmt.Errorf("pricing.unknown_policy %q is invalid", c.Pricing.UnknownPolicy))
	}
	if c.Pricing.Default != nil {
		if err := c.Pricing.Default.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("pricing.default: %w", err))
		}
	}

	switch c.Settlement.MissingUsage {
	case MissingUsageSettleReserved, MissingUsageRelease:
	default:
		errs = append(errs, fmt.Errorf("settlement.missing_usage %q is invalid", c.Settlement.MissingUsage))
	}

	if _, err := c.LoadPeppers(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// PepperSet is external verification material. Never serialize into the database.
type PepperSet struct {
	ActiveID string
	// Values maps pepper id -> raw pepper bytes.
	Values map[string][]byte
}

// LoadPeppers resolves pepper material in this order:
//  1. Environment variable keys.pepper_env (if set and non-empty)
//  2. Existing keys.pepper_file under data_dir (or absolute path)
//  3. First-run auto-generate: write a random pepper into pepper_file and reuse it
func (c Config) LoadPeppers() (PepperSet, error) {
	envName := strings.TrimSpace(c.Keys.PepperEnv)
	if envName != "" {
		if raw := strings.TrimSpace(os.Getenv(envName)); raw != "" {
			return parsePepperList(raw, "environment variable "+envName, c.Keys.ActivePepperID)
		}
	}

	path := c.PepperFilePath()
	if path == "" || path == "." {
		return PepperSet{}, errors.New("pepper file path is empty")
	}

	raw, err := os.ReadFile(path)
	if err == nil {
		if strings.TrimSpace(string(raw)) != "" {
			return parsePepperList(string(raw), "pepper file "+path, c.Keys.ActivePepperID)
		}
		// Empty file: treat as missing and regenerate.
	} else if !errors.Is(err, os.ErrNotExist) {
		return PepperSet{}, fmt.Errorf("read pepper file %q: %w", path, err)
	}

	// First run (or empty file): create data dir + pepper file atomically when possible.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return PepperSet{}, fmt.Errorf("create pepper file directory: %w", err)
	}
	generated, err := generatePepperMaterial(c.Keys.ActivePepperID)
	if err != nil {
		return PepperSet{}, err
	}
	if err := writePepperFileExclusive(path, generated); err != nil {
		// Empty existing file or concurrent create: rewrite/read carefully.
		if errors.Is(err, os.ErrExist) {
			if raw, readErr := os.ReadFile(path); readErr == nil && len(strings.TrimSpace(string(raw))) > 0 {
				return parsePepperList(string(raw), "pepper file "+path, c.Keys.ActivePepperID)
			}
		}
		// Replace empty file.
		if writeErr := os.WriteFile(path, []byte(generated+"\n"), 0o600); writeErr != nil {
			return PepperSet{}, fmt.Errorf("write pepper file %q: %w", path, writeErr)
		}
	}
	return parsePepperList(generated, "pepper file "+path, c.Keys.ActivePepperID)
}

func generatePepperMaterial(activeID string) (string, error) {
	id := strings.TrimSpace(activeID)
	if id == "" {
		id = "active"
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate pepper: %w", err)
	}
	return id + ":" + hex.EncodeToString(buf), nil
}

func writePepperFileExclusive(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(content + "\n"); err != nil {
		return err
	}
	return f.Sync()
}

func parsePepperList(raw, source, activePepperID string) (PepperSet, error) {
	values := make(map[string][]byte)
	var order []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Allow multi-line files: also split on newlines inside entries.
		for _, line := range strings.Split(part, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			id, secret, ok := strings.Cut(line, ":")
			id = strings.TrimSpace(id)
			secret = strings.TrimSpace(secret)
			if !ok || id == "" || secret == "" {
				return PepperSet{}, fmt.Errorf("%s entry %q must be id:hex_or_text", source, line)
			}
			if _, exists := values[id]; exists {
				return PepperSet{}, fmt.Errorf("%s has duplicate pepper id %q", source, id)
			}
			var material []byte
			if decoded, err := decodeMaybeHex(secret); err == nil && len(decoded) >= 16 {
				material = decoded
			} else if len(secret) >= 16 {
				material = []byte(secret)
			} else {
				return PepperSet{}, fmt.Errorf("pepper %q must be at least 16 bytes", id)
			}
			values[id] = material
			order = append(order, id)
		}
	}
	if len(values) == 0 {
		return PepperSet{}, fmt.Errorf("%s contains no peppers", source)
	}
	active := strings.TrimSpace(activePepperID)
	if active == "" {
		active = order[0]
	}
	if _, ok := values[active]; !ok {
		return PepperSet{}, fmt.Errorf("active pepper id %q is not present in %s", active, source)
	}
	return PepperSet{ActiveID: active, Values: values}, nil
}

func decodeMaybeHex(value string) ([]byte, error) {
	// local import avoidance: use encoding/hex via small helper file would bloat;
	// keep decode in keys package if needed. Here only attempt simple even-length hex.
	if len(value)%2 != 0 {
		return nil, errors.New("not hex")
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return nil, errors.New("not hex")
		}
	}
	out := make([]byte, len(value)/2)
	for i := 0; i < len(out); i++ {
		out[i] = (hexNibble(value[2*i]) << 4) | hexNibble(value[2*i+1])
	}
	return out, nil
}

func hexNibble(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	default:
		return 0
	}
}

func (p PerMTokPricing) Validate() error {
	values := []struct {
		name  string
		value int64
	}{
		{"input", p.Input},
		{"output", p.Output},
		{"reasoning", p.Reasoning},
		{"cached", p.Cached},
		{"cache_read", p.CacheRead},
		{"cache_creation", p.CacheCreation},
	}
	var errs []error
	for _, item := range values {
		if item.value < 0 {
			errs = append(errs, fmt.Errorf("%s price must not be negative", item.name))
		}
	}
	return errors.Join(errs...)
}

// ValidateTokenEstimate rejects missing estimates when strict estimation is
// enabled and always rejects estimates above the configured request maximum.
func (c Config) ValidateTokenEstimate(estimate int64) error {
	if estimate < 0 {
		return errors.New("request token estimate must not be negative")
	}
	if c.Limits.RequireEstimate && estimate == 0 {
		return errors.New("request token estimate is required")
	}
	if estimate > c.Limits.MaxTokenEstimate {
		return fmt.Errorf("request token estimate %d exceeds maximum %d", estimate, c.Limits.MaxTokenEstimate)
	}
	return nil
}
