// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/IncSW/geoip2"
)

type LocationRewrite struct {
	IpRange string `json:"ipRange"`
	Country string `json:"country,omitempty"`
	Region  string `json:"region,omitempty"`
	City    string `json:"city,omitempty"`
	IPnet   *net.IPNet
}

// Config the plugin configuration.
type Config struct {
	DBPath           string            `json:"dbPath,omitempty"`
	LocationRewrites []LocationRewrite `json:"locationRewrites,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath: DefaultDBPath,
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next             http.Handler
	lookup           LookupGeoIP2
	name             string
	locationRewrites []LocationRewrite
}

// New created a new TraefikGeoIP2 plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	var err error
	for i := range cfg.LocationRewrites {
		_, cfg.LocationRewrites[i].IPnet, err = net.ParseCIDR(cfg.LocationRewrites[i].IpRange)
		if err != nil {
			break
		}
	}
	if err != nil {
		return &TraefikGeoIP2{
			lookup:           nil,
			next:             next,
			name:             name,
			locationRewrites: cfg.LocationRewrites,
		}, nil
	}

	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("[geoip2] DB `%s' not found: %v", cfg.DBPath, err)
		return &TraefikGeoIP2{
			lookup:           nil,
			next:             next,
			name:             name,
			locationRewrites: cfg.LocationRewrites,
		}, nil
	}

	var lookup LookupGeoIP2
	if strings.Contains(cfg.DBPath, "City") {
		rdr, err := geoip2.NewCityReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("[geoip2] DB `%s' not initialized: %v", cfg.DBPath, err)
		} else {
			lookup = CreateCityDBLookup(rdr)
		}
	}

	if strings.Contains(cfg.DBPath, "Country") {
		rdr, err := geoip2.NewCountryReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("[geoip2] DB `%s' not initialized: %v", cfg.DBPath, err)
		} else {
			lookup = CreateCountryDBLookup(rdr)
		}
	}

	return &TraefikGeoIP2{
		lookup:           lookup,
		next:             next,
		name:             name,
		locationRewrites: cfg.LocationRewrites,
	}, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(reqWr http.ResponseWriter, req *http.Request) {
	if mw.lookup == nil {
		req.Header.Set(CountryHeader, Unknown)
		req.Header.Set(RegionHeader, Unknown)
		req.Header.Set(CityHeader, Unknown)
		mw.next.ServeHTTP(reqWr, req)
		return
	}

	ipStr := req.RemoteAddr
	tmp, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = tmp
	}

	ip := net.ParseIP(ipStr)
	res, err := mw.lookup(ip)
	if err != nil {
		res, err = mw.findLocalRewrite(ip)
		if err != nil {
			log.Printf("[geoip2] Unable to find for `%s', %v", ipStr, err)
			res = &GeoIPResult{
				country: Unknown,
				region:  Unknown,
				city:    Unknown,
			}
		}
	}

	req.Header.Set(CountryHeader, res.country)
	req.Header.Set(RegionHeader, res.region)
	req.Header.Set(CityHeader, res.city)

	mw.next.ServeHTTP(reqWr, req)
}

func (mw *TraefikGeoIP2) findLocalRewrite(ip net.IP) (*GeoIPResult, error) {
	for _, lr := range mw.locationRewrites {
		if lr.IPnet.Contains(ip) {
			return &GeoIPResult{
				country: lr.Country,
				region:  lr.Region,
				city:    lr.City,
			}, nil
		}
	}
	return nil, geoip2.ErrNotFound
}
