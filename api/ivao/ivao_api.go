package ivao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type JSONstruct struct {
	UpdatedAt    string            `json:"updatedAt"`
	Servers      []ServerInfo      `json:"servers"`
	VoiceServers []VoiceServerInfo `json:"voiceServers"`
	Connections  Connections       `json:"connections"`
}

type Connections struct {
	Pilots    []PilotInfo    `json:"pilots"`
	Atcs      []AtcInfo      `json:"atcs"`
	Observers []ObserverInfo `json:"observers"`
	FollowMe  []FollowMeInfo `json:"-"`
}

type ServerInfo struct {
	Id               string `json:"id"`
	Hostname         string `json:"hostname"`
	Ip               string `json:"ip"`
	Description      string `json:"description"`
	ConnectedClients int    `json:"connectedClients"`
}

type VoiceServerInfo struct {
	Id          string `json:"id"`
	Hostname    string `json:"hostname"`
	Ip          string `json:"ip"`
	Description string `json:"description"`
}

type PilotInfo struct {
	Id              int         `json:"id"`
	UserId          int         `json:"userId"`
	Callsign        string      `json:"callsign"`
	ServerId        string      `json:"serverId"`
	SoftwareTypeId  string      `json:"softwareTypeId"`
	SoftwareVersion string      `json:"softwareVersion"`
	Rating          int         `json:"rating"`
	CreatedAt       string      `json:"createdAt"`
	LastTrack       PilotTrack  `json:"lastTrack"`
	FlightPlan      *FlightPlan `json:"flightPlan"`
}

