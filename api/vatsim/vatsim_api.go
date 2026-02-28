package vatsim

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Declarations of the Struct for the url arguments and Json
type UrlArgument struct {
	ParameterName string
	Value         string
}

type JSONstruct struct {
	General         GeneralInfo
	Pilots          []PilotInfo
	Controllers     []ControllerInfo
	Atis            []AtisInfo
	Servers         []ServerInfo
	Prefiles        []PrefileInfo
	Facilities      []FacilityInfo
	Ratings         []RatingInfo
	PilotRatings    []PilotRatingInfo    `json:"pilot_ratings"`
	MilitaryRatings []MilitaryRatingInfo `json:"military_ratings"`
}

type GeneralInfo struct {
	Version          int
	UpdateTimestamp  string `json:"update_timestamp"`
	ConnectedClients int    `json:"connected_clients"`
	UniqueUsers      int    `json:"unique_users"`
}

type PilotInfo struct {
	Cid            int
	Name           string
	Callsign       string
	Server         string
	PilotRating    int `json:"pilot_rating"`
	MilitaryRating int `json:"military_rating"`
	Latitude       float64
	Longitude      float64
	Altitude       int
	Groundspeed    int
	Transponder    string
	Heading        int
	QnhInHg        float64        `json:"qnh_i_hg"`
	QnhInmB        int            `json:"qnh_mb"`
	FlightPlan     FlightPlanInfo `json:"flight_plan"`
	LogonTime      string         `json:"logon_time"`
	LastUpdated    string         `json:"last_updated"`
}

type FlightPlanInfo struct {
	FlightRules         string `json:"flight_rules"`
	Aircraft            string
	AircraftFaa         string `json:"aircraft_faa"`
	AircraftShort       string `json:"aircraft_short"`
	Departure           string
	Arrival             string
	Alternate           string
	Deptime             string
	EnrouteTime         string `json:"enroute_time"`
	FuelTime            string `json:"fuel_time"`
	Remarks             string
	Route               string
	RevisionId          int    `json:"revision_id"`
	AssignedTransponder string `json:"assigned_transponder"`
}

type ControllerInfo struct {
	Cid          int
	Name         string
	Callsign     string
	Frequency    string
	Facility     int
	Rating       int
	Server       string
	VisualRange  int      `json:"visual_range"`
	TextPosition []string `json:"text_atis"`
	LastUpdated  string   `json:"last_updated"`
	LogonTime    string   `json:"logon_time"`
}

type AtisInfo struct {
	Cid         int
	Name        string
	Callsign    string
	Frequency   string
	Facility    int
	Rating      int
	Server      string
	VisualRange int      `json:"visual_range"`
	AtisCode    string   `json:"atis_code"`
	TextAtis    []string `json:"text_atis"`
	LastUpdated string   `json:"last_updated"`
	LogonTime   string   `json:"logon_time"`
}

type ServerInfo struct {
	Ident                    string
	HostnameOrIp             string `json:"hostname_or_ip"`
	Location                 string
	Name                     string
	ClientConnectionsAllowed bool `json:"client_connections_allowed"`
	IsSweatbox               bool `json:"is_sweatbox"`
}

type PrefileInfo struct {
	Cid         int
	Name        string
	Callsign    string
	FlightPlan  FlightPlanInfo `json:"flight_plan"`
	LastUpdated string         `json:"last_updated"`
}

