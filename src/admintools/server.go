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
	"github.com/reshetylo/cmdexec"
)

var tpl *template.Template

type Confiuration struct {
	BaseURL       string
	ServerAddress string
	StaticFolder  string
	StaticAddress string
	Templates     string
	Modules       string
}

type Context struct {
	Title   string
	Data    string
	BaseURL string
}

var context = Context{}
var config = Confiuration{}

func Index(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	render(w, "index.gohtml")
}

func Page(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("access", req.Host, req.RequestURI)

	pagename := ps.ByName("page")
	if _, err := os.Stat(strings.Replace(config.Templates, "*", pagename, 1)); err != nil {
		pagename = "not_found"
		log.Println("not_found", req.Host, req.RequestURI)
	}
	render(w, pagename+".gohtml")
}

func ApiModule(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	runfile := strings.Replace(config.Modules, "*", ps.ByName("name"), 1)

	req_url, _ := url.ParseRequestURI(req.RequestURI)
	param, _ := url.ParseQuery(req_url.RawQuery)
	log.Println("api_module", req.Host, req.RequestURI)

	if _, err := os.Stat(runfile); err == nil {
		cmdexec.RenderFile(runfile, param, w)
		//response := executor.ExecFile(runfile, param)
		//responsejson := json.NewEncoder(reponse)
		//		log.Println(response)
		//		w.Header().Set("Content-Type", "application/json")
		//		w.Write([]byte(response))
		//		for _, result := range response {
		//			w.Write([]byte(result))
		//		}
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
	context.BaseURL = config.BaseURL
}

func render(w http.ResponseWriter, tpl_name string) {
	err := tpl.ExecuteTemplate(w, tpl_name, context)
	if err != nil {
		log.Fatalln("Render: ", err)
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
