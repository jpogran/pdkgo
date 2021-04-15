package main

import (
	"github.com/puppetlabs/pdkgo/pkg/cmd/completion"
	"github.com/puppetlabs/pdkgo/pkg/cmd/new"
	"github.com/puppetlabs/pdkgo/pkg/cmd/root"
	appver "github.com/puppetlabs/pdkgo/pkg/cmd/version"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.1 "
	commit  = "79740ad713dbd4f1f7ffeca0c822837e2b9b82a0"
	date    = "unknown"
)

func main() {
	var rootCmd = root.CreateRootCommand()

	v := appver.Format(version, date, commit)
	rootCmd.Version = v
	rootCmd.SetVersionTemplate(v)

	var verCmd = appver.CreateVersionCommand(version, date, commit)
	rootCmd.AddCommand(verCmd)

	rootCmd.AddCommand(new.CreateNewCommand())
	rootCmd.AddCommand(completion.CreateCompletionCommand())

	cobra.OnInitialize(root.InitConfig)
	cobra.CheckErr(rootCmd.Execute())
}
