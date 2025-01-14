// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/packer/hcl2template/addrs"
	plugingetter "github.com/hashicorp/packer/packer/plugin-getter"
	"github.com/mitchellh/cli"
)

type PluginsRemoveCommand struct {
	Meta
}

func (c *PluginsRemoveCommand) Synopsis() string {
	return "Remove Packer plugins [matching a version]"
}

func (c *PluginsRemoveCommand) Help() string {
	helpText := `
Usage: packer plugins remove <plugin> [<version constraint>]

  This command will remove all Packer plugins matching the version constraint
  for the current OS and architecture.
  When the version is omitted all installed versions will be removed.

  Ex: packer plugins remove github.com/hashicorp/happycloud v1.2.3
`

	return strings.TrimSpace(helpText)
}

func (c *PluginsRemoveCommand) Run(args []string) int {
	ctx, cleanup := handleTermInterrupt(c.Ui)
	defer cleanup()

	return c.RunContext(ctx, args)
}

func (c *PluginsRemoveCommand) RunContext(buildCtx context.Context, args []string) int {
	if len(args) < 1 || len(args) > 2 {
		return cli.RunResultHelp
	}

	opts := plugingetter.ListInstallationsOptions{
		PluginDirectory: c.Meta.CoreConfig.Components.PluginConfig.PluginDirectory,
		BinaryInstallationOptions: plugingetter.BinaryInstallationOptions{
			OS:   runtime.GOOS,
			ARCH: runtime.GOARCH,
			Checksummers: []plugingetter.Checksummer{
				{Type: "sha256", Hash: sha256.New()},
			},
		},
	}

	if runtime.GOOS == "windows" && opts.Ext == "" {
		opts.BinaryInstallationOptions.Ext = ".exe"
	}

	plugin, diags := addrs.ParsePluginSourceString(args[0])
	if diags.HasErrors() {
		c.Ui.Error(diags.Error())
		return 1
	}

	// a plugin requirement that matches them all
	pluginRequirement := plugingetter.Requirement{
		Identifier: plugin,
	}

	if len(args) > 1 {
		constraints, err := version.NewConstraint(args[1])
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		pluginRequirement.VersionConstraints = constraints
	}

	installations, err := pluginRequirement.ListInstallations(opts)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	for _, installation := range installations {
		if err := os.Remove(installation.BinaryPath); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
		shasumFile := fmt.Sprintf("%s_SHA256SUM", installation.BinaryPath)
		if err := os.Remove(shasumFile); err != nil {
			c.Ui.Error(fmt.Sprintf("failed to remove %s: %s", shasumFile, err))
			c.Ui.Error("You may need to remove it manually")
		}
		c.Ui.Message(installation.BinaryPath)
	}

	if len(installations) == 0 {
		errMsg := fmt.Sprintf("No installed plugin found matching the plugin constraints %s", args[0])
		if len(args) == 2 {
			errMsg = fmt.Sprintf("%s %s", errMsg, args[1])
		}
		c.Ui.Error(errMsg)
		return 1
	}

	return 0
}
