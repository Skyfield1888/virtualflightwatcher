package vatsim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EventInfo struct {
	Id         int    `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Link       string `json:"link"`
	Organisers []struct {
		Region           string `json:"region"`
		Division         string `json:"division"`
		Subdivision      string `json:"subdivision"`
		OrganisationCode string `json:"organisation_code"`
	} `json:"organisers"`
	Airports []struct {
		Icao string `json:"icao"`
	} `json:"airports"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	ShortDescription string    `json:"short_description"`
	Description      string    `json:"description"`
	Banner           string    `json:"banner"`
}

// FetchEvents returns the upcoming / running VATSIM events.
func FetchEvents() ([]EventInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://my.vatsim.net/api/v1/events/all")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vatsim events: unexpected status %s", resp.Status)
	}
	var out struct {
		Data []EventInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}