package traefikgeoip2_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mw "github.com/prochri/traefikgeoip2"
)

const (
	ValidIP       = "188.193.88.199"
	ValidIPNoCity = "20.1.184.61"
	LocalIP       = "10.0.0.42"
)

func TestGeoIPConfig(t *testing.T) {
	mwCfg := mw.CreateConfig()
	if mw.CreateConfig().DBPath != mwCfg.DBPath {
		t.Fatalf("Incorrect path")
	}

	mwCfg.DBPath = "./non-existing"
	mw.ResetLookup()
	_, err := mw.New(context.TODO(), nil, mwCfg, "")
	if err != nil {
		t.Fatalf("Must not fail on missing DB")
	}

	mwCfg.DBPath = "Makefile"
	_, err = mw.New(context.TODO(), nil, mwCfg, "")
	if err != nil {
		t.Fatalf("Must not fail on invalid DB format")
	}
}

func TestGeoIPBasic(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-City.mmdb"

	called := false
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { called = true })

	mw.ResetLookup()
	instance, err := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")
	if err != nil {
		t.Fatalf("Error creating %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

	instance.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Invalid return code")
	}
	if called != true {
		t.Fatalf("next handler was not called")
	}
}

func TestMissingGeoIPDB(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./missing"

	called := false
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { called = true })

	mw.ResetLookup()
	instance, err := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")
	if err != nil {
		t.Fatalf("Error creating %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

	instance.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Invalid return code")
	}
	if called != true {
		t.Fatalf("next handler was not called")
	}
	// TODO: assert no header
	// hearders := mw.CreateConfig().Headers
	// assertHeader(t, req, hearders.Country, mw.Unknown)
	// assertHeader(t, req, hearders.Region, mw.Unknown)
	// assertHeader(t, req, hearders.City, mw.Unknown)
}

func TestGeoIPFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-City.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	mw.ResetLookup()
	instance, _ := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = fmt.Sprintf("%s:9999", ValidIP)
	instance.ServeHTTP(httptest.NewRecorder(), req)
	hearders := mw.CreateConfig().Headers
	assertHeader(t, req, hearders.Country, "DE")
	assertHeader(t, req, hearders.Region, "BY")
	assertHeader(t, req, hearders.City, "Munich")

	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "qwerty:9999"
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, hearders.Country, mw.Unknown)
	assertHeader(t, req, hearders.Region, mw.Unknown)
	assertHeader(t, req, hearders.City, mw.Unknown)
}

func TestGeoIPCountryDBFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-Country.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	mw.ResetLookup()
	instance, _ := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = fmt.Sprintf("%s:9999", ValidIP)
	instance.ServeHTTP(httptest.NewRecorder(), req)

	hearders := mw.CreateConfig().Headers
	assertHeader(t, req, hearders.Country, "DE")
	assertHeader(t, req, hearders.Region, mw.Unknown)
	assertHeader(t, req, hearders.City, mw.Unknown)
}

func TestLocalAddress(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-Country.mmdb"
	mwCfg.LocationRewrites = append(mwCfg.LocationRewrites, mw.LocationRewrite{
		IpRange: "10.0.0.0/8",
		Country: "DE",
		Region:  "BY",
		City:    "Munich",
	})

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = fmt.Sprintf("%s:9999", LocalIP)
	instance.ServeHTTP(httptest.NewRecorder(), req)
	hearders := mw.CreateConfig().Headers
	assertHeader(t, req, hearders.Country, "DE")
	assertHeader(t, req, hearders.Region, "BY")
	assertHeader(t, req, hearders.City, "Munich")
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Fatalf("invalid value of header [%s] != %s", key, req.Header.Get(key))
	}
}
