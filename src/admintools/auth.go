package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
)

const userSessionName = "user-session"

var sessionStore *sessions.CookieStore

type CustomSessions struct {
	*sessions.Session
	r *http.Request
	w http.ResponseWriter
}

func startSessions() {
	sessionStore = sessions.NewCookieStore([]byte(config.CookieSecret))
	sessionStore.Options = &sessions.Options{
		MaxAge: 60 * 60 * 24,
		Path:   "/",
	}
}

func getUserSession(w http.ResponseWriter, r *http.Request) *CustomSessions {
	session, err := sessionStore.Get(r, userSessionName)
	if err != nil {
		log.Print("ERROR Session", err.Error())
	}
	customSession := &CustomSessions{session, r, w}
	for k, v := range session.Values {
		customSession.Values[k] = v
	}
	return customSession
}

func (s *CustomSessions) Login() {
	if config.AuthType == "custom_sso" {
		t := template.Must(template.New("redirect_url").Parse(config.AuthParameters.RedirectURL))
		redirectURL := new(bytes.Buffer)
		t.Execute(redirectURL, render_context)
		http.Redirect(s.w, s.r, redirectURL.String(), 302)
	}
}

func (s *CustomSessions) Logout() {
	log.Printf("User %s logged out", s.Username())
	s.Options.MaxAge = -1
	delete(s.Values, "user")
	err := s.Save(s.r, s.w)
	if err != nil {
		http.Error(s.w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(s.w, s.r, config.BaseURL, http.StatusFound)
}

func (s *CustomSessions) Username() string {
	if username, ok := s.isAuthenticated(); ok {
		return username
	}
	return ""
}

func (s *CustomSessions) Authenticate(param map[string][]string) {
	if config.AuthType == "custom_sso" {
		user := getParamKey(param, "user", 0)
		hash := getParamKey(param, "hash", 0)
		if hash != "" && user != "" {
			user_ctx := render_context
			user_ctx.User = user
			t := template.Must(template.New("request_url").Parse(config.AuthParameters.Endpoint))
			getHashURL := new(bytes.Buffer)
			t.Execute(getHashURL, user_ctx)
			remoteHash, err := http.Get(getHashURL.String())
			if err != nil {
				http.Error(s.w, err.Error(), http.StatusInternalServerError)
				return
			}
			responseBody := new(bytes.Buffer)
			responseBody.ReadFrom(remoteHash.Body)
			if responseBody.String() == hash {
				s.Values["user"] = user
				s.Save(s.r, s.w)
				log.Printf("User %s logged in", user)
				http.Redirect(s.w, s.r, config.BaseURL, 302)
			}
		} else {
			http.Error(s.w, "User or Hash empty", http.StatusBadRequest)
		}
	}
}

func (s *CustomSessions) isAuthenticated() (string, bool) {
	var user string
	if user, ok := s.Values["user"].(string); ok {
		if len(user) > 0 {
			return user, true
		}
	}
	return user, false
}

func (s *CustomSessions) filterModules(modules []Module) []Module {
	permissions := false
	modules_ret := make([]Module, len(modules))
	_, permissions = s.isAuthenticated()
	for idx, module := range modules {
		if module.AccessLevel == "guest" {
			modules_ret = append(modules_ret, modules[idx])
		} else if module.AccessLevel != "guest" && permissions != false {
			modules_ret = append(modules_ret, modules[idx])
		}
	}
	return modules_ret
}

func (s *CustomSessions) hasModuleAccess(page string, modules []Module) bool {
	access := true
	_, permissions := s.isAuthenticated()
	for _, module := range modules {
		if module.Template == page && module.AccessLevel != "guest" && permissions == false {
			access = false
		}
	}
	return access
}
