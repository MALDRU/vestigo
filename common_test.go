package vestigo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAddRoute(t *testing.T) {
	router := NewRouter()
	m := map[string]func(path string, handler http.HandlerFunc){
		"GET":     router.Get,
		"POST":    router.Post,
		"CONNECT": router.Connect,
		"DELETE":  router.Delete,
		"PATCH":   router.Patch,
		"PUT":     router.Put,
		"TRACE":   router.Trace,
	}
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("success-" + r.Method))
	}
	path := "/test"
	for _, v := range m {
		v(path, f)
	}
	for k, _ := range m {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(k, path, nil)
		if err != nil {
			t.Errorf("Failed to create a new request, method: %s, path: %s", k, path)
		}
		router.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "success-"+k {
			t.Errorf("Invalid response, method: %s, path: %s, code: %s, body: %s", k, path, w.Code, w.Body.String())
		}
	}
}

func TestTrace(t *testing.T) {
	router := NewRouter()
	AllowTrace = true
	defer func() {
		AllowTrace = false
	}()
	path := "/test"
	router.Get(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("TRACE", path, bytes.NewBufferString("awesome trace"))
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "TRACE", path)
	}
	router.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "awesome trace" || w.Header().Get("Content-Type") != "message/http" {
		t.Errorf("Invalid TRACE response, method: %s, path: %s, code: %s, body: %s", "TRACE", path, w.Code, w.Body.String())
	}
}

func TestHead(t *testing.T) {
	router := NewRouter()
	path := "/test"
	router.Get(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-TestHeader", "true")
		w.WriteHeader(200)
		w.Write([]byte("some return body"))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("HEAD", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "HEAD", path)
	}
	router.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "" || w.Header().Get("X-TestHeader") != "true" {
		t.Errorf("Invalid HEAD response, method: %s, path: %s, code: %s, body: %s", "HEAD", path, w.Code, w.Body.String())
	}
}

func TestMethodNotFoundDifferentMethodAllowed(t *testing.T) {
	router := NewRouter()
	path := "/test"
	router.Add("GET", path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("some return body"))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "GETFAKEMETHOD", path)
	}
	router.ServeHTTP(w, r)

	if w.Code != 405 || w.Body.String() != "Method Not Allowed" {
		t.Errorf("Invalid GETFAKEMETHOD response, method: %s, path: %s, code: %s, body: %s", "GETFAKEMETHOD", path, w.Code, w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	router := NewRouter()
	path := "/test"
	router.Add("GET", path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("some return body"))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", path+"broken", nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "GETFAKEMETHOD", path)
	}
	router.ServeHTTP(w, r)

	if w.Code != 404 || w.Body.String() != "Not Found" {
		t.Errorf("Invalid response, method: %s, path: %s, code: %s, body: %s", "GET", path, w.Code, w.Body.String())
	}
}

func TestMethodNotFound(t *testing.T) {
	router := NewRouter()
	path := "/test"
	router.Add("GET", path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("some return body"))
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("FAKEMETHOD", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "GETFAKEMETHOD", path)
	}
	router.ServeHTTP(w, r)

	if w.Code != 405 || w.Body.String() != "Method Not Allowed" {
		t.Errorf("Invalid GETFAKEMETHOD response, method: %s, path: %s, code: %s, body: %s", "GETFAKEMETHOD", path, w.Code, w.Body.String())
	}
}
func TestCorsPreflight(t *testing.T) {
	router := NewRouter()
	router.SetGlobalCors(&CorsAccessControl{
		AllowOrigin:      []string{"*", "test.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Header", "X-Y-Header"},
		MaxAge:           3600 * time.Second,
		AllowHeaders:     []string{"X-Header", "X-Y-Header"},
	})

	path := "/test"
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	}
	router.Get(path, f)
	router.Post(path, f)

	router.SetCors(path, &CorsAccessControl{
		AllowMethods: []string{"GET"},                    // only allow cors for this resource on GET calls
		AllowHeaders: []string{"X-Header", "X-Z-Header"}, // Allow this one header for this resource
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "OPTIONS", path)
	}

	// add preflight headers
	r.Header.Add("Origin", "test.com")
	r.Header.Add("Access-Control-Request-Method", "GET")
	r.Header.Add("Access-Control-Request-Headers", "X-Y-Header")

	router.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "" || w.Header().Get("Access-Control-Allow-Origin") != "test.com" {
		t.Errorf("Invalid OPTIONS response, method: %s, path: %s, code: %s, body: %s", "OPTIONS", path, w.Code, w.Body.String())
	}

}

