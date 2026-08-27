package plugin

import (
	"context"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func configure(req lifecycleRequest) error {
	if err := service.Configure(context.Background(), req.ConfigYAML); err != nil {
		return err
	}
	service.Current().SetAuthQuotaSource(hostAuthQuotaSource{})
	service.Current().SetModelDirectorySyncer(hostModelDirectorySyncer{})
	_ = service.Current().RefreshModelDirectory(context.Background())
	return nil
}

func pluginRegistration(schemaVersion uint32) registration {
	formats := []string{
		"openai", "chat-completions", "claude", "gemini", "openai-response", "responses", "codex",
		"openai-image", "openai-video",
	}
	return registration{
		SchemaVersion: schemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             service.PluginName,
			Version:          service.PluginVersion,
			Author:           "yuluo688",
			GitHubRepository: "https://github.com/yuluo688/credit-manager",
			ConfigFields: []pluginapi.ConfigField{
				{Name: "config_file", Type: pluginapi.ConfigFieldTypeString, Description: "External YAML path (or set CREDIT_MANAGER_CONFIG_FILE). Host may only set this."},
				{Name: "data_dir", Type: pluginapi.ConfigFieldTypeString, Description: "Plugin-managed data directory for SQLite and lock files."},
				{Name: "database_file", Type: pluginapi.ConfigFieldTypeString, Description: "SQLite filename under data_dir (default credit-manager.db)."},
				{Name: "busy_timeout", Type: pluginapi.ConfigFieldTypeString, Description: "SQLite busy timeout duration, e.g. 5s."},
				{Name: "keys.pepper_env", Type: pluginapi.ConfigFieldTypeString, Description: "Optional env override for id:pepper list; wins over pepper file when set."},
				{Name: "keys.pepper_file", Type: pluginapi.ConfigFieldTypeString, Description: "Pepper file under data_dir (default key-peppers). Auto-created on first run."},
				{Name: "keys.active_pepper_id", Type: pluginapi.ConfigFieldTypeString, Description: "Pepper id used when minting new keys."},
				{Name: "limits.max_token_estimate", Type: pluginapi.ConfigFieldTypeInteger, Description: "Strict maximum tokens reserved per request."},
				{Name: "limits.default_output_reserve", Type: pluginapi.ConfigFieldTypeInteger, Description: "Default output token reserve when body omits max_tokens."},
				{Name: "pricing.unknown_policy", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{"deny", "allow", "default"}, Description: "Behavior when no price rule matches."},
				{Name: "settlement.missing_usage", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{"settle_reserved", "release"}, Description: "Settlement when upstream returns no usage."},
				{Name: "settlement.host_usage_wait", Type: pluginapi.ConfigFieldTypeString, Description: "How long to wait for usage.handle before reserved_fallback, e.g. 1500ms. 0 disables."},
			},
		},
		Capabilities: registrationCapability{
			FrontendAuthProvider:          true,
			FrontendAuthProviderExclusive: true,
			ModelRouter:                   true,
			Executor:                      true,
			ExecutorModelScope:            string(pluginapi.ExecutorModelScopeBoth),
			ExecutorInputFormats:          formats,
			ExecutorOutputFormats:         formats,
			UsagePlugin:                   true,
			ManagementAPI:                 true,
			RequestInterceptor:            true,
			RequestLifecyclePlugin:        true,
			ResponseInterceptor:           true,
			Scheduler:                     true,
		},
	}
}
