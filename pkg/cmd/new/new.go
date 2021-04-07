package new

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/pdkgo/internal/pkg/pct"

	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	localTemplateCache string
	jsonOutput         bool

	selectedTemplate string
	listTemplates    bool

	targetName   string
	targetOutput string
)

func CreateNewCommand() *cobra.Command {
	tmp := &cobra.Command{
		Use:   "new <template> [flags] [args]",
		Short: "Creates a Puppet project or other artifact based on a template",
		Long:  `Creates a Puppet project or other artifact based on a template`,
		Args: func(cmd *cobra.Command, args []string) error {
			return validateRootArguments(args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			localTemplateCache = viper.GetString("templatepath")
			return completeName(localTemplateCache, toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.Trace().Msg("PreRunE")
			localTemplateCache = viper.GetString("templatepath")
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Trace().Msg("Run")
			return executeCommand()
		},
	}
	tmp.Flags().StringVar(&localTemplateCache, "templatepath", "", "Location of installed templates")
	viper.BindPFlag("templatepath", tmp.Flags().Lookup("templatepath")) //nolint:errcheck
	home, _ := homedir.Dir()
	viper.SetDefault("templatepath", filepath.Join(home, ".pdk", "pct"))

	tmp.Flags().BoolVarP(&listTemplates, "list", "l", false, "list templates")
	tmp.RegisterFlagCompletionFunc("list", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) { //nolint:errcheck
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeName(localTemplateCache, toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})

	tmp.Flags().StringVarP(&targetName, "name", "n", "", "the name for the created output. (default is the name of the current directory)")
	tmp.Flags().StringVarP(&targetOutput, "output", "o", "", "location to place the generated output. (default is the current directory)")
	tmp.Flags().BoolVar(&jsonOutput, "json", false, "json output")
	return tmp
}

func executeCommand() error {
	log.Trace().Msgf("Template path: %v", localTemplateCache)
	log.Trace().Msgf("Selected template: %v", selectedTemplate)

	if listTemplates {
		tmpls, err := pct.List(localTemplateCache, selectedTemplate)
		if err != nil {
			return err
		}

		if jsonOutput {
			j := jsoniter.ConfigFastest
			prettyJSON, err := j.MarshalIndent(&tmpls, "", "  ")
			if err != nil {
				log.Error().Msgf("Error converting to json: %v", err)
			}
			fmt.Printf("%s\n", string(prettyJSON))
		} else {
			fmt.Println("")
			if len(tmpls) == 1 {
				fmt.Printf("DisplayName:     %v\n", tmpls[0].Display)
				fmt.Printf("Name:            %v\n", tmpls[0].Name)
				fmt.Printf("TemplateType:    %v\n", tmpls[0].Type)
				fmt.Printf("TemplateURL:     %v\n", tmpls[0].URL)
				fmt.Printf("TemplateVersion: %v\n", tmpls[0].Version)
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"DisplayName", "Name", "Type"})
				table.SetBorder(false)
				for _, v := range tmpls {
					table.Append([]string{v.Display, v.Name, v.Type})
				}
				table.Render()
			}
		}

		return nil
	}

	deployed := pct.Deploy(
		selectedTemplate,
		localTemplateCache,
		targetOutput,
		targetName,
	)

	if jsonOutput {
		j := jsoniter.ConfigFastest
		prettyJSON, _ := j.MarshalIndent(deployed, "", "  ")
		fmt.Printf("%s\n", prettyJSON)
	} else {
		for _, d := range deployed {
			log.Info().Msgf("Deployed: %v", d)
		}
	}

	return nil
}

func validateRootArguments(args []string) error {
	if len(args) == 0 && !listTemplates {
		listTemplates = true
	}

	if targetName == "" && len(args) == 2 {
		targetName = args[1]
	}

	if len(args) >= 1 {
		selectedTemplate = args[0]
	}

	return nil
}

func completeName(cache string, match string) []string {
	tmpls, _ := pct.List(cache, "")
	var names []string
	for _, tmpl := range tmpls {
		if strings.HasPrefix(tmpl.Name, match) {
			m := tmpl.Name + "\t" + tmpl.Display
			names = append(names, m)
		}
	}
	return names
}
