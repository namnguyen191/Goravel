package render

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/v2"
	"github.com/justinas/nosurf"
)

type Render struct {
	Renderer   string
	RootPath   string
	Secure     bool
	Port       string
	ServerName string
	JetViews   *jet.Set
	Session    *scs.SessionManager
}

type TemplateData struct {
	IsAuthenticated bool
	IntMap          map[string]int
	StringMap       map[string]string
	FloatMap        map[string]float32
	Data            map[string]interface{}
	CSRFToken       string
	Port            string
	ServerName      string
	Secure          bool
	Error           string
	Flash           string
}

func (ren *Render) defaultData(td *TemplateData, r *http.Request) *TemplateData {
	td.Secure = ren.Secure
	td.ServerName = ren.ServerName
	td.CSRFToken = nosurf.Token(r)
	td.Port = ren.Port

	if ren.Session.Exists(r.Context(), "userID") {
		td.IsAuthenticated = true
	}

	td.Error = ren.Session.PopString(r.Context(), "error")
	td.Flash = ren.Session.PopString(r.Context(), "flash")

	return td
}

func (ren *Render) Page(rw http.ResponseWriter, r *http.Request, view string, variables, data interface{}) error {
	switch strings.ToLower(ren.Renderer) {
	case "go":
		return ren.GoPage(rw, r, view, data)
	case "jet":
		return ren.JetPage(rw, r, view, variables, data)
	default:
		return errors.New("no rendering engine specify")
	}
}

// GoPage renders a standard Go template
func (ren *Render) GoPage(rw http.ResponseWriter, r *http.Request, view string, data interface{}) error {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/views/%s.page.tmpl", ren.RootPath, view))

	if err != nil {
		return err
	}

	td := &TemplateData{}
	if data != nil {
		td = data.(*TemplateData)
	}

	err = tmpl.Execute(rw, &td)

	if err != nil {
		return err
	}

	return nil
}

// JetPage render the template using Jet templating engine
func (ren *Render) JetPage(rw http.ResponseWriter, r *http.Request, templateName string, variables, data interface{}) error {
	var vars jet.VarMap

	if variables == nil {
		vars = make(jet.VarMap)
	} else {
		vars = variables.(jet.VarMap)
	}

	td := &TemplateData{}

	if data != nil {
		td = data.(*TemplateData)
	}

	td = ren.defaultData(td, r)

	t, err := ren.JetViews.GetTemplate(fmt.Sprintf("%s.jet", templateName))

	if err != nil {
		log.Println(err)
		return err
	}

	if err = t.Execute(rw, vars, td); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