func TestCorsWildcardPreflight(t *testing.T) {
	router := NewRouter()
	router.SetGlobalCors(&CorsAccessControl{
		AllowOrigin:      []string{"*", "test.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Header", "X-Y-Header"},
		MaxAge:           3600 * time.Second,
		AllowHeaders:     []string{"X-Header", "X-Y-Header"},
	})

	path := "/test"
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	}
	router.Get(path, f)
	router.Post(path, f)

	router.SetCors(path, &CorsAccessControl{
		AllowMethods: []string{"GET"},                    // only allow cors for this resource on GET calls
		AllowHeaders: []string{"X-Header", "X-Z-Header"}, // Allow this one header for this resource
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "OPTIONS", path)
	}

	// add preflight headers
	r.Header.Add("Origin", "wildcardtest.com")
	r.Header.Add("Access-Control-Request-Method", "GET")
	r.Header.Add("Access-Control-Request-Headers", "X-Y-Header")

	router.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "" || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Invalid OPTIONS response, method: %s, path: %s, code: %s, body: %s", "OPTIONS", path, w.Code, w.Body.String())
	}

}

func TestFailCorsPreflight(t *testing.T) {
	router := NewRouter()
	router.SetGlobalCors(&CorsAccessControl{
		AllowOrigin:      []string{"test.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Header", "X-Y-Header"},
		MaxAge:           3600 * time.Second,
		AllowHeaders:     []string{"X-Header", "X-Y-Header"},
	})

	path := "/test"
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	}
	router.Get(path, f)
	router.Post(path, f)

	router.SetCors(path, &CorsAccessControl{
		AllowMethods: []string{"GET"},                    // only allow cors for this resource on GET calls
		AllowHeaders: []string{"X-Header", "X-Z-Header"}, // Allow this one header for this resource
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "OPTIONS", path)
	}

	// add preflight headers
	r.Header.Add("Origin", "badtest.com")
	r.Header.Add("Access-Control-Request-Method", "GET")
	r.Header.Add("Access-Control-Request-Headers", "X-Y-Header")

	router.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") == "badtest.com" {
		t.Errorf("should have failed preflight, but didn't")
	}

}

func TestFailCorsBadMethodsPreflight(t *testing.T) {
	router := NewRouter()
	router.SetGlobalCors(&CorsAccessControl{
		AllowOrigin:      []string{"test.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Header", "X-Y-Header"},
		MaxAge:           3600 * time.Second,
		AllowHeaders:     []string{"X-Header", "X-Y-Header"},
	})

	path := "/test"
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	}
	router.Post(path, f)

	router.SetCors(path, &CorsAccessControl{
		AllowHeaders: []string{"X-Header", "X-Z-Header"}, // Allow this one header for this resource
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "OPTIONS", path)
	}

	// add preflight headers
	r.Header.Add("Origin", "badtest.com")
	r.Header.Add("Access-Control-Request-Method", "GET")
	r.Header.Add("Access-Control-Request-Headers", "X-Y-Header")

	router.ServeHTTP(w, r)
	if strings.Contains(w.Header().Get("Access-Control-Allow-Method"), "GET") {
		t.Errorf("should have failed preflight, but didn't")
	}

}

func TestFailCorsNoMethodsPreflight(t *testing.T) {
	router := NewRouter()
	router.SetGlobalCors(&CorsAccessControl{
		AllowOrigin:      []string{"test.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Header", "X-Y-Header"},
		MaxAge:           3600 * time.Second,
		AllowHeaders:     []string{"X-Header", "X-Y-Header"},
	})

	path := "/test"
	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(""))
	}
	router.Get(path, f)
	router.Post(path, f)

	router.SetCors(path, &CorsAccessControl{
		AllowMethods: []string{},                         // only allow cors for this resource on GET calls
		AllowHeaders: []string{"X-Header", "X-Z-Header"}, // Allow this one header for this resource
	})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("OPTIONS", path, nil)
	if err != nil {
		t.Errorf("Failed to create a new request, method: %s, path: %s", "OPTIONS", path)
	}

	// add preflight headers
	r.Header.Add("Origin", "badtest.com")
	r.Header.Add("Access-Control-Request-Method", "GET")
	r.Header.Add("Access-Control-Request-Headers", "X-Y-Header")

	router.ServeHTTP(w, r)
	if strings.Contains(w.Header().Get("Access-Control-Allow-Method"), "GET") {
		t.Errorf("should have failed preflight, but didn't")
	}

}