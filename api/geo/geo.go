// Package geo builds a static map image URL (Mapbox Static Images API)
// showing a flight's planned route and the aircraft's current position.
package geo

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Point struct {
	Lat, Lon float64
}

// ---------------------------------------------------------------------------
// Airport database (OurAirports, public domain)
// ---------------------------------------------------------------------------

const airportsURL = "https://davidmegginson.github.io/ourairports-data/airports.csv"

var (
	airportsMu sync.RWMutex
	airports   = map[string]Point{}
)

// LoadAirports downloads the OurAirports CSV and indexes it by ICAO / ident / GPS code.
// Call it once at startup (it is ~10 MB, so do it in a goroutine).
func LoadAirports() error {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(airportsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("airports: unexpected status %s", resp.Status)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return err
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	iIdent, okA := col["ident"]
	iLat, okB := col["latitude_deg"]
	iLon, okC := col["longitude_deg"]
	iGps, hasGps := col["gps_code"]
	if !okA || !okB || !okC {
		return errors.New("airports: unexpected CSV header")
	}

	m := make(map[string]Point, 80000)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(rec) <= max(iIdent, iLat, iLon) {
			continue
		}
		lat, e1 := strconv.ParseFloat(rec[iLat], 64)
		lon, e2 := strconv.ParseFloat(rec[iLon], 64)
		if e1 != nil || e2 != nil {
			continue
		}
		p := Point{lat, lon}
		m[strings.ToUpper(rec[iIdent])] = p
		if hasGps && len(rec) > iGps && rec[iGps] != "" {
			if _, exists := m[strings.ToUpper(rec[iGps])]; !exists {
				m[strings.ToUpper(rec[iGps])] = p
			}
		}
	}

	airportsMu.Lock()
	for k, v := range m { // merge: never override airports already loaded from navdata
		if _, ok := airports[k]; !ok {
			airports[k] = v
		}
	}
	airportsMu.Unlock()
	log.Printf("geo: loaded %d airports", len(m))
	return nil
}

func Airport(icao string) (Point, bool) {
	airportsMu.RLock()
	defer airportsMu.RUnlock()
	p, ok := airports[strings.ToUpper(strings.TrimSpace(icao))]
	return p, ok
}

// ---------------------------------------------------------------------------
// Geometry
// ---------------------------------------------------------------------------

func rad(d float64) float64 { return d * math.Pi / 180 }
func deg(r float64) float64 { return r * 180 / math.Pi }

// centralAngle returns the great-circle angle between a and b, in radians.
func centralAngle(a, b Point) float64 {
	dLat := rad(b.Lat - a.Lat)
	dLon := rad(b.Lon - a.Lon)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(a.Lat))*math.Cos(rad(b.Lat))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * math.Asin(math.Min(1, math.Sqrt(h)))
}

func DistanceNM(a, b Point) float64 { return centralAngle(a, b) * 3440.065 }

// GreatCircle returns n+1 points along the great circle from a to b.
func GreatCircle(a, b Point, n int) []Point {
	d := centralAngle(a, b)
	if d < 1e-9 || n < 1 {
		return []Point{a, b}
	}
	pts := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		A := math.Sin((1-f)*d) / math.Sin(d)
		B := math.Sin(f*d) / math.Sin(d)
		x := A*math.Cos(rad(a.Lat))*math.Cos(rad(a.Lon)) + B*math.Cos(rad(b.Lat))*math.Cos(rad(b.Lon))
		y := A*math.Cos(rad(a.Lat))*math.Sin(rad(a.Lon)) + B*math.Cos(rad(b.Lat))*math.Sin(rad(b.Lon))
		z := A*math.Sin(rad(a.Lat)) + B*math.Sin(rad(b.Lat))
		pts = append(pts, Point{deg(math.Atan2(z, math.Hypot(x, y))), deg(math.Atan2(y, x))})
	}
	return pts
}

// RoutePoints chains great-circle legs dep -> via... -> arr.
func RoutePoints(dep, arr Point, via []Point) []Point {
	wps := append([]Point{dep}, via...)
	wps = append(wps, arr)

	var out []Point
	for i := 0; i < len(wps)-1; i++ {
		n := int(DistanceNM(wps[i], wps[i+1]) / 150)
		n = min(max(n, 1), 24)
		leg := GreatCircle(wps[i], wps[i+1], n)
		if i > 0 {
			leg = leg[1:]
		}
		out = append(out, leg...)
	}
	unwrap(out)
	return out
}

// unwrap makes longitudes continuous so routes crossing the antimeridian don't wrap around the globe.
func unwrap(pts []Point) {
	for i := 1; i < len(pts); i++ {
		for pts[i].Lon-pts[i-1].Lon > 180 {
			pts[i].Lon -= 360
		}
		for pts[i].Lon-pts[i-1].Lon < -180 {
			pts[i].Lon += 360
		}
	}
}

func normLon(l float64) float64 {
	for l > 180 {
		l -= 360
	}
	for l < -180 {
		l += 360
	}
	return l
}

// ---------------------------------------------------------------------------
// Route parsing (lat/lon waypoints only for now, e.g. 46N015E or 4630N01530E)
// ---------------------------------------------------------------------------

var coordRe = regexp.MustCompile(`^(\d{2})(\d{2})?([NS])(\d{3})(\d{2})?([EW])$`)

