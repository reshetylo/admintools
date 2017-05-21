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

// configuration file structure
type Configuration struct {
	BaseURL       string
	ServerAddress string
	StaticFolder  string
	StaticAddress string
	Templates     string
	Modules       string
	DefaultModule string
}

// render context structure
type Context struct {
	Title       string
	Data        string
	CurrentPage string
	Version     string
	BaseURL     string
}

const notFoundPage = "not_found"

var render_context = Context{Version: "1.0"}
var tpl *template.Template
var config = Configuration{}

// function for index / route. redirects to default module from configuration file
func Index(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	http.Redirect(w, req, config.BaseURL+"/page/"+config.DefaultModule, 302)
}

// function for /page/:name routes
func Page(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("page access", req.Host, req.RequestURI)

	pagename := ps.ByName("page")
	if _, err := os.Stat(strings.Replace(config.Templates, "*", pagename, 1)); err != nil {
		pagename = notFoundPage
		log.Println(notFoundPage, req.Host, req.RequestURI)
	}
	ctx := render_context
	ctx.CurrentPage = pagename
	if pagename == notFoundPage {
		w.WriteHeader(404)
	}
	render(w, pagename+".gohtml", ctx)
}

// function for /api/:name routes
func ApiModule(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("api access", req.Host, req.RequestURI)

	runfile := strings.Replace(config.Modules, "*", ps.ByName("name"), 1)
	req_url, puerr := url.ParseRequestURI(req.RequestURI)
	if puerr != nil {
		log.Print("Can not parse request URL: ", puerr, req)
	}
	param, pqerr := url.ParseQuery(req_url.RawQuery)
	if pqerr != nil {
		log.Print("Can not parse request Query: ", pqerr, req_url)
	}
	if _, err := os.Stat(runfile); err == nil {
		RenderFile(runfile, param, w)
	}
}

// render templates helper function
func render(w http.ResponseWriter, tpl_name string, ctx interface{}) {
	if ctx == nil {
		ctx = render_context
	}
	err := tpl.ExecuteTemplate(w, tpl_name, ctx)
	if err != nil {
		log.Println("ERROR Render: ", err)
	}
}

// application init. read configuration file, parse some data from it before the server started
func init() {
	// read command line arguments/flags
	config_file := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	// get configuration from defined json file
	cfg, _ := os.Open(*config_file)
	decoder := json.NewDecoder(cfg)
	err := decoder.Decode(&config)
	if err != nil {
		log.Fatal("Init: ", err)
	}

	// application init. parse templates and define Base URL value in global render context
	tpl = template.Must(template.ParseGlob(config.Templates))
	render_context.BaseURL = config.BaseURL
}

// main function. http routes setup and server starts and running here
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