type PilotTrack struct {
	Altitude           int     `json:"altitude"`
	AltitudeDifference int     `json:"altitudeDifference"`
	ArrivalDistance    float64 `json:"arrivalDistance"`
	DepartureDistance  float64 `json:"departureDistance"`
	Groundspeed        int     `json:"groundspeed"`
	Heading            int     `json:"heading"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	OnGround           bool    `json:"onGround"`
	State              string  `json:"state"`
	Timestamp          string  `json:"timestamp"`
	Transponder        string  `json:"transponder"`
	TransponderMode    string  `json:"transponderMode"`
}

type FlightPlan struct {
	Id             int    `json:"id"`
	Revision       int    `json:"revision"`
	AircraftId     string `json:"aircraftId"`
	AircraftNumber int    `json:"aircraftNumber"`
	DepartureId    string `json:"departureId"`
	ArrivalId      string `json:"arrivalId"`
	AlternativeId  string `json:"alternativeId"`
	AltAltId       string `json:"altAltId"`
	DepartureTime  int    `json:"departureTime"`
	Eet            int    `json:"eet"`
	Endurance      int    `json:"endurance"`
	CruisingSpeed  string `json:"cruisingSpeed"`
	CruisingLevel  string `json:"cruisingLevel"`
	FlightRules    string `json:"flightRules"`
	FlightType     string `json:"flightType"`
	Route          string `json:"route"`
	Remarks        string `json:"remarks"`
}

type AtcInfo struct {
	Id              int        `json:"id"`
	UserId          int        `json:"userId"`
	Callsign        string     `json:"callsign"`
	ServerId        string     `json:"serverId"`
	SoftwareTypeId  string     `json:"softwareTypeId"`
	SoftwareVersion string     `json:"softwareVersion"`
	Rating          int        `json:"rating"`
	CreatedAt       string     `json:"createdAt"`
	LastTrack       AtcTrack   `json:"lastTrack"`
	AtcSession      AtcSession `json:"atcSession"`
	Atis            *AtisInfo  `json:"atis"`
}

type AtcTrack struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp string  `json:"timestamp"`
}

type AtcSession struct {
	Frequency float64 `json:"frequency"`
	Position  string  `json:"position"`
}

type AtisInfo struct {
	Lines     []string `json:"lines"`
	Revision  string   `json:"revision"`
	Timestamp string   `json:"timestamp"`
}

type ObserverInfo struct {
	Id              int    `json:"id"`
	UserId          int    `json:"userId"`
	Callsign        string `json:"callsign"`
	ServerId        string `json:"serverId"`
	SoftwareTypeId  string `json:"softwareTypeId"`
	SoftwareVersion string `json:"softwareVersion"`
	Rating          int    `json:"rating"`
	CreatedAt       string `json:"createdAt"`
}

type FollowMeInfo struct {
	Id              int    `json:"id"`
	UserId          int    `json:"userId"`
	Callsign        string `json:"callsign"`
	ServerId        string `json:"serverId"`
	SoftwareTypeId  string `json:"softwareTypeId"`
	SoftwareVersion string `json:"softwareVersion"`
	Rating          int    `json:"rating"`
	CreatedAt       string `json:"createdAt"`
}

var (
	mu sync.RWMutex

	pilotByCallsign map[string]PilotInfo
	pilotByUserId   map[string]PilotInfo

	atcByCallsign map[string]AtcInfo
	atcByUserId   map[string]AtcInfo

	observerByCallsign map[string]ObserverInfo
	followMeByCallsign map[string]FollowMeInfo

	serverById      map[string]ServerInfo
	voiceServerById map[string]VoiceServerInfo
)

func StartIvaoApi() {
	fetchAndBuild()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fetchAndBuild()
		}
	}()
}

func fetchAndBuild() {
	const url = "https://api.ivao.aero/v2/tracker/whazzup"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("HTTP error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	var data JSONstruct
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Unmarshal error:", err)
		return
	}

	newPilotByCallsign := make(map[string]PilotInfo, len(data.Connections.Pilots))
	newPilotByUserId := make(map[string]PilotInfo, len(data.Connections.Pilots))
	for _, p := range data.Connections.Pilots {
		newPilotByCallsign[p.Callsign] = p
		newPilotByUserId[fmt.Sprintf("%d", p.UserId)] = p
	}

	newAtcByCallsign := make(map[string]AtcInfo, len(data.Connections.Atcs))
	newAtcByUserId := make(map[string]AtcInfo, len(data.Connections.Atcs))
	for _, a := range data.Connections.Atcs {
		newAtcByCallsign[a.Callsign] = a
		newAtcByUserId[fmt.Sprintf("%d", a.UserId)] = a
	}

	newObserverByCallsign := make(map[string]ObserverInfo, len(data.Connections.Observers))
	for _, o := range data.Connections.Observers {
		newObserverByCallsign[o.Callsign] = o
	}

	newFollowMeByCallsign := make(map[string]FollowMeInfo, len(data.Connections.FollowMe))
	for _, f := range data.Connections.FollowMe {
		newFollowMeByCallsign[f.Callsign] = f
	}

	newServerById := make(map[string]ServerInfo, len(data.Servers))
	for _, s := range data.Servers {
		newServerById[s.Id] = s
	}

	newVoiceServerById := make(map[string]VoiceServerInfo, len(data.VoiceServers))
	for _, v := range data.VoiceServers {
		newVoiceServerById[v.Id] = v
	}

	mu.Lock()
	pilotByCallsign = newPilotByCallsign
	pilotByUserId = newPilotByUserId
	atcByCallsign = newAtcByCallsign
	atcByUserId = newAtcByUserId
	observerByCallsign = newObserverByCallsign
	followMeByCallsign = newFollowMeByCallsign
	serverById = newServerById
	voiceServerById = newVoiceServerById
	mu.Unlock()
}

// Pilots
func GetPilotFromCallsign(callsign string) (PilotInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := pilotByCallsign[callsign]; ok {
		return p, nil
	}
	return PilotInfo{}, errors.New("no pilot found with this callsign")
}

func GetPilotFromUserId(userId string) (PilotInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := pilotByUserId[userId]; ok {
		return p, nil
	}
	return PilotInfo{}, errors.New("no pilot found with this userId")
}

// ATC
func GetAtcFromCallsign(callsign string) (AtcInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if a, ok := atcByCallsign[callsign]; ok {
		return a, nil
	}
	return AtcInfo{}, errors.New("no ATC found with this callsign")
}

func GetAtcFromUserId(userId string) (AtcInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if a, ok := atcByUserId[userId]; ok {
		return a, nil
	}
	return AtcInfo{}, errors.New("no ATC found with this userId")
}

// Observers
func GetObserverFromCallsign(callsign string) (ObserverInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if o, ok := observerByCallsign[callsign]; ok {
		return o, nil
	}
	return ObserverInfo{}, errors.New("no observer found with this callsign")
}

// FollowMe vehicles
func GetFollowMeFromCallsign(callsign string) (FollowMeInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := followMeByCallsign[callsign]; ok {
		return f, nil
	}
	return FollowMeInfo{}, errors.New("no follow-me vehicle found with this callsign")
}

// Servers

func GetServerFromId(id string) (ServerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := serverById[id]; ok {
		return s, nil
	}
	return ServerInfo{}, errors.New("no server found with this id")
}

func GetVoiceServerFromId(id string) (VoiceServerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if v, ok := voiceServerById[id]; ok {
		return v, nil
	}
	return VoiceServerInfo{}, errors.New("no voice server found with this id")
}
