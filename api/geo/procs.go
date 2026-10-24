package geo

import (
	"bytes"
	"encoding/xml"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Per-airport procedures file (<icao>.xml, lowercase) from the Navigraph native data:
// <Sid Name=".." Runways=".."> / <Star ...> with <Sid_Waypoint> / <Star_Waypoint> children
// that carry their own coordinates.

type xmlWpt struct {
	Name string  `xml:"Name"`
	Type string  `xml:"Type"`
	Lat  float64 `xml:"Latitude"`
	Lon  float64 `xml:"Longitude"`
}

type xmlProc struct {
	Name     string   `xml:"Name,attr"`
	Runways  string   `xml:"Runways,attr"`
	SidWpts  []xmlWpt `xml:"Sid_Waypoint"`
	StarWpts []xmlWpt `xml:"Star_Waypoint"`
}

type xmlDoc struct {
	Sids  []xmlProc `xml:"Airport>Sid"`
	Stars []xmlProc `xml:"Airport>Star"`
}

type proc struct {
	name, runways string
	names         []string
	pts           []Point
}

type procSet struct{ sids, stars []proc }

func toProcs(xs []xmlProc, isStar bool) []proc {
	var out []proc
	for _, x := range xs {
		w := x.SidWpts
		if isStar {
			w = x.StarWpts
		}
		p := proc{name: strings.ToUpper(x.Name), runways: x.Runways}
		for _, wp := range w {
			switch wp.Type {
			case "Hold", "ConstHdgtoAlt":
				continue
			case "Runway":
				if isStar { // the landing runway is not drawn
					continue
				}
			}
			if wp.Lat == 0 && wp.Lon == 0 {
				continue
			}
			p.names = append(p.names, strings.ToUpper(wp.Name))
			p.pts = append(p.pts, Point{wp.Lat, wp.Lon})
		}
		if len(p.pts) > 0 {
			out = append(out, p)
		}
	}
	return out
}

func (n *navdata) procsFor(icao string) procSet {
	n.procMu.Lock()
	defer n.procMu.Unlock()
	if s, ok := n.procs[icao]; ok {
		return s
	}
	n.procs[icao] = procSet{} // cache failures too

	raw, err := os.ReadFile(filepath.Join(n.dir, strings.ToLower(icao)+".xml"))
	if err != nil {
		log.Printf("geo: no procedures file for %s (%v)", icao, err)
		return procSet{}
	}
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	var doc xmlDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		log.Printf("geo: %s.xml: %v", strings.ToLower(icao), err)
		return procSet{}
	}
	s := procSet{sids: toProcs(doc.Sids, false), stars: toProcs(doc.Stars, true)}
	n.procs[icao] = s
	return s
}

var procRe = regexp.MustCompile(`^([A-Z]+)(\d[A-Z]?)$`)

// sameProc tolerates the 5-letter vs ARINC-truncated naming (MIRBA2K vs MIRB2K).
func sameProc(a, b string) bool {
	ma, mb := procRe.FindStringSubmatch(a), procRe.FindStringSubmatch(b)
	if ma == nil || mb == nil || ma[2] != mb[2] {
		return false
	}
	return strings.HasPrefix(ma[1], mb[1]) || strings.HasPrefix(mb[1], ma[1])
}

// choose picks one candidate. The pick is pseudo-random but stable for a given key
// (same flight -> same procedure on every call).
func choose(c []proc, key string) proc {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c[int(h.Sum32()%uint32(len(c)))]
}

// named finds the procedures whose name appears in the route (exact, then tolerant).
func named(all []proc, toks []string) []proc {
	for pass := 0; pass < 2; pass++ {
		for _, t := range toks {
			var m []proc
			for _, p := range all {
				if (pass == 0 && p.name == t) || (pass == 1 && sameProc(p.name, t)) {
					m = append(m, p)
				}
			}
			if len(m) > 0 {
				return m
			}
		}
	}
	return nil
}

// departure returns the SID named in the route, or a random SID ending at the first en-route fix.
func (n *navdata) departure(dep string, toks, fixes []string, key string) []Point {
	all := n.procsFor(dep).sids
	if len(all) == 0 {
		return nil
	}
	cands := named(all, toks)
	if len(cands) == 0 && len(fixes) > 0 {
		for _, p := range all {
			for i := len(p.names) - 1; i >= 0 && i >= len(p.names)-2; i-- {
				if p.names[i] == fixes[0] {
					cands = append(cands, p)
					break
				}
			}
		}
	}
	if len(cands) == 0 {
		return nil
	}
	p := choose(cands, key)
	log.Printf("geo: SID %s at %s (runways %s), %d candidate(s)", p.name, dep, p.runways, len(cands))
	return p.pts
}

// arrival returns the STAR named in the route, or a random STAR starting at the last en-route fix.
func (n *navdata) arrival(arr string, toks, fixes []string, key string) []Point {
	all := n.procsFor(arr).stars
	if len(all) == 0 {
		return nil
	}
	cands := named(all, toks)
	if len(cands) == 0 && len(fixes) > 0 {
		last := fixes[len(fixes)-1]
		for _, p := range all {
			for i := 0; i < len(p.names) && i < 2; i++ {
				if p.names[i] == last {
					cands = append(cands, p)
					break
				}
			}
		}
	}
	if len(cands) == 0 {
		return nil
	}
	p := choose(cands, key)
	log.Printf("geo: STAR %s at %s (runways %s), %d candidate(s)", p.name, arr, p.runways, len(cands))
	return p.pts
}