type FacilityInfo struct {
	Id        int
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

type RatingInfo struct {
	Id        int
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

type PilotRatingInfo struct {
	Id        int
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

type MilitaryRatingInfo struct {
	Id        int
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

// Maps + RWMutex (multiple readers, one writer)
var (
	mu sync.RWMutex

	pilotByCallsign map[string]PilotInfo
	pilotByCid      map[string]PilotInfo

	controllerByCallsign map[string]ControllerInfo
	controllerByCid      map[string]ControllerInfo

	atisByCallsign map[string]AtisInfo
	atisByCid      map[string]AtisInfo

	serverByIdent map[string]ServerInfo

	PrefileByCallsign map[string]PrefileInfo
	prefileByCid      map[string]PrefileInfo

	facilityById        map[string]FacilityInfo
	facilityByShortName map[string]FacilityInfo

	ratingById        map[string]RatingInfo
	ratingByShortName map[string]RatingInfo

	pilotRatingById        map[string]PilotRatingInfo
	pilotRatingByShortName map[string]PilotRatingInfo

	militaryRatingById        map[string]MilitaryRatingInfo
	militaryRatingByShortName map[string]MilitaryRatingInfo
)

// call on startup and evry 15sec
func StartVatsimApi() {
	fetchAndBuild()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fetchAndBuild()
		}
	}()
}

func getUrl(endpoint string, parameters []UrlArgument) string {
	if len(parameters) == 0 {
		return endpoint
	}
	endpoint += "?"
	for i, parameter := range parameters {
		endpoint += parameter.ParameterName + "=" + parameter.Value
		if i != 0 && i != len(parameters) {
			endpoint += "&"
		}
	}
	return endpoint
}

func fetchAndBuild() {
	url := getUrl("https://data.vatsim.net/v3/vatsim-data.json", []UrlArgument{})
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("HTTP error:", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	var data JSONstruct
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Unmarshal error:", err)
		return
	}

	newPilotByCallsign := make(map[string]PilotInfo, len(data.Pilots))
	newPilotByCid := make(map[string]PilotInfo, len(data.Pilots))
	for _, p := range data.Pilots {
		newPilotByCallsign[p.Callsign] = p
		newPilotByCid[fmt.Sprintf("%d", p.Cid)] = p
	}

	newControllerByCallsign := make(map[string]ControllerInfo, len(data.Controllers))
	newControllerByCid := make(map[string]ControllerInfo, len(data.Controllers))
	for _, c := range data.Controllers {
		newControllerByCallsign[c.Callsign] = c
		newControllerByCid[fmt.Sprintf("%d", c.Cid)] = c
	}

	newAtisByCallsign := make(map[string]AtisInfo, len(data.Atis))
	newAtisByCid := make(map[string]AtisInfo, len(data.Atis))
	for _, a := range data.Atis {
		newAtisByCallsign[a.Callsign] = a
		newAtisByCid[fmt.Sprintf("%d", a.Cid)] = a
	}

	newServerByIdent := make(map[string]ServerInfo, len(data.Servers))
	for _, s := range data.Servers {
		newServerByIdent[s.Ident] = s
	}

	newPrefileByCallsign := make(map[string]PrefileInfo, len(data.Prefiles))
	newPrefileByCid := make(map[string]PrefileInfo, len(data.Prefiles))
	for _, p := range data.Prefiles {
		newPrefileByCallsign[p.Callsign] = p
		newPrefileByCid[fmt.Sprintf("%d", p.Cid)] = p
	}

	newFacilityById := make(map[string]FacilityInfo, len(data.Facilities))
	newFacilityByShortName := make(map[string]FacilityInfo, len(data.Facilities))
	for _, f := range data.Facilities {
		newFacilityById[fmt.Sprintf("%d", f.Id)] = f
		newFacilityByShortName[f.ShortName] = f
	}

	newRatingById := make(map[string]RatingInfo, len(data.Ratings))
	newRatingByShortName := make(map[string]RatingInfo, len(data.Ratings))
	for _, r := range data.Ratings {
		newRatingById[fmt.Sprintf("%d", r.Id)] = r
		newRatingByShortName[r.ShortName] = r
	}

	newPilotRatingById := make(map[string]PilotRatingInfo, len(data.PilotRatings))
	newPilotRatingByShortName := make(map[string]PilotRatingInfo, len(data.PilotRatings))
	for _, pr := range data.PilotRatings {
		newPilotRatingById[fmt.Sprintf("%d", pr.Id)] = pr
		newPilotRatingByShortName[pr.ShortName] = pr
	}

	newMilitaryRatingById := make(map[string]MilitaryRatingInfo, len(data.MilitaryRatings))
	newMilitaryRatingByShortName := make(map[string]MilitaryRatingInfo, len(data.MilitaryRatings))
	for _, mr := range data.MilitaryRatings {
		newMilitaryRatingById[fmt.Sprintf("%d", mr.Id)] = mr
		newMilitaryRatingByShortName[mr.ShortName] = mr
	}

	// Swap all maps at once under write lock (very fast)
	mu.Lock()
	pilotByCallsign = newPilotByCallsign
	pilotByCid = newPilotByCid
	controllerByCallsign = newControllerByCallsign
	controllerByCid = newControllerByCid
	atisByCallsign = newAtisByCallsign
	atisByCid = newAtisByCid
	serverByIdent = newServerByIdent
	PrefileByCallsign = newPrefileByCallsign
	prefileByCid = newPrefileByCid
	facilityById = newFacilityById
	facilityByShortName = newFacilityByShortName
	ratingById = newRatingById
	ratingByShortName = newRatingByShortName
	pilotRatingById = newPilotRatingById
	pilotRatingByShortName = newPilotRatingByShortName
	militaryRatingById = newMilitaryRatingById
	militaryRatingByShortName = newMilitaryRatingByShortName
	mu.Unlock()
}

// Pilots
func GetPilotFromCallsign(Callsign string) (PilotInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := pilotByCallsign[Callsign]; ok {
		return p, nil
	}
	return PilotInfo{}, errors.New("No pilot found with this callsign")
}

func GetPilotFromId(Cid string) (PilotInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := pilotByCid[Cid]; ok {
		return p, nil
	}
	return PilotInfo{}, errors.New("No pilot found with this Cid")
}

// Controllers
func GetControllerFromCallsign(Callsign string) (ControllerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if c, ok := controllerByCallsign[Callsign]; ok {
		return c, nil
	}
	return ControllerInfo{}, errors.New("No Controller found with this callsign")
}

func GetControllerFromId(Cid string) (ControllerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if c, ok := controllerByCid[Cid]; ok {
		return c, nil
	}
	return ControllerInfo{}, errors.New("No Controller found with this Cid")
}

// Atis
func GetAtisFromCallsign(Callsign string) (AtisInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if a, ok := atisByCallsign[Callsign]; ok {
		return a, nil
	}
	return AtisInfo{}, errors.New("No Atis found with this callsign")
}

func GetAtisFromId(Cid string) (AtisInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if a, ok := atisByCid[Cid]; ok {
		return a, nil
	}
	return AtisInfo{}, errors.New("No Atis found with this Cid")
}

// Servers
func GetServerFromIdent(Ident string) (ServerInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := serverByIdent[Ident]; ok {
		return s, nil
	}
	return ServerInfo{}, errors.New("No Server found with this Ident")
}

// Prefiles
func GetPrefileFromCallsign(Callsign string) (PrefileInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := PrefileByCallsign[Callsign]; ok {
		return p, nil
	}
	return PrefileInfo{}, errors.New("No Prefile found with this callsign")
}

func GetPrefileFromId(Cid string) (PrefileInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := prefileByCid[Cid]; ok {
		return p, nil
	}
	return PrefileInfo{}, errors.New("No Prefile found with this Cid")
}

// Facilities
func GetFacilityFromId(Id string) (FacilityInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := facilityById[Id]; ok {
		return f, nil
	}
	return FacilityInfo{}, errors.New("No Facility found with this Id")
}

func GetFacilityFromShortName(Name string) (FacilityInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := facilityByShortName[Name]; ok {
		return f, nil
	}
	return FacilityInfo{}, errors.New("No Facility found with this Short Name")
}

// Ratings
func GetRatingFromId(Id string) (RatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if r, ok := ratingById[Id]; ok {
		return r, nil
	}
	return RatingInfo{}, errors.New("No Rating found with this Id")
}

func GetRatingFromShortName(Name string) (RatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if r, ok := ratingByShortName[Name]; ok {
		return r, nil
	}
	return RatingInfo{}, errors.New("No Rating found with this Short Name")
}

// PilotRatings
func GetPilotRatingFromId(Id string) (PilotRatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if pr, ok := pilotRatingById[Id]; ok {
		return pr, nil
	}
	return PilotRatingInfo{}, errors.New("No PilotRating found with this Id")
}

func GetPilotRatingFromShortName(Name string) (PilotRatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if pr, ok := pilotRatingByShortName[Name]; ok {
		return pr, nil
	}
	return PilotRatingInfo{}, errors.New("No PilotRating found with this Short Name")
}

// MilitaryRatings
func GetMilitaryRatingFromId(Id string) (MilitaryRatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if mr, ok := militaryRatingById[Id]; ok {
		return mr, nil
	}
	return MilitaryRatingInfo{}, errors.New("No MilitaryRating found with this Id")
}

func GetMilitaryRatingFromShortName(Name string) (MilitaryRatingInfo, error) {
	mu.RLock()
	defer mu.RUnlock()
	if mr, ok := militaryRatingByShortName[Name]; ok {
		return mr, nil
	}
	return MilitaryRatingInfo{}, errors.New("No MilitaryRating found with this Short Name")
}
