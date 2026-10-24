package geo

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Navigraph "native" navdata (the wpnav* text files + one <icao>.xml per airport).
// Nodes are keyed "IDENT|lat,lon" because the same ident exists in many places.
type navdata struct {
	nodes   map[string]Point
	byIdent map[string][]string
	airways map[string]map[string][]string // airway name -> node key -> neighbour keys
	dir     string

	procMu sync.Mutex
	procs  map[string]procSet // ICAO -> SIDs/STARs (parsed lazily from <icao>.xml)
}

var (
	navMu sync.RWMutex
	nav   *navdata
)

func identOf(key string) string {
	if i := strings.IndexByte(key, '|'); i >= 0 {
		return key[:i]
	}
	return key
}

func (n *navdata) add(ident string, p Point) string {
	id := strings.ToUpper(strings.TrimSpace(ident))
	k := fmt.Sprintf("%s|%.4f,%.4f", id, p.Lat, p.Lon)
	if _, ok := n.nodes[k]; !ok {
		n.nodes[k] = p
		n.byIdent[id] = append(n.byIdent[id], k)
	}
	return k
}

func eachLine(path string, fn func(line string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if line == "" || line[0] == ';' {
			continue
		}
		fn(line)
	}
	return sc.Err()
}

var (
	fixCoordRe = regexp.MustCompile(`^\s*(-?\d+\.\d+)\s*(-?\d+\.\d+)`)
	aidRe      = regexp.MustCompile(`^(\S+)\s+\S+\s+(-?\d+\.\d{6})\s*(-?\d+\.\d{6})`)
)

func parseLatLon(a, b string) (Point, bool) {
	lat, e1 := strconv.ParseFloat(a, 64)
	lon, e2 := strconv.ParseFloat(b, 64)
	return Point{lat, lon}, e1 == nil && e2 == nil
}

// LoadNavdata reads the Navigraph native files from dir:
// airports.dat, wpnavfix.txt, wpnavaid.txt, wpnavrte.txt (and, lazily, <icao>.xml for STARs).
func LoadNavdata(dir string) error {
	n := &navdata{
		nodes:   make(map[string]Point, 400000),
		byIdent: make(map[string][]string, 400000),
		airways: make(map[string]map[string][]string),
		dir:     dir,
		procs:   make(map[string]procSet),
	}

	// airports.dat: ICAO(4) lat lon  e.g. "LFBD 44.828611  -0.715278"
	apts := map[string]Point{}
	if err := eachLine(filepath.Join(dir, "airports.dat"), func(l string) {
		if len(l) < 8 {
			return
		}
		if m := fixCoordRe.FindStringSubmatch(l[4:]); m != nil {
			if p, ok := parseLatLon(m[1], m[2]); ok {
				apts[strings.ToUpper(strings.TrimSpace(l[:4]))] = p
			}
		}
	}); err != nil {
		log.Println("geo: navdata airports.dat:", err)
	}
	if len(apts) > 0 {
		airportsMu.Lock()
		for k, v := range apts {
			airports[k] = v
		}
		airportsMu.Unlock()
	}

	// wpnavfix.txt: name(24) ident(5) lat lon
	if err := eachLine(filepath.Join(dir, "wpnavfix.txt"), func(l string) {
		if len(l) < 30 {
			return
		}
		if m := fixCoordRe.FindStringSubmatch(l[29:]); m != nil {
			if p, ok := parseLatLon(m[1], m[2]); ok {
				n.add(l[24:29], p)
			}
		}
	}); err != nil {
		log.Println("geo: navdata wpnavfix.txt:", err)
	}

	// wpnavaid.txt: name(24) ident type lat lon freq  (VOR, VORD, NDB, DME...)
	if err := eachLine(filepath.Join(dir, "wpnavaid.txt"), func(l string) {
		if len(l) < 30 {
			return
		}
		if m := aidRe.FindStringSubmatch(l[24:]); m != nil {
			if p, ok := parseLatLon(m[2], m[3]); ok {
				n.add(m[1], p)
			}
		}
	}); err != nil {
		log.Println("geo: navdata wpnavaid.txt:", err)
	}

	// wpnavrte.txt: AIRWAY seq ident lat lon (each airway is one ordered chain)
	segs := 0
	prevName, prevKey := "", ""
	if err := eachLine(filepath.Join(dir, "wpnavrte.txt"), func(l string) {
		f := strings.Fields(l)
		if len(f) != 5 {
			return
		}
		p, ok := parseLatLon(f[3], f[4])
		if !ok {
			return
		}
		name := strings.ToUpper(f[0])
		k := n.add(f[2], p)
		if name == prevName && k != prevKey {
			g := n.airways[name]
			if g == nil {
				g = map[string][]string{}
				n.airways[name] = g
			}
			g[prevKey] = append(g[prevKey], k)
			g[k] = append(g[k], prevKey)
			segs++
		}
		prevName, prevKey = name, k
	}); err != nil {
		log.Println("geo: navdata wpnavrte.txt:", err)
	}

	if len(n.nodes) == 0 {
		return errors.New("navdata: no fixes loaded from " + dir)
	}
	navMu.Lock()
	nav = n
	navMu.Unlock()
	log.Printf("geo: navdata loaded: %d points, %d airways (%d segments), %d airports", len(n.nodes), len(n.airways), segs, len(apts))
	return nil
}

