package pct

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func processConfiguration(projectName string, configFile string, projectTemplate string, tmpl PuppetContentTemplateInfo, pdkInfo PDKInfo) map[string]interface{} {
	v := viper.New()

	log.Trace().Msgf("PDKInfo: %+v", pdkInfo)
	/*
		Inheritance (each level overwritten by next):
			convention based variables
				- pdk specific variables based on transformed user input
			machine variables
				- information that comes from the current machine
				- user name, hostname, etc
			template variables
				- information from the template itself
				- designed to be runnable defaults for everything inside template
			user overrides
				- ~/.pdk/pdk.yml
				- user customizations for their preferences
	*/

	// Convention based variables
	switch tmpl.Type {
	case "project":
		v.SetDefault("project_name", projectName)
	case "item":
		v.SetDefault("item_name", projectName)
	}
	user := getCurrentUser()
	v.SetDefault("user", user)
	v.SetDefault("puppet_module.author", user)

	// Machine based variables
	cwd, _ := os.Getwd()
	hostName, _ := os.Hostname()
	v.SetDefault("cwd", cwd)
	v.SetDefault("hostname", hostName)

	// PDK binary specific variables
	v.SetDefault("pdk.version", pdkInfo.Version)
	v.SetDefault("pdk.commit_hash", pdkInfo.Commit)
	v.SetDefault("pdk.build_date", pdkInfo.BuildDate)

	// Template specific variables
	log.Trace().Msgf("Adding %v", filepath.Dir(configFile))
	v.SetConfigName(TemplateConfigName)
	v.SetConfigType("yml")
	v.AddConfigPath(filepath.Dir(configFile))
	if err := v.ReadInConfig(); err == nil {
		log.Trace().Msgf("Merging config file: %v", v.ConfigFileUsed())
	} else {
		log.Error().Msgf("Error reading config: %v", err)
	}

	// User specified variable overrides
	home, _ := homedir.Dir()
	userConfigPath := filepath.Join(home, ".pdk")
	log.Trace().Msgf("Adding %v", userConfigPath)
	v.SetConfigName("pdk")
	v.SetConfigType("yml")
	v.AddConfigPath(userConfigPath)
	if err := v.MergeInConfig(); err == nil {
		log.Trace().Msgf("Merging config file: %v", v.ConfigFileUsed())
	} else {
		log.Error().Msgf("Error reading config: %v", err)
	}

	config := make(map[string]interface{})
	err := v.Unmarshal(&config)
	if err != nil {
		log.Error().Msgf("unable to decode into struct, %v", err)
		return nil
	}

	return config
}

func readTemplateConfig(configFile string) PuppetContentTemplateInfo {
	v := viper.New()
	userConfigFileBase := filepath.Base(configFile)
	v.AddConfigPath(filepath.Dir(configFile))
	v.SetConfigName(userConfigFileBase)
	v.SetConfigType("yml")
	if err := v.ReadInConfig(); err == nil {
		log.Trace().Msgf("Using template config file: %v", v.ConfigFileUsed())
	}
	var config PuppetContentTemplate
	err := v.Unmarshal(&config)
	if err != nil {
		log.Error().Msgf("unable to decode into struct, %v", err)
	}
	return config.Template
}

func getCurrentUser() string {
	user, _ := user.Current()
	if strings.Contains(user.Username, "\\") {
		v := strings.Split(user.Username, "\\")
		return v[1]
	}
	return user.Username
}
