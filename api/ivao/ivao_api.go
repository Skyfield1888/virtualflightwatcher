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
	UpdatedAt    string       `json:"updatedAt"`
	Clients      Clients      `json:"clients"`
	Servers      []ServerInfo `json:"servers"`
	VoiceServers []ServerInfo `json:"voiceServers"`
	Connections  NetworkStats `json:"connections"`
}

type Clients struct {
	Pilots    []PilotInfo    `json:"pilots"`
	Atcs      []AtcInfo      `json:"atcs"`
	Observers []ObserverInfo `json:"observers"`
	FollowMe  FlexFollowMe   `json:"followMe"`
}

type NetworkStats struct {
	Total          int `json:"total"`
	Supervisor     int `json:"supervisor"`
	Atc            int `json:"atc"`
	Observer       int `json:"observer"`
	Pilot          int `json:"pilot"`
	WorldTour      int `json:"worldTour"`
	FollowMe       int `json:"followMe"`
	UniqueUsers24h int `json:"uniqueUsers24h"`
}

type ServerInfo struct {
	Id                 string `json:"id"`
	Hostname           string `json:"hostname"`
	Ip                 string `json:"ip"`
	Description        string `json:"description"`
	CountryId          string `json:"countryId"`
	CurrentConnections int    `json:"currentConnections"`
	MaximumConnections int    `json:"maximumConnections"`
}

type UserRating struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
}

type UserRatings struct {
	IsAtc         bool       `json:"isAtc"`
	IsPilot       bool       `json:"isPilot"`
	PilotRating   UserRating `json:"pilotRating"`
	AtcRating     UserRating `json:"atcRating"`
	NetworkRating UserRating `json:"networkRating"`
}

type User struct {
	Id         int         `json:"id"`
	FirstName  string      `json:"firstName"`
	LastName   string      `json:"lastName"`
	DivisionId string      `json:"divisionId"`
	Rating     UserRatings `json:"rating"`
}

type BaseClient struct {
	Id              int    `json:"id"`
	Callsign        string `json:"callsign"`
	UserId          int    `json:"userId"`
	ConnectionType  string `json:"connectionType"`
	ServerId        string `json:"serverId"`
	Time            int    `json:"time"`
	SoftwareTypeId  string `json:"softwareTypeId"`
	SoftwareVersion string `json:"softwareVersion"`
	Sandbagging     bool   `json:"sandbagging"`
	IsMilitary      bool   `json:"isMilitary"`
	IsWorldTour     bool   `json:"isWorldTour"`
	CreatedAt       string `json:"createdAt"`
	CompletedAt     string `json:"completedAt"`
	UpdatedAt       string `json:"updatedAt"`
	User            User   `json:"user"`
}

type PilotInfo struct {
	BaseClient
	LastTrack  PilotTrack  `json:"lastTrack"`
	FlightPlan *FlightPlan `json:"flightPlan"`
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
	Transponder        int     `json:"transponder"`
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
	Alternative2Id string `json:"alternative2Id"`
	DepartureTime  int    `json:"departureTime"`
	Eet            int    `json:"eet"`
	Endurance      int    `json:"endurance"`
	Speed          string `json:"speed"`
	Level          string `json:"level"`
	FlightRules    string `json:"flightRules"`
	FlightType     string `json:"flightType"`
	Route          string `json:"route"`
	Remarks        string `json:"remarks"`
	PeopleOnBoard  int    `json:"peopleOnBoard"`
}

type AtcInfo struct {
	BaseClient
	LastTrack  AtcTrack   `json:"lastTrack"`
	AtcSession AtcSession `json:"atcSession"`
	Atis       *AtisInfo  `json:"atis"`
}

type AtcTrack struct {
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
	Transponder        int     `json:"transponder"`
	TransponderMode    string  `json:"transponderMode"`
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
	BaseClient
	AtcSession AtcSession `json:"atcSession"`
}

type FollowMeInfo struct {
	BaseClient
}

type FlexFollowMe []FollowMeInfo

func (f *FlexFollowMe) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] != '[' {
		return nil
	}
	return json.Unmarshal(data, (*[]FollowMeInfo)(f))
}

var (
	mu sync.RWMutex

	PilotByCallsign map[string]PilotInfo
	pilotByUserId   map[string]PilotInfo

	atcByCallsign map[string]AtcInfo
	atcByUserId   map[string]AtcInfo

	observerByCallsign map[string]ObserverInfo
	followMeByCallsign map[string]FollowMeInfo

	serverById      map[string]ServerInfo
	voiceServerById map[string]ServerInfo
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

	newPilotByCallsign := make(map[string]PilotInfo, len(data.Clients.Pilots))
	newPilotByUserId := make(map[string]PilotInfo, len(data.Clients.Pilots))
	for _, p := range data.Clients.Pilots {
		newPilotByCallsign[p.Callsign] = p
		newPilotByUserId[fmt.Sprintf("%d", p.UserId)] = p
	}

	newAtcByCallsign := make(map[string]AtcInfo, len(data.Clients.Atcs))
	newAtcByUserId := make(map[string]AtcInfo, len(data.Clients.Atcs))
	for _, a := range data.Clients.Atcs {
		newAtcByCallsign[a.Callsign] = a
		newAtcByUserId[fmt.Sprintf("%d", a.UserId)] = a
	}

	newObserverByCallsign := make(map[string]ObserverInfo, len(data.Clients.Observers))
	for _, o := range data.Clients.Observers {
		newObserverByCallsign[o.Callsign] = o
	}

	newFollowMeByCallsign := make(map[string]FollowMeInfo, len(data.Clients.FollowMe))
	for _, f := range data.Clients.FollowMe {
		newFollowMeByCallsign[f.Callsign] = f
	}

	newServerById := make(map[string]ServerInfo, len(data.Servers))
	for _, s := range data.Servers {
		newServerById[s.Id] = s
	}

	newVoiceServerById := make(map[string]ServerInfo, len(data.VoiceServers))
	for _, v := range data.VoiceServers {
		newVoiceServerById[v.Id] = v
	}

	mu.Lock()
	PilotByCallsign = newPilotByCallsign
	pilotByUserId = newPilotByUserId
	atcByCallsign = newAtcByCallsign
	atcByUserId = newAtcByUserId
	observerByCallsign = newObserverByCallsign
	followMeByCallsign = newFollowMeByCallsign
	serverById = newServerById
	voiceServerById = newVoiceServerById
	mu.Unlock()
}

func GetPilotFromCallsign(callsign string) (PilotInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := PilotByCallsign[callsign]; ok {
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

func GetObserverFromCallsign(callsign string) (ObserverInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if o, ok := observerByCallsign[callsign]; ok {
		return o, nil
	}
	return ObserverInfo{}, errors.New("no observer found with this callsign")
}

func GetFollowMeFromCallsign(callsign string) (FollowMeInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := followMeByCallsign[callsign]; ok {
		return f, nil
	}
	return FollowMeInfo{}, errors.New("no follow-me vehicle found with this callsign")
}

func GetServerFromId(id string) (ServerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := serverById[id]; ok {
		return s, nil
	}
	return ServerInfo{}, errors.New("no server found with this id")
}

func GetVoiceServerFromId(id string) (ServerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if v, ok := voiceServerById[id]; ok {
		return v, nil
	}
	return ServerInfo{}, errors.New("no voice server found with this id")
}