func (n *navdata) nearest(keys []string, from Point) string {
	best, bestD := "", 1e18
	for _, k := range keys {
		if d := DistanceNM(from, n.nodes[k]); d < bestD {
			best, bestD = k, d
		}
	}
	return best
}

// expand walks airway g from startKey to the node whose ident is endIdent.
// It returns the intermediate points only (start and end excluded).
func (n *navdata) expand(g map[string][]string, startKey, endIdent string) ([]Point, bool) {
	start := startKey
	if _, ok := g[start]; !ok { // same ident, other place
		var cands []string
		for _, k := range n.byIdent[identOf(startKey)] {
			if _, ok := g[k]; ok {
				cands = append(cands, k)
			}
		}
		if len(cands) == 0 {
			return nil, false
		}
		start = n.nearest(cands, n.nodes[startKey])
	}

	parent := map[string]string{start: ""}
	queue := []string{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur != start && identOf(cur) == endIdent {
			var path []Point
			for k := parent[cur]; k != "" && k != start; k = parent[k] {
				path = append([]Point{n.nodes[k]}, path...)
			}
			return path, true
		}
		for _, nb := range g[cur] {
			if _, seen := parent[nb]; !seen {
				parent[nb] = cur
				queue = append(queue, nb)
			}
		}
	}
	return nil, false
}

var speedLevelRe = regexp.MustCompile(`^[NMK]\d{3,4}[FASM]\d{3,4}$`)

func parseCoord(tok string) (Point, bool) {
	if p := ParseRouteCoords(tok); len(p) == 1 {
		return p[0], true
	}
	return Point{}, false
}

// ResolveRoute converts a flight-plan route string into intermediate points
// (departure and arrival excluded): fixes, navaids, airway expansion and lat/lon waypoints.
// sid / star are the departure and arrival procedures: the one named in the route, or, when the
// route names none, a (stable) random one among those that connect to the first / last fix.
func ResolveRoute(depICAO string, dep Point, arrICAO, route string) (via, sid, star []Point) {
	navMu.RLock()
	n := nav
	navMu.RUnlock()

	var toks []string
	for _, t := range strings.Fields(strings.ToUpper(route)) {
		if i := strings.IndexByte(t, '/'); i >= 0 {
			t = t[:i]
		}
		if t == "" || t == "DCT" || t == "SID" || t == "STAR" || speedLevelRe.MatchString(t) {
			continue
		}
		toks = append(toks, t)
	}

	var fixes []string // named fixes met in the route, in order
	last, lastKey := dep, ""
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if p, ok := parseCoord(tok); ok {
			via = append(via, p)
			last, lastKey = p, ""
			continue
		}
		if n == nil {
			continue
		}
		if g, ok := n.airways[tok]; ok {
			if lastKey != "" && i+1 < len(toks) {
				if path, ok := n.expand(g, lastKey, toks[i+1]); ok {
					via = append(via, path...)
					if len(path) > 0 {
						last = path[len(path)-1]
					}
				}
			}
			continue
		}
		if keys, ok := n.byIdent[tok]; ok {
			k := n.nearest(keys, last)
			p := n.nodes[k]
			if DistanceNM(last, p) > 3500 { // wrong namesake on the other side of the world
				continue
			}
			via = append(via, p)
			last, lastKey = p, k
			fixes = append(fixes, tok)
		}
	}
	if n != nil {
		key := depICAO + "|" + arrICAO + "|" + route
		if depICAO != "" {
			sid = n.departure(strings.ToUpper(depICAO), toks, fixes, key)
			if len(sid) > 0 && (DistanceNM(dep, sid[0]) > 80 ||
				(len(via) > 0 && DistanceNM(sid[len(sid)-1], via[0]) > 400)) { // doesn't connect
				sid = nil
			}
		}
		if arrICAO != "" {
			star = n.arrival(strings.ToUpper(arrICAO), toks, fixes, key)
			if len(star) > 0 && DistanceNM(last, star[0]) > 800 {
				star = nil
			}
		}
	}
	log.Printf("geo: route %q -> %d intermediate points, SID %d pts, STAR %d pts", route, len(via), len(sid), len(star))
	return via, sid, star
}
