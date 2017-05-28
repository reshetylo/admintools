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

	"path/filepath"

	"github.com/julienschmidt/httprouter"
)

// configuration file structure
type Configuration struct {
	BaseURL         string
	ServerAddress   string
	StaticFolder    string
	StaticAddress   string
	Templates       string
	Modules         string
	DefaultTemplate string
	CookieSecret    string
	AuthType        string
	AuthName        string
	AuthParameters  struct {
		Endpoint    string
		RedirectURL string
	}
}

// render context structure
type Context struct {
	Title       string
	Data        string
	CurrentPage string
	Version     string
	BaseURL     string
	User        string
	AuthName    string
	Modules     []Module
}

// module.json format
type Module struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Author      string `json:"Author"`
	Version     string `json:"Version"`
	Template    string `json:"Template"`
	Enabled     bool   `json:"Enabled"`
	AccessLevel string `json:"AccessLevel"`
}

const notFoundPage = "not_found"

var render_context = Context{Version: "1.0"}
var tpl *template.Template
var config = Configuration{}
var workDir string

// function for index / route. redirects to default module from configuration file
func Index(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	http.Redirect(w, req, config.BaseURL+"/page/"+config.DefaultTemplate, 302)
}

// function for /page/:name routes
func Page(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println("page access", r.Host, r.RequestURI)

	session := getUserSession(w, r)

	pagename := ps.ByName("page")
	if !session.hasModuleAccess(pagename, render_context.Modules) {
		pagename = "no_access"
		w.WriteHeader(http.StatusUnauthorized)
	}
	if _, err := os.Stat(strings.Replace(workDir+"/"+config.Templates, "*", pagename, 1)); err != nil {
		pagename = notFoundPage
		log.Println(notFoundPage, r.Host, r.RequestURI)
	}
	ctx := render_context
	ctx.CurrentPage = pagename
	if user, ok := session.isAuthenticated(); ok {
		ctx.User = user
	}
	ctx.Modules = session.filterModules(render_context.Modules)
	if pagename == notFoundPage {
		w.WriteHeader(404)
	}
	render(w, pagename+".gohtml", ctx)
}

// function for /api/:name routes
func ApiModule(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println("api access", r.Host, r.RequestURI)

	module := ""
	if ps.ByName("module") != "" {
		module += ps.ByName("module") + "/"
	}
	runfile := workDir + "/" + strings.Replace(config.Modules[:strings.Index(config.Modules, "*")+1], "*", module+ps.ByName("name")+".yaml", 1)
	// change work dir
	//log.Print("Work dir:", workDir+"/"+runfile[:strings.LastIndex(runfile, ps.ByName("name"))])
	os.Chdir(runfile[:strings.LastIndex(runfile, ps.ByName("name"))])
	req_url, puerr := url.ParseRequestURI(r.RequestURI)
	if puerr != nil {
		log.Print("Can not parse request URL: ", puerr, r)
	}
	param, pqerr := url.ParseQuery(req_url.RawQuery)
	if pqerr != nil {
		log.Print("Can not parse request Query: ", pqerr, req_url)
	}
	if _, err := os.Stat(runfile); err == nil {
		//RenderFile(runfile, param, w)
		InteractiveExec(w, runfile, param)
	}
}

// function for /auth/:type
func Auth(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println("auth access", r.Host, r.RequestURI, ps)

	session := getUserSession(w, r)

	switch action := ps.ByName("type"); action {
	case "logout":
		session.Logout()
		return
	case "login":
		if user, ok := session.isAuthenticated(); ok {
			log.Print(user, " is already authenticated")
			http.Redirect(w, r, config.BaseURL, 302)
			return
		} else {
			session.Login()
			return
		}
	}

	if ps.ByName("type") != config.AuthType {
		http.Error(w, "This auth type is not allowed", http.StatusNotAcceptable)
		return
	}

	req_url, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	param, err := url.ParseQuery(req_url.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// checked all required fields, authenticate this user
	session.Authenticate(param)
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

func getParamKey(param map[string][]string, key string, idx int) (ret string) {
	ret = ""
	if val, ok := param[key]; ok {
		if len(val) >= idx+1 {
			ret = val[idx]
		}
	}
	return
}

func loadModules() {
	modDir := workDir + "/" + config.Modules
	log.Printf("Loading modules from %s", modDir)
	modPath, err := filepath.Glob(modDir)
	if err != nil {
		log.Fatal("Can not parse modules directory")
	}
	for _, module := range modPath {
		moddata, err := os.Open(module)
		if err != nil {
			log.Fatal("Can not open module file: ", module, err)
		}
		decoder := json.NewDecoder(moddata)
		modcfg := Module{}
		err = decoder.Decode(&modcfg)
		if err != nil {
			log.Fatal("Can not decode data from module file: ", err)
		}
		moddata.Close()
		if modcfg.Enabled {
			render_context.Modules = append(render_context.Modules, modcfg)
		}
	}
}

// application init. read configuration file, parse some data from it before the server started
func init() {
	// read command line arguments/flags
	config_file := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	// get configuration from defined json file
	cfg, err := os.Open(*config_file)
	defer cfg.Close()
	if err != nil {
		log.Fatal("Can not open configuration file: ", *config_file, err)
	}
	decoder := json.NewDecoder(cfg)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Can not decode data from config file: ", err)
	}

	// application init. parse templates and define Base URL value in global render context
	tpl = template.Must(template.ParseGlob(config.Templates))
	render_context.BaseURL = config.BaseURL
	render_context.AuthName = config.AuthName

	workDir, err = os.Getwd()
	if err != nil {
		log.Print("Can not get working directory")
	}
	loadModules()
	startSessions()
}

// main function. http routes setup and server starts and running here
func main() {
	// serve files from static folder
	fileServer := http.FileServer(http.Dir(workDir + "/static"))

	// router configuration
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/page/:page", Page)
	router.GET("/api/:module/:name", ApiModule)
	if config.AuthType != "" {
		router.GET("/auth/:type", Auth)
	}
	router.GET("/static/*filepath", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=432000") // 5 day cache
		r.URL.Path = p.ByName("filepath")
		fileServer.ServeHTTP(w, r)
	})
	log.Println("Server starting on ", config.ServerAddress)
	err := http.ListenAndServe(config.ServerAddress, router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
