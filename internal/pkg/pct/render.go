package pct

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

func renderFile(fileName string, vars interface{}) string {
	tmpl, err := template.
		New(filepath.Base(fileName)).
		Funcs(
			template.FuncMap{
				"toClassName": func(itemName string) string {
					return strings.Title(strings.ToLower(itemName))
				},
			},
		).
		ParseFiles(fileName)

	if err != nil {
		log.Error().Msgf("Error parsing config: %v", err)
		return ""
	}

	return process(tmpl, vars)
}

func process(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		log.Error().Msgf("Error parsing config: %v", err)
		return ""
	}
	return tmplBytes.String()
}
