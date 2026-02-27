package vatsim

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	RevisionId          string `json:"revision_id"`
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

func getUrl(endpoint string, parameters []UrlArgument) string {
	// Get the full url whith arguments for the endpoint
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

// Functions for Searshing the specific instance in eatch data type
func GetPilotFromCallsign(Callsign string) (PilotInfo, error) {
	// Search the Pilots struct dor the right callsign
	Json := GetVatsimInfo()
	for _, Pilot := range Json.Pilots {
		if Pilot.Callsign == Callsign {
			return Pilot, nil
		}
	}
	return PilotInfo{}, errors.New("No pilot found with this callsign")
}
func GetPilotFromId(Cid string) (PilotInfo, error) {
	// Search the Pilots struct dor the right Cid
	Json := GetVatsimInfo()
	for _, Pilot := range Json.Pilots {
		if fmt.Sprintf("%d", Pilot.Cid) == Cid {
			return Pilot, nil
		}
	}
	return PilotInfo{}, errors.New("No pilot found with this Cid")
}
func GetControllerFromCallsign(Callsign string) (ControllerInfo, error) {
	// Search the Controllers struct dor the right callsign
	Json := GetVatsimInfo()
	for _, Controller := range Json.Controllers {
		if Controller.Callsign == Callsign {
			return Controller, nil
		}
	}
	return ControllerInfo{}, errors.New("No Controller found with this callsign")
}
func GetControllerFromId(Cid string) (ControllerInfo, error) {
	// Search the Controllers struct dor the right Cid
	Json := GetVatsimInfo()
	for _, Controller := range Json.Controllers {
		if fmt.Sprintf("%d", Controller.Cid) == Cid {
			return Controller, nil
		}
	}
	return ControllerInfo{}, errors.New("No Controller found with this Cid")
}
func GetAtisFromCallsign(Callsign string) (AtisInfo, error) {
	Json := GetVatsimInfo()
	for _, AnAtis := range Json.Atis {
		if AnAtis.Callsign == Callsign {
			return AnAtis, nil
		}
	}
	return AtisInfo{}, errors.New("No Atis found with this callsign")
}
func GetAtisFromId(Cid string) (AtisInfo, error) {
	Json := GetVatsimInfo()
	for _, AnAtis := range Json.Atis {
		if fmt.Sprintf("%d", AnAtis.Cid) == Cid {
			return AnAtis, nil
		}
	}
	return AtisInfo{}, errors.New("No Atis found with this Cid")
}
func GetServerFromIdent(Ident string) (ServerInfo, error) {
	Json := GetVatsimInfo()
	for _, Server := range Json.Servers {
		if Server.Ident == Ident {
			return Server, nil
		}
	}
	return ServerInfo{}, errors.New("No Server found with this Cid")
}
func GetPrefileFromCallsign(Callsign string) (PrefileInfo, error) {
	Json := GetVatsimInfo()
	for _, Prefile := range Json.Prefiles {
		if Prefile.Callsign == Callsign {
			return Prefile, nil
		}
	}
	return PrefileInfo{}, errors.New("No Prefile found with this callsign")
}
func GetPrefileFromId(Cid string) (PrefileInfo, error) {
	Json := GetVatsimInfo()
	for _, Prefile := range Json.Prefiles {
		if fmt.Sprintf("%d", Prefile.Cid) == Cid {
			return Prefile, nil
		}
	}
	return PrefileInfo{}, errors.New("No Prefile found with this Cid")
}
func GetFacilityFromId(Id string) (FacilityInfo, error) {
	Json := GetVatsimInfo()
	for _, Facility := range Json.Facilities {
		if fmt.Sprintf("%d", Facility.Id) == Id {
			return Facility, nil
		}
	}
	return FacilityInfo{}, errors.New("No Facility found with this Id")
}
func GetFacilityFromShortName(Name string) (FacilityInfo, error) {
	Json := GetVatsimInfo()
	for _, Facility := range Json.Facilities {
		if Facility.ShortName == Name {
			return Facility, nil
		}
	}
	return FacilityInfo{}, errors.New("No Facility found with this Short Name")
}
func GetRatingFromId(Id string) (RatingInfo, error) {
	Json := GetVatsimInfo()
	for _, Rating := range Json.Ratings {
		if fmt.Sprintf("%d", Rating.Id) == Id {
			return Rating, nil
		}
	}
	return RatingInfo{}, errors.New("No Rating found with this Id")
}
func GetRatingFromShortName(Name string) (RatingInfo, error) {
	Json := GetVatsimInfo()
	for _, Rating := range Json.Ratings {
		if Rating.ShortName == Name {
			return Rating, nil
		}
	}
	return RatingInfo{}, errors.New("No Rating found with this Short Name")
}
func GetPilotRatingFromId(Id string) (PilotRatingInfo, error) {
	Json := GetVatsimInfo()
	for _, PilotRating := range Json.PilotRatings {
		if fmt.Sprintf("%d", PilotRating.Id) == Id {
			return PilotRating, nil
		}
	}
	return PilotRatingInfo{}, errors.New("No PilotRating found with this Id")
}
func GetPilotRatingFromShortName(Name string) (PilotRatingInfo, error) {
	Json := GetVatsimInfo()
	for _, PilotRating := range Json.PilotRatings {
		if PilotRating.ShortName == Name {
			return PilotRating, nil
		}
	}
	return PilotRatingInfo{}, errors.New("No PilotRating found with this Short Name")
}
func GetMilitaryRatingFromId(Id string) (MilitaryRatingInfo, error) {
	Json := GetVatsimInfo()
	for _, MilitaryRating := range Json.MilitaryRatings {
		if fmt.Sprintf("%d", MilitaryRating.Id) == Id {
			return MilitaryRating, nil
		}
	}
	return MilitaryRatingInfo{}, errors.New("No PilotRating found with this Id")
}
func GetMilitaryRatingFromShortName(Name string) (MilitaryRatingInfo, error) {
	Json := GetVatsimInfo()
	for _, MilitaryRating := range Json.MilitaryRatings {
		if MilitaryRating.ShortName == Name {
			return MilitaryRating, nil
		}
	}
	return MilitaryRatingInfo{}, errors.New("No MilitaryRating found with this Short Name")
}

func GetVatsimInfo() JSONstruct {
	// Request the Json from the vatsm live Data api
	url := getUrl("https://data.vatsim.net/v3/vatsim-data.json", []UrlArgument{})

	response, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	result := JSONstruct{}
	json.Unmarshal(body, &result)
	return result
}
