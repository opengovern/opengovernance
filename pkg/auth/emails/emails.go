package emails

import (
	"bytes"
	_ "embed"
	"html/template"
)

var (
	//go:embed inviteUserTemplate.html
	inviteUserTemplateString string
	//go:embed newUserTemplate.html
	newUserTemplateString string
	//go:embed newWorkspaceTemplate.html
	newWorkspaceTemplateString string
)

var (
	inviteUserTemplate   *template.Template
	newUserTemplate      *template.Template
	newWorkspaceTemplate *template.Template
)

func init() {
	inviteUserTemplate = template.Must(
		template.New("webpage").Parse(inviteUserTemplateString))

	newUserTemplate = template.Must(
		template.New("webpage").Parse(newUserTemplateString))

	newWorkspaceTemplate = template.Must(
		template.New("webpage").Parse(newWorkspaceTemplateString))
}

func GetInviteMailBody(link string, workspaceName string) (string, error) {
	var body bytes.Buffer

	err := inviteUserTemplate.Execute(&body, map[string]interface{}{
		"inviteLink":    link,
		"workspaceName": workspaceName,
	})
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

func GetNewUserMailBody(password string) (string, error) {
	var body bytes.Buffer

	err := newUserTemplate.Execute(&body, map[string]interface{}{
		"password": password,
	})
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

func GetNewWorkspaceMailBody(workspaceName string) (string, error) {
	var body bytes.Buffer

	err := newWorkspaceTemplate.Execute(&body, map[string]interface{}{
		"workspaceName": workspaceName,
	})
	if err != nil {
		return "", err
	}

	return body.String(), nil
}
