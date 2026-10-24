package geo

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// QNH from the real-world METAR (aviationweather.gov, free, no key). IVAO's data feed carries
// no pressure, so the QNH shown is the one of the departure/arrival airport nearest to the aircraft.

var metarURL = "https://aviationweather.gov/api/data/metar"

type qnhEntry struct {
	hpa float64
	ok  bool
	at  time.Time
}

var (
	qnhMu    sync.Mutex
	qnhCache = map[string]qnhEntry{}
	qRe      = regexp.MustCompile(`\bQ(\d{4})\b`)
	aRe      = regexp.MustCompile(`\bA(\d{4})\b`)
)

func parseQNH(metar string) (float64, bool) {
	if m := qRe.FindStringSubmatch(metar); m != nil {
		v, _ := strconv.Atoi(m[1])
		return float64(v), true
	}
	if m := aRe.FindStringSubmatch(metar); m != nil { // altimeter in hundredths of inHg
		v, _ := strconv.Atoi(m[1])
		return float64(v) / 100 * 33.8639, true
	}
	return 0, false
}

// QNH returns the QNH in hPa for an airport. Results are cached (10 min, 2 min on failure) and
// the request is limited to 1.5 s so a slow API can't break Discord's 3 s response window.
func QNH(icao string) (float64, bool) {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	if icao == "" {
		return 0, false
	}
	qnhMu.Lock()
	e, found := qnhCache[icao]
	qnhMu.Unlock()
	if found {
		ttl := 10 * time.Minute
		if !e.ok {
			ttl = 2 * time.Minute
		}
		if time.Since(e.at) < ttl {
			return e.hpa, e.ok
		}
	}

	hpa, ok := fetchQNH(icao)
	qnhMu.Lock()
	qnhCache[icao] = qnhEntry{hpa, ok, time.Now()}
	qnhMu.Unlock()
	return hpa, ok
}

func fetchQNH(icao string) (float64, bool) {
	req, err := http.NewRequest("GET", metarURL+"?format=raw&ids="+url.QueryEscape(icao), nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("User-Agent", "VirtualFlightWatcher/1.0 (discord bot)")
	resp, err := (&http.Client{Timeout: 1500 * time.Millisecond}).Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return 0, false
	}
	return parseQNH(string(body))
}

// QNHNear returns the QNH of the departure or arrival airport closest to pos
// (falling back to the other one when the nearest has no METAR).
func QNHNear(depICAO, arrICAO string, pos Point) (hpa float64, icao string, ok bool) {
	type cand struct {
		icao string
		d    float64
	}
	var cs []cand
	for _, c := range []string{depICAO, arrICAO} {
		if p, found := Airport(c); found {
			cs = append(cs, cand{strings.ToUpper(c), DistanceNM(pos, p)})
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	for _, c := range cs {
		if v, good := QNH(c.icao); good {
			return v, c.icao, true
		}
	}
	return 0, "", false
}
