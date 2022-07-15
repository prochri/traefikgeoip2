package traefikgeoip2

import (
	"fmt"
	"net"

	"github.com/IncSW/geoip2"
)

// Unknown constant for undefined data.
const Unknown = "XX"

const (
	// RealIPHeader real ip header.
	RealIPHeader = "X-Real-IP"
)

// GeoIPResult GeoIPResult.
type GeoIPResult struct {
	country   string
	region    string
	city      string
	latitude  string
	longitude string
}

// LookupGeoIP2 LookupGeoIP2.
type LookupGeoIP2 func(ip net.IP) (*GeoIPResult, error)

// CreateCityDBLookup CreateCityDBLookup.
func CreateCityDBLookup(rdr *geoip2.CityReader) LookupGeoIP2 {
	return func(ip net.IP) (*GeoIPResult, error) {
		rec, err := rdr.Lookup(ip)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		retval := GeoIPResult{
			country:   rec.Country.ISOCode,
			region:    Unknown,
			city:      rec.City.Names["en"],
			latitude:  fmt.Sprintf("%f", rec.Location.Latitude),
			longitude: fmt.Sprintf("%f", rec.Location.Longitude),
		}
		if rec.Subdivisions != nil {
			retval.region = rec.Subdivisions[0].ISOCode
		}
		return &retval, nil
	}
}

// CreateCountryDBLookup CreateCountryDBLookup.
func CreateCountryDBLookup(rdr *geoip2.CountryReader) LookupGeoIP2 {
	return func(ip net.IP) (*GeoIPResult, error) {
		rec, err := rdr.Lookup(ip)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		retval := GeoIPResult{
			country: rec.Country.ISOCode,
			region:  Unknown,
			city:    Unknown,
		}
		return &retval, nil
	}
}
