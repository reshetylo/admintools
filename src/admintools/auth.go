package main

import (
	"github.com/gorilla/sessions"
)

var sessionStore *sessions.CookieStore

func startSessions() {
	sessionStore = sessions.NewCookieStore([]byte(config.CookieSecret))
}
