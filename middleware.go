// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/IncSW/geoip2"
	cache "github.com/patrickmn/go-cache"
)

var (
	logInfo = log.New(ioutil.Discard, "geoip2-", log.Ldate|log.Ltime|log.Lshortfile)
	logWarn = log.New(ioutil.Discard, "geoip2-", log.Ldate|log.Ltime|log.Lshortfile)
	logErr  = log.New(ioutil.Discard, "geoip2-", log.Ldate|log.Ltime|log.Lshortfile)
)

// Headers part of the configuration
type Headers struct {
	Country   string `json:"country"`
	Region    string `json:"region"`
	City      string `json:"city"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// Config the plugin configuration.
type Config struct {
	DBPath  string   `json:"dbPath,omitempty"`
	Headers *Headers `json:"headers"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath:  "GeoLite2-Country.mmdb",
		Headers: &Headers{},
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next    http.Handler
	lookup  LookupGeoIP2
	name    string
	headers *Headers
	cache   *cache.Cache
}

// New created a new TraefikGeoIP2 plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(cfg.DBPath); err != nil {
		logErr.Printf("[geoip2] DB `%s' not found: %v", cfg.DBPath, err)
		return &TraefikGeoIP2{
			lookup: nil,
			next:   next,
			name:   name,
			cache:  nil,
		}, nil
	}

	var lookup LookupGeoIP2
	if strings.Contains(cfg.DBPath, "City") {
		rdr, err := geoip2.NewCityReaderFromFile(cfg.DBPath)
		if err != nil {
			logErr.Printf("[geoip2] DB `%s' not initialized: %v", cfg.DBPath, err)
		} else {
			lookup = CreateCityDBLookup(rdr)
		}
	}

	if strings.Contains(cfg.DBPath, "Country") {
		rdr, err := geoip2.NewCountryReaderFromFile(cfg.DBPath)
		if err != nil {
			logErr.Printf("[geoip2] DB `%s' not initialized: %v", cfg.DBPath, err)
		} else {
			lookup = CreateCountryDBLookup(rdr)
		}
	}

	return &TraefikGeoIP2{
		lookup:  lookup,
		next:    next,
		name:    name,
		headers: cfg.Headers,
		cache:   cache.New(DefaultCacheExpire, DefaultCachePurge),
	}, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if mw.lookup == nil {
		logErr.Println("The db path must contains City/Country")
		mw.next.ServeHTTP(rw, req)
		return
	}

	ipStr := req.Header.Get(RealIPHeader)
	if ipStr == "" {
		ipStr = req.RemoteAddr
		tmp, _, err := net.SplitHostPort(ipStr)
		if err == nil {
			ipStr = tmp
		}
	}

	var (
		record *GeoIPResult
		err    error
	)

	if c, found := mw.cache.Get(ipStr); found {
		record = c.(*GeoIPResult)
	} else {
		record, err = mw.lookup(net.ParseIP(ipStr))
		if err != nil {
			logWarn.Printf("Unable to find GeoIP data for `%s', %v", ipStr, err)
			record = &GeoIPResult{
				country:   Unknown,
				region:    Unknown,
				city:      Unknown,
				latitude:  Unknown,
				longitude: Unknown,
			}
		}
		mw.cache.Set(ipStr, record, cache.DefaultExpiration)
	}

	mw.addHeaders(req, record)

	mw.next.ServeHTTP(rw, req)
}

func (a *TraefikGeoIP2) addHeaders(req *http.Request, record *GeoIPResult) {
	if a.headers.Country != "" {
		req.Header.Add(a.headers.Country, record.country)
	}
	if a.headers.Region != "" {
		req.Header.Add(a.headers.Region, record.region)
	}
	if a.headers.City != "" {
		req.Header.Add(a.headers.City, record.city)
	}
	if a.headers.Latitude != "" {
		req.Header.Add(a.headers.Latitude, record.latitude)
	}
	if a.headers.Longitude != "" {
		req.Header.Add(a.headers.Longitude, record.longitude)
	}
}
