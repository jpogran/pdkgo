package pct

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	TemplateConfigName     = "pct-config"
	TemplateConfigFileName = "pct-config.yml"
)

type PDKInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

type PuppetContentTemplate struct {
	Template PuppetContentTemplateInfo `mapstructure:"template"`
}

type PuppetContentTemplateInfo struct {
	Name    string `mapstructure:"name"`
	Type    string `mapstructure:"type"`
	Display string `mapstructure:"display"`
	Version string `mapstructure:"version"`
	URL     string `mapstructure:"url"`
}

type PuppetContentTemplateFileInfo struct {
	TemplatePath   string
	TargetFilePath string
	TargetDir      string
	TargetFile     string
	IsDirectory    bool
}

func List(templatePath string, templateName string) ([]PuppetContentTemplateInfo, error) {
	matches, _ := filepath.Glob(templatePath + "/**/" + TemplateConfigFileName)
	var tmpls []PuppetContentTemplateInfo
	for _, file := range matches {
		log.Debug().Msgf("Found: %+v", file)
		i := readTemplateConfig(file)
		tmpls = append(tmpls, i)
	}

	if templateName != "" {
		log.Debug().Msgf("Filtering for: %s", templateName)
		tmpls = filterFiles(tmpls, func(f PuppetContentTemplateInfo) bool { return f.Name == templateName })
	}

	return tmpls, nil
}

func Deploy(selectedTemplate string, localTemplateCache string, targetOutput string, targetName string, pdkInfo PDKInfo) []string {

	log.Trace().Msgf("PDKInfo: %+v", pdkInfo)

	file := filepath.Join(localTemplateCache, selectedTemplate, TemplateConfigFileName)
	log.Debug().Msgf("Template: %s", file)
	tmpl := readTemplateConfig(file)
	log.Trace().Msgf("Parsed: %+v", tmpl)

	// pdk new foo-foo
	if targetName == "" && targetOutput == "" {
		cwd, _ := os.Getwd()
		targetName = filepath.Base(cwd)
		targetOutput = cwd
	}

	// pdk new foo-foo -n wakka
	if targetName != "" && targetOutput == "" {
		cwd, _ := os.Getwd()
		targetOutput = filepath.Join(cwd, targetName)
	}

	// pdk new foo-foo -o /foo/bar/baz
	if targetName == "" && targetOutput != "" {
		targetName = filepath.Base(targetOutput)
	}

	// pdk new foo-foo
	if targetName == "" {
		cwd, _ := os.Getwd()
		targetName = filepath.Base(cwd)
	}

	// pdk new foo-foo
	// pdk new foo-foo -n wakka
	// pdk new foo-foo -n wakka -o c:/foo
	// pdk new foo-foo -n wakka -o c:/foo/wakka
	switch tmpl.Type {
	case "project":
		if targetOutput == "" {
			cwd, _ := os.Getwd()
			targetOutput = cwd
		} else if strings.HasSuffix(targetOutput, targetName) {
			// user has specified outputpath with the targetname in it
		} else {
			targetOutput = filepath.Join(targetOutput, targetName)
		}
	case "item":
		if targetOutput == "" {
			cwd, _ := os.Getwd()
			targetOutput = cwd
		} else if strings.HasSuffix(targetOutput, targetName) {
			// user has specified outputpath with the targetname in it
			targetOutput, _ = filepath.Split(targetOutput)
			log.Debug().Msgf("Changing target to :%s", targetOutput)
			targetOutput = filepath.Clean(targetOutput)
			log.Debug().Msgf("Changing target to :%s", targetOutput)
		}
		// } else {
		// 	// use what the user tells us
		// }

	}

	contentDir := filepath.Join(localTemplateCache, selectedTemplate, "content")
	log.Debug().Msgf("Target Name: %s", targetName)
	log.Debug().Msgf("Target Output: %s", targetOutput)

	var templateFiles []PuppetContentTemplateFileInfo
	err := filepath.WalkDir(contentDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		log.Trace().Msgf("Processing: %s", path)

		replacer := strings.NewReplacer(
			contentDir, targetOutput,
			"__REPLACE__", targetName,
			".tmpl", "",
		)
		targetFile := replacer.Replace(path)

		log.Debug().Msgf("Resolved '%s' to '%s'", path, targetFile)
		dir, file := filepath.Split(targetFile)
		i := PuppetContentTemplateFileInfo{
			TemplatePath:   path,
			TargetFilePath: targetFile,
			TargetDir:      dir,
			TargetFile:     file,
			IsDirectory:    info.IsDir(),
		}
		log.Trace().Msgf("Processed: %+v", i)

		templateFiles = append(templateFiles, i)
		return nil
	})
	if err != nil {
		log.Error().AnErr("content", err)
	}

	var deployed []string
	for _, templateFile := range templateFiles {
		log.Debug().Msgf("Deploying: %s", templateFile.TargetFilePath)
		if templateFile.IsDirectory {
			err := createTemplateDirectory(templateFile.TargetFilePath)
			if err == nil {
				deployed = append(deployed, templateFile.TargetFilePath)
			}
		} else {
			err := createTemplateFile(targetName, file, templateFile, tmpl, pdkInfo)
			if err != nil {
				log.Error().Msgf("%s", err)
				continue
			}
			deployed = append(deployed, templateFile.TargetFilePath)
		}
	}

	return deployed
}

func filterFiles(ss []PuppetContentTemplateInfo, test func(PuppetContentTemplateInfo) bool) (ret []PuppetContentTemplateInfo) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func createTemplateDirectory(targetDir string) error {
	log.Trace().Msgf("Creating: '%s'", targetDir)
	err := os.MkdirAll(targetDir, os.ModePerm)

	if err != nil {
		log.Error().Msgf("Error: %v", err)
		return err
	}

	return nil
}

func createTemplateFile(targetName string, configFile string, templateFile PuppetContentTemplateFileInfo, tmpl PuppetContentTemplateInfo, pdkInfo PDKInfo) error {
	log.Trace().Msgf("Creating: '%s'", templateFile.TargetFilePath)
	config := processConfiguration(
		targetName,
		configFile,
		templateFile.TemplatePath,
		tmpl,
		pdkInfo,
	)

	text := renderFile(templateFile.TemplatePath, config)
	if text == "" {
		return fmt.Errorf("Failed to create %s", templateFile.TargetFilePath)
	}

	log.Trace().Msgf("Writing: '%s' '%s'", templateFile.TargetFilePath, text)
	err := os.MkdirAll(templateFile.TargetDir, os.ModePerm)
	if err != nil {
		log.Error().Msgf("Error: %v", err)
		return err
	}

	file, err := os.Create(templateFile.TargetFilePath)
	if err != nil {
		log.Error().Msgf("Error: %v", err)
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, text)
	if err != nil {
		log.Error().Msgf("Error: %v", err)
		return err
	}

	err = file.Sync()
	if err != nil {
		log.Error().Msgf("Error: %v", err)
		return err
	}

	return nil
}
