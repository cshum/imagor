package imagor

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/cshum/imagor/imagorpath"
)

//go:embed templates/*.html
var templatesFS embed.FS

var (
	landingTemplate *template.Template
	uploadTemplate  *template.Template
)

// TemplateData holds data for template rendering
type TemplateData struct {
	Version string
	Path    string
	Params  interface{}
}

// init initializes the templates
func init() {
	var err error
	
	landingTemplate, err = template.ParseFS(templatesFS, "templates/landing.html")
	if err != nil {
		panic("failed to parse landing template: " + err.Error())
	}
	
	uploadTemplate, err = template.ParseFS(templatesFS, "templates/upload.html")
	if err != nil {
		panic("failed to parse upload template: " + err.Error())
	}
}

// renderLandingPage renders the landing page template
func renderLandingPage(w http.ResponseWriter) {
	data := TemplateData{
		Version: Version,
	}
	
	var buf bytes.Buffer
	if err := landingTemplate.Execute(&buf, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(buf.Bytes())
}

// renderUploadForm renders the upload form template
func renderUploadForm(w http.ResponseWriter, path string) {
	// Parse the path to get imagor parameters for debug display
	params := imagorpath.Parse(path)
	
	// Marshal params as indented JSON for better readability
	paramsJSON, _ := json.MarshalIndent(params, "", "  ")
	
	data := TemplateData{
		Version: Version,
		Path:    path,
		Params:  string(paramsJSON),
	}
	
	var buf bytes.Buffer
	if err := uploadTemplate.Execute(&buf, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(buf.Bytes())
}
