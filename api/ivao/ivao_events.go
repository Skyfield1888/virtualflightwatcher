package ivao

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type EventInfo struct {
	Id          string
	Title       string
	Description string
	Link        string
	Airports    []string
	Start, End  time.Time
}

// FetchEvents is deliberately tolerant about the JSON shape (array or {items|data|events: [...]}).
// If the endpoint needs authentication, set IVAO_API_KEY in .env.
func FetchEvents() ([]EventInfo, error) {
	req, err := http.NewRequest("GET", "https://api.ivao.aero/v2/events", nil)
	if err != nil {
		return nil, err
	}
	if key := os.Getenv("IVAO_API_KEY"); key != "" {
		req.Header.Set("apiKey", key)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ivao events: unexpected status %s (IVAO_API_KEY needed?)", resp.Status)
	}

	var raw any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var list []any
	switch v := raw.(type) {
	case []any:
		list = v
	case map[string]any:
		for _, k := range []string{"items", "data", "events"} {
			if a, ok := v[k].([]any); ok {
				list = a
				break
			}
		}
	}

	var out []EventInfo
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		e := EventInfo{
			Id:          pickStr(m, "id", "eventId"),
			Title:       pickStr(m, "title", "name"),
			Description: pickStr(m, "description", "shortDescription"),
			Link:        pickStr(m, "url", "link"),
			Start:       pickTime(m, "startDate", "startTime", "start", "beginsAt"),
			End:         pickTime(m, "endDate", "endTime", "end", "endsAt"),
		}
		if arr, ok := m["airports"].([]any); ok {
			for _, a := range arr {
				switch x := a.(type) {
				case string:
					e.Airports = append(e.Airports, x)
				case map[string]any:
					if s := pickStr(x, "icao", "airportId", "id"); s != "" {
						e.Airports = append(e.Airports, s)
					}
				}
			}
		}
		if e.Id == "" || e.Title == "" || e.Start.IsZero() {
			continue
		}
		if e.Link == "" {
			e.Link = "https://events.ivao.aero"
		}
		out = append(out, e)
	}
	return out, nil
}

func pickStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if v != "" {
				return v
			}
		case float64:
			return fmt.Sprintf("%.0f", v)
		}
	}
	return ""
}

func pickTime(m map[string]any, keys ...string) time.Time {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
