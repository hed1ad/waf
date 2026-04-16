package enrich

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoInfo struct {
	CountryCode string
	CountryName string
	City        string
	ASN         uint32
	AsnOrg      string
}

type GeoEnricher struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

func NewGeoEnricher(cityPath, asnPath string) (*GeoEnricher, error) {
	cityDB, err := geoip2.Open(cityPath)
	if err != nil {
		return nil, err
	}
	asnDB, err := geoip2.Open(asnPath)
	if err != nil {
		cityDB.Close()
		return nil, err
	}
	return &GeoEnricher{cityDB: cityDB, asnDB: asnDB}, nil
}

func (g *GeoEnricher) Close() {
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.asnDB != nil {
		g.asnDB.Close()
	}
}

func (g *GeoEnricher) Lookup(ipStr string) GeoInfo {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoInfo{}
	}

	info := GeoInfo{}

	if g.cityDB != nil {
		if rec, err := g.cityDB.City(ip); err == nil {
			info.CountryCode = rec.Country.IsoCode
			if name, ok := rec.Country.Names["en"]; ok {
				info.CountryName = name
			}
			if name, ok := rec.City.Names["en"]; ok {
				info.City = name
			}
		}
	}

	if g.asnDB != nil {
		if rec, err := g.asnDB.ASN(ip); err == nil {
			info.ASN = uint32(rec.AutonomousSystemNumber)
			info.AsnOrg = rec.AutonomousSystemOrganization
		}
	}

	return info
}

// NoopEnricher is used when GeoIP databases are not available.
type NoopEnricher struct{}

func (NoopEnricher) Lookup(_ string) GeoInfo { return GeoInfo{} }
func (NoopEnricher) Close()                  {}