func ParseRouteCoords(route string) []Point {
	var pts []Point
	for _, tok := range strings.Fields(strings.ToUpper(route)) {
		if i := strings.Index(tok, "/"); i >= 0 { // strip "/N0450F360"
			tok = tok[:i]
		}
		m := coordRe.FindStringSubmatch(tok)
		if m == nil {
			continue
		}
		lat := num(m[1]) + num(m[2])/60
		lon := num(m[4]) + num(m[5])/60
		if m[3] == "S" {
			lat = -lat
		}
		if m[6] == "W" {
			lon = -lon
		}
		pts = append(pts, Point{lat, lon})
		if len(pts) >= 20 {
			break
		}
	}
	return pts
}

func num(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ---------------------------------------------------------------------------
// Polyline encoding (Google format, precision 5)
// ---------------------------------------------------------------------------

func encodePolyline(pts []Point) string {
	var b strings.Builder
	var pLat, pLon int
	for _, p := range pts {
		lat := int(math.Round(p.Lat * 1e5))
		lon := int(math.Round(p.Lon * 1e5))
		encodeValue(&b, lat-pLat)
		encodeValue(&b, lon-pLon)
		pLat, pLon = lat, lon
	}
	return b.String()
}

func encodeValue(b *strings.Builder, n int) {
	v := n << 1
	if n < 0 {
		v = ^v
	}
	for v >= 0x20 {
		b.WriteByte(byte((0x20 | (v & 0x1f)) + 63))
		v >>= 5
	}
	b.WriteByte(byte(v + 63))
}

// ---------------------------------------------------------------------------
// Map URL
// ---------------------------------------------------------------------------

func marker(style string, p Point) string {
	return fmt.Sprintf("%s(%.5f,%.5f)", style, normLon(p.Lon), p.Lat)
}

const maxURLLen = 2000 // Discord rejects very long embed image URLs

func thin(via []Point) []Point {
	if len(via) <= 1 {
		return nil
	}
	out := make([]Point, 0, len(via)/2+1)
	for i := 0; i < len(via); i += 2 {
		out = append(out, via[i])
	}
	return out
}

// FlightMapURL returns a Mapbox static map URL, or "" when no map can be built
// (no MAPBOX_TOKEN, or nothing locatable). cur may be nil (e.g. for prefiles).
func FlightMapURL(depICAO, arrICAO, route string, cur *Point) string {
	token := os.Getenv("MAPBOX_TOKEN")
	if token == "" {
		log.Println("geo: MAPBOX_TOKEN is empty, no map (check .env and restart the bot)")
		return ""
	}
	dep, okD := Airport(depICAO)
	arr, okA := Airport(arrICAO)

	count := 0
	if okD {
		count++
	}
	if okA {
		count++
	}
	if cur != nil {
		count++
	}
	if count == 0 {
		log.Printf("geo: nothing locatable (dep=%q found=%v, arr=%q found=%v) - airports loaded yet?", depICAO, okD, arrICAO, okA)
		return ""
	}

	view := "auto"
	if count == 1 { // "auto" needs at least two things to frame
		c := dep
		switch {
		case cur != nil:
			c = *cur
		case okA:
			c = arr
		}
		view = fmt.Sprintf("%.4f,%.4f,6", normLon(c.Lon), c.Lat)
	}

	var via, sid, star []Point
	if okD && okA {
		via, sid, star = ResolveRoute(depICAO, dep, arrICAO, route)
	}

	build := func(via []Point) string {
		var overlays []string
		if okD && okA {
			mainVia := append(append(append([]Point{}, sid...), via...), star...)
			pts := RoutePoints(dep, arr, mainVia)
			overlays = append(overlays, "path-4+5865f2-0.9("+url.QueryEscape(encodePolyline(pts))+")")
			if len(sid) > 1 { // departure procedure on top in teal
				sp := RoutePoints(sid[0], sid[len(sid)-1], sid[1:len(sid)-1])
				overlays = append(overlays, "path-5+2dd4bf-0.95("+url.QueryEscape(encodePolyline(sp))+")")
			}
			if len(star) > 0 { // arrival procedure on top in orange
				sp := RoutePoints(star[0], arr, star[1:])
				overlays = append(overlays, "path-5+f0a020-0.95("+url.QueryEscape(encodePolyline(sp))+")")
			}
		}
		if okD {
			overlays = append(overlays, marker("pin-s-d+57f287", dep))
		}
		if okA {
			overlays = append(overlays, marker("pin-s-a+fee75c", arr))
		}
		if cur != nil {
			overlays = append(overlays, marker("pin-l-airport+ed4245", *cur))
		}
		return fmt.Sprintf(
			"https://api.mapbox.com/styles/v1/mapbox/dark-v11/static/%s/%s/900x500@2x?padding=70&logo=false&attribution=false&access_token=%s",
			strings.Join(overlays, ","), view, token)
	}

	u := build(via)
	for len(u) > maxURLLen && (len(via) > 0 || len(star) > 0 || len(sid) > 0) {
		switch {
		case len(via) > 0:
			via = thin(via)
		case len(star) > 0:
			star = thin(star)
		default:
			sid = thin(sid)
		}
		u = build(via)
	}
	if len(u) > maxURLLen {
		log.Printf("geo: map URL too long (%d chars), skipped", len(u))
		return ""
	}
	return u
}
