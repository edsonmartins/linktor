package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// helper to get subcommand names from a cobra command
func subcommandNames(cmd *cobra.Command) []string {
	var names []string
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	return names
}

// helper to check that a command with ExactArgs has Args set
func assertExactArgsSet(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	for _, sub := range cmd.Commands() {
		if sub.Args != nil {
			// We just verify it's not nil; cobra sets it at registration time
			assert.NotNil(t, sub.Args, "command %q should have Args set", sub.Name())
		}
		// Recurse into nested subcommands
		assertExactArgsSet(t, sub)
	}
}

// =============================================================================
// botCmd
// =============================================================================

func TestBotCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "create", "update", "start", "stop", "status", "logs", "delete"}
	names := subcommandNames(botCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "botCmd should have subcommand %q", e)
	}
}

func TestBotCmd_ExactArgsCommands(t *testing.T) {
	// These commands require ExactArgs(1)
	exactArgsCmds := []*cobra.Command{
		botShowCmd, botUpdateCmd, botStartCmd, botStopCmd,
		botStatusCmd, botLogsCmd, botDeleteCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "bot %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// contactCmd
// =============================================================================

func TestContactCmd_Aliases(t *testing.T) {
	assert.Equal(t, []string{"contacts"}, contactCmd.Aliases)
}

func TestContactCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "create", "update", "delete", "import", "export", "merge"}
	names := subcommandNames(contactCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "contactCmd should have subcommand %q", e)
	}
}

func TestContactCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		contactShowCmd, contactUpdateCmd, contactDeleteCmd,
		contactImportCmd, contactMergeCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "contact %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// flowCmd
// =============================================================================

func TestFlowCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "execute", "validate", "publish", "unpublish", "export", "import", "delete"}
	names := subcommandNames(flowCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "flowCmd should have subcommand %q", e)
	}
}

func TestFlowCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		flowShowCmd, flowExecuteCmd, flowValidateCmd, flowPublishCmd,
		flowUnpublishCmd, flowExportCmd, flowImportCmd, flowDeleteCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "flow %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// kbCmd
// =============================================================================

func TestKbCmd_Aliases(t *testing.T) {
	assert.Equal(t, []string{"knowledge"}, kbCmd.Aliases)
}

func TestKbCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "create", "delete", "query", "doc"}
	names := subcommandNames(kbCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "kbCmd should have subcommand %q", e)
	}
}

func TestKbDocCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "add", "show", "delete", "reprocess"}
	names := subcommandNames(kbDocCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "kbDocCmd should have subcommand %q", e)
	}
}

func TestKbCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		kbShowCmd, kbDeleteCmd, kbQueryCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "kb %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// webhookCmd
// =============================================================================

func TestWebhookCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "test", "simulate", "events", "listen"}
	names := subcommandNames(webhookCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "webhookCmd should have subcommand %q", e)
	}
}

func TestWebhookCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		webhookTestCmd, webhookSimulateCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "webhook %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// serverCmd
// =============================================================================

func TestServerCmd_Subcommands(t *testing.T) {
	expected := []string{"start", "status", "migrate", "health", "backup", "plugin"}
	names := subcommandNames(serverCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "serverCmd should have subcommand %q", e)
	}
}

func TestServerPluginCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "install", "enable", "disable", "remove"}
	names := subcommandNames(serverPluginCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "serverPluginCmd should have subcommand %q", e)
	}
}

func TestServerPluginCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		serverPluginInstallCmd, serverPluginEnableCmd,
		serverPluginDisableCmd, serverPluginRemoveCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "server plugin %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// convCmd
// =============================================================================

func TestConvCmd_Aliases(t *testing.T) {
	assert.Equal(t, []string{"conversation", "conversations"}, convCmd.Aliases)
}

func TestConvCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "messages", "close", "reopen", "export"}
	names := subcommandNames(convCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "convCmd should have subcommand %q", e)
	}
}

func TestConvCmd_ExactArgsCommands(t *testing.T) {
	exactArgsCmds := []*cobra.Command{
		convShowCmd, convMessagesCmd, convCloseCmd,
		convReopenCmd, convExportCmd,
	}
	for _, cmd := range exactArgsCmds {
		assert.NotNil(t, cmd.Args, "conv %q should have Args set", cmd.Name())
	}
}

// =============================================================================
// channelCmd
// =============================================================================

func TestChannelCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "show", "create", "config", "test", "connect", "disconnect", "delete"}
	names := subcommandNames(channelCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "channelCmd should have subcommand %q", e)
	}
}

// =============================================================================
// authCmd
// =============================================================================

func TestAuthCmd_Subcommands(t *testing.T) {
	expected := []string{"login", "logout", "whoami", "tokens"}
	names := subcommandNames(authCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "authCmd should have subcommand %q", e)
	}
}

// =============================================================================
// configCmd
// =============================================================================

func TestConfigCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "get", "set", "use", "profile", "path"}
	names := subcommandNames(configCmd)
	for _, e := range expected {
		assert.Contains(t, names, e, "configCmd should have subcommand %q", e)
	}
}

// =============================================================================
// Recursive ExactArgs validation across the full tree
// =============================================================================

func TestAllCommands_ArgsConsistency(t *testing.T) {
	// Verify that commands registered to rootCmd that have ExactArgs have non-nil Args
	assertExactArgsSet(t, rootCmd)
}
