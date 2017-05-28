package main

import (
	"bytes"
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
func Page(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("page access", req.Host, req.RequestURI)

	pagename := ps.ByName("page")
	if _, err := os.Stat(strings.Replace(workDir+"/"+config.Templates, "*", pagename, 1)); err != nil {
		pagename = notFoundPage
		log.Println(notFoundPage, req.Host, req.RequestURI)
	}
	ctx := render_context
	ctx.CurrentPage = pagename

	session, err := sessionStore.Get(req, "user-session")
	if err != nil {
		http.Error(w, "Can not find session", http.StatusInternalServerError)
		return
	}
	if user, ok := session.Values["user"]; ok {
		ctx.User = user.(string)
	}
	if pagename == notFoundPage {
		w.WriteHeader(404)
	}
	render(w, pagename+".gohtml", ctx)
}

// function for /api/:name routes
func ApiModule(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("api access", req.Host, req.RequestURI)

	module := ""
	if ps.ByName("module") != "" {
		module += ps.ByName("module") + "/"
	}
	runfile := workDir + "/" + strings.Replace(config.Modules[:strings.Index(config.Modules, "*")+1], "*", module+ps.ByName("name")+".yaml", 1)
	// change work dir
	//log.Print("Work dir:", workDir+"/"+runfile[:strings.LastIndex(runfile, ps.ByName("name"))])
	os.Chdir(runfile[:strings.LastIndex(runfile, ps.ByName("name"))])
	req_url, puerr := url.ParseRequestURI(req.RequestURI)
	if puerr != nil {
		log.Print("Can not parse request URL: ", puerr, req)
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

	session, err := sessionStore.Get(r, "user-session")
	if err != nil {
		http.Error(w, "Can not find session", http.StatusInternalServerError)
		return
	}

	if ps.ByName("type") == "logout" {
		if config.AuthType == "custom_sso" {
			session.Options.MaxAge = -1
			delete(session.Values, "user")
			err := session.Save(r, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			http.Redirect(w, r, config.BaseURL, http.StatusFound)
			return
		}
	}

	if _, ok := session.Values["user"]; ok {
		http.Redirect(w, r, config.BaseURL, 302)
		return
	}

	if ps.ByName("type") == "login" {
		if config.AuthType == "custom_sso" {
			t := template.Must(template.New("redirect_url").Parse(config.AuthParameters.RedirectURL))
			redirectURL := new(bytes.Buffer)
			t.Execute(redirectURL, render_context)
			http.Redirect(w, r, redirectURL.String(), 302)
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

	if config.AuthType == "custom_sso" {
		user := getParamKey(param, "user", 0)
		hash := getParamKey(param, "hash", 0)
		log.Printf("User: '%s' Hash: '%s'", user, hash)
		if hash != "" && user != "" {
			user_ctx := render_context
			user_ctx.User = user
			t := template.Must(template.New("request_url").Parse(config.AuthParameters.Endpoint))
			getHashURL := new(bytes.Buffer)
			t.Execute(getHashURL, user_ctx)
			remoteHash, err := http.Get(getHashURL.String())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			responseBody := new(bytes.Buffer)
			responseBody.ReadFrom(remoteHash.Body)
			if responseBody.String() == hash {
				session, err := sessionStore.Get(r, "user-session")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				session.Values["user"] = user
				session.Save(r, w)
				http.Redirect(w, r, config.BaseURL, 302)
			}
		} else {
			http.Error(w, "User or Hash empty", http.StatusBadRequest)
			return
		}
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

	workDir, err = os.Getwd()
	if err != nil {
		log.Print("Can not get working directory")
	}
	loadModules()
	if config.AuthType != "" {
		startSessions()
	}
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
