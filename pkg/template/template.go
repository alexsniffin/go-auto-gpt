package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// this is facade
// todo refactor
func Parse(text string, fields any) (string, error) {
	tmpl := template.Must(template.New("").Parse(text)) // todo cache this
	var result bytes.Buffer
	err := tmpl.Execute(&result, fields)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return result.String(), nil
}
