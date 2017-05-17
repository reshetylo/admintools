package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var tpl *template.Template

type Confiuration struct {
	BaseURL       string
	ServerAddress string
	StaticFolder  string
	StaticAddress string
	Templates     string
	Modules       string
	DefaultModule string
}

type Context struct {
	Title       string
	Data        string
	CurrentPage string
	Version     string
	BaseURL     string
}

var render_context = Context{Version:"1.0"}
var config = Confiuration{}

func Index(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	http.Redirect(w, req, config.BaseURL+"/page/"+config.DefaultModule, 302)
}

func Page(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("page access", req.Host, req.RequestURI)

	pagename := ps.ByName("page")
	if _, err := os.Stat(strings.Replace(config.Templates, "*", pagename, 1)); err != nil {
		pagename = "not_found"
		log.Println("not_found", req.Host, req.RequestURI)
	}
	ctx := render_context
	ctx.CurrentPage = pagename
	render(w, pagename+".gohtml", ctx)
}

func ApiModule(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	runfile := strings.Replace(config.Modules, "*", ps.ByName("name"), 1)

	req_url, _ := url.ParseRequestURI(req.RequestURI)
	param, _ := url.ParseQuery(req_url.RawQuery)
	log.Println("api_module", req.Host, req.RequestURI)
	log.Print(filecache)

	if _, err := os.Stat(runfile); err == nil {
		RenderFile(runfile, param, w)
	}
}

func init() {
	// read cli flags
	config_file := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	// get configuration
	cfg, _ := os.Open(*config_file)
	decoder := json.NewDecoder(cfg)
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal("Init: ", err)
	}

	// application init
	tpl = template.Must(template.ParseGlob(config.Templates))
	render_context.BaseURL = config.BaseURL
}

func render(w http.ResponseWriter, tpl_name string, ctx interface{}) {
	if ctx == nil {
		ctx = render_context
	}
	err := tpl.ExecuteTemplate(w, tpl_name, ctx)
	if err != nil {
		log.Println("ERROR Render: ", err)
	}
}

func main() {
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/page/:page", Page)
	router.GET("/api/:name", ApiModule)
	router.ServeFiles("/static/*filepath", http.Dir("./static"))
	log.Println("Server starting on ", config.ServerAddress)
	err := http.ListenAndServe(config.ServerAddress, router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
