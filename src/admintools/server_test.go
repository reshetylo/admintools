package main

import (
	"net/http"
	"net/http/httptest"
	//	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
)

//func TestMain(m *testing.M) {
//	os.Exit(m.Run())
//}

var ps httprouter.Params

func TestRedirect(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		Index(w, r, nil)
	}
	httpContentCheck(*t, "GET", "/", handler, 302, "")
}

func TestPage(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		Page(w, r, ps)
	}
	httpContentCheck(*t, "GET", "/page/:name", handler, 200, "")
}

func TestApi(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		ApiModule(w, r, ps)
	}
	httpContentCheck(*t, "GET", "/api/:name", handler, 200, "")
}

func httpContentCheck(t testing.T, rtype string, rpath string, rhandler http.HandlerFunc, estatus int, econtent string) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest(rtype, rpath, nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(rhandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != estatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, estatus)
	}

	// Check the response body is what we expect.
	if econtent != "" {
		expected := string(econtent)
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	}
}
