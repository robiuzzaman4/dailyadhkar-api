package email

import (
	"embed"
	"fmt"
	"strings"
)

var templateFS embed.FS

const (
	TemplateDailyAdhkar = "templates/daily_adhkar.html"
)

type TemplateData map[string]string

func RenderTemplate(templateName string, data TemplateData) (string, error) {
	templateBytes, err := templateFS.ReadFile(templateName)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	content := string(templateBytes)

	// Replace all template variables with data
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}
