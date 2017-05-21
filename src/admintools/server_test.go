package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"os"
	"strings"
)

func TestMain(m *testing.M) {

	appInit()
	os.Exit(m.Run())
}

func TestRedirect(t *testing.T) {
	httpContentCheck(t, "GET", "/", 302, "")
}

func TestPage(t *testing.T) {
	httpContentCheck(t, "GET", "/page/network", 200, "Basic network tools")
	httpContentCheck(t, "GET", "/page/about", 200, "Admin Tools release "+render_context.Version)
	httpContentCheck(t, "GET", "/page/not_found", 404, "NOT FOUND")
	httpContentCheck(t, "GET", "/page/some_not_existing_file", 404, "NOT FOUND")
}

func TestApi(t *testing.T) {
	httpContentCheck(t, "GET", "/api/test", 200, "{")
}

func httpContentCheck(t *testing.T, rmethod string, rpath string, estatus int, econtent string) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest(rmethod, rpath, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = rpath

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/page/:page", Page)
	router.GET("/api/:name", ApiModule)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != estatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, estatus)
	}

	// Check the response body is what we expect.
	if econtent != "" {
		if !strings.Contains(rr.Body.String(), econtent) {
			t.Errorf("%v handler returned unexpected body: got %v want %v",
				rpath, rr.Body.String(), econtent)
		}
	}
}
