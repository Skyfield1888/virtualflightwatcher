package commands

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/Skyfield1888/virtualflightwatcher/api/geo"
	ivao "github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	vatsim "github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/Skyfield1888/virtualflightwatcher/config"
	"github.com/bwmarrin/discordgo"
)

func isPilotCallsign(s string) bool {
	return regexp.MustCompile(`^[A-Z]{3}[A-Z0-9]{1,4}$`).MatchString(s)
}

func isFacilityCallsign(s string) bool {
	return regexp.MustCompile(`(?i)^[A-Z]{2,7}(_+[A-Z0-9]{1,3})*_+(DEL|GND|TWR|APP|DEP|CTR|FSS|ATIS|OBS|AFIS|INFO|RADAR|OCA)$`).MatchString(s)
}

// normalizeCallsign trims spaces and uppercases, so "afr123" and " Afr123 " match "AFR123".
// CIDs (digits only) are not affected.
func normalizeCallsign(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

func formatFlightRules(code string) string {
	switch code {
	case "I":
		return "IFR"
	case "V":
		return "VFR"
	case "Y":
		return "IFR/VFR"
	case "Z":
		return "VFR/IFR"
	default:
		return code
	}
}

func formatDuration(hhmm string) string {
	if len(hhmm) < 3 {
		return hhmm
	}
	for len(hhmm) < 4 {
		hhmm = "0" + hhmm
	}
	return hhmm[:2] + "h " + hhmm[2:] + "m"
}

func formatFacility(facility int) string {
	switch facility {
	case 0:
		return "OBS"
	case 1:
		return "FSS"
	case 2:
		return "DEL"
	case 3:
		return "GND"
	case 4:
		return "TWR"
	case 5:
		return "APP"
	case 6:
		return "CTR"
	default:
		return "Unknown"
	}
}

func formatRating(rating int) string {
	switch rating {
	case 1:
		return "OBS"
	case 2:
		return "S1"
	case 3:
		return "S2"
	case 4:
		return "S3"
	case 5:
		return "C1"
	case 7:
		return "C3"
	case 8:
		return "I1"
	case 10:
		return "I3"
	case 11:
		return "SUP"
	case 12:
		return "ADM"
	default:
		return "Unknown"
	}
}

func toAtisCallsign(s string) string {
	s = strings.ToUpper(s)
	if strings.HasSuffix(s, "_ATIS") {
		return s
	}
	return s + "_ATIS"
}

// respond honours the "private replies" setting (send is in settings.go).
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	send(s, i, embed, SettingsOf(i).Ephemeral)
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	send(s, i, &discordgo.MessageEmbed{
		Title:       tr(SettingsOf(i))("error.title"),
		Description: message,
		Color:       0xFF0000,
	}, true)
}

func getOptions(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range i.ApplicationCommandData().Options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

func formatSeconds(s int) string { // IVAO durations (seconds)
	return fmt.Sprintf("%02dh %02dm", s/3600, (s%3600)/60)
}

func formatClock(s int) string { // IVAO time of day (seconds from 00:00 Z)
	s %= 86400
	return fmt.Sprintf("%02d:%02d Z", s/3600, (s%3600)/60)
}

func formatHHMMZ(hhmm string) string { // VATSIM "1215" -> "12:15 Z"
	for len(hhmm) < 4 {
		hhmm = "0" + hhmm
	}
	return hhmm[:2] + ":" + hhmm[2:] + " Z"
}

func yesNo(b bool, st config.Settings) string {
	if b {
		return tr(st)("yes")
	}
	return tr(st)("no")
}

func ivaoQnh(info ivao.PilotInfo, st config.Settings) string {
	if info.FlightPlan == nil {
		return "N/A"
	}
	hpa, icao, ok := geo.QNHNear(info.FlightPlan.DepartureId, info.FlightPlan.ArrivalId,
		geo.Point{Lat: info.LastTrack.Latitude, Lon: info.LastTrack.Longitude})
	if !ok {
		return "N/A"
	}
	if isMetric(st) {
		return fmt.Sprintf("%.0f hPa (%s)", hpa, icao)
	}
	if isImperial(st) {
		return fmt.Sprintf("%.2f inHg (%s)", hpa*0.02953, icao)
	}
	return fmt.Sprintf("%.0f hPa / %.2f inHg (%s)", hpa, hpa*0.02953, icao)
}

// orNA replaces empty values (pilots without a filed flight plan) by "N/A".
func orNA(s string) string {
	if strings.TrimSpace(s) == "" {
		return "N/A"
	}
	return s
}

func vatsimPilotEmbed(info vatsim.PilotInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	fp := info.FlightPlan

	title := info.Callsign
	if fp.Departure != "" || fp.Arrival != "" {
		title += "  •  " + orNA(fp.Departure) + " → " + orNA(fp.Arrival)
	}
	title += "  •  VATSIM"

	etd, enroute, fuel := "N/A", "N/A", "N/A"
	if fp.Deptime != "" {
		etd = formatHHMMZ(fp.Deptime)
	}
	if fp.EnrouteTime != "" {
		enroute = formatDuration(fp.EnrouteTime)
	}
	if fp.FuelTime != "" {
		fuel = formatDuration(fp.FuelTime)
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**%s** | %s | CID: `%d`", info.Name, orNA(fp.AircraftShort), info.Cid),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: t("f.route"), Value: fmt.Sprintf("```%s```", orNA(fp.Route)), Inline: false},
			{Name: t("f.position"), Value: fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.Latitude, info.Longitude), Inline: true},
			{Name: t("f.state"), Value: fmt.Sprintf("**Alt:** %s\n**GS:** %s\n**HDG:** %d°", fmtAltitude(info.Altitude, st), fmtSpeed(info.Groundspeed, st), info.Heading), Inline: true},
			{Name: t("f.weather"), Value: fmt.Sprintf("%s\n**Squawk:** %s", fmtPressure(info.QnhInHg, info.QnhInmB, st), info.Transponder), Inline: true},
			{Name: t("f.flightplan"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.rules"), orNA(formatFlightRules(fp.FlightRules)), t("l.dep"), orNA(fp.Departure), t("l.arr"), orNA(fp.Arrival), t("l.alt"), orNA(fp.Alternate)), Inline: true},
			{Name: t("f.timing"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.etd"), etd, t("l.enroute"), enroute, t("l.fuel"), fuel), Inline: true},
			{Name: t("f.tech"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.ac"), orNA(fp.AircraftFaa), t("l.assigned"), orNA(fp.AssignedTransponder), t("f.server"), info.Server), Inline: true},
			{Name: t("f.remarks"), Value: fmt.Sprintf("```%s```", orNA(fp.Remarks)), Inline: false},
		},
		Footer: footer("VATSIM", st, info.LogonTime, info.LastUpdated),
	}
}

func ivaoPilotEmbed(info ivao.PilotInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	dep, arr := "N/A", "N/A"
	route, remarks, aircraft, aircraftFaa := "N/A", "N/A", "N/A", "N/A"
	flightRules, alternate, deptime, enroute, fuel := "N/A", "N/A", "N/A", "N/A", "N/A"
	if info.FlightPlan != nil {
		fp := info.FlightPlan
		dep = fp.DepartureId
		arr = fp.ArrivalId
		route = fp.Route
		remarks = fp.Remarks
		aircraft = fp.AircraftId
		aircraftFaa = fp.AircraftId
		flightRules = formatFlightRules(fp.FlightRules)
		alternate = fp.AlternativeId
		if alternate == "" {
			alternate = "N/A"
		}
		if fp.DepartureTime > 0 {
			deptime = formatClock(fp.DepartureTime)
		}
		if fp.Eet > 0 {
			enroute = formatSeconds(fp.Eet)
		}
		if fp.Endurance > 0 {
			fuel = formatSeconds(fp.Endurance)
		}
	}
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + dep + " → " + arr + "  •  IVAO",
		Description: fmt.Sprintf("**%s %s** | %s | ID: `%d`", info.User.FirstName, info.User.LastName, aircraft, info.UserId),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: t("f.route"), Value: fmt.Sprintf("```%s```", route), Inline: false},
			{Name: t("f.position"), Value: fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.LastTrack.Latitude, info.LastTrack.Longitude), Inline: true},
			{Name: t("f.state"), Value: fmt.Sprintf("**Alt:** %s\n**GS:** %s\n**HDG:** %d°\n**%s:** %s", fmtAltitude(info.LastTrack.Altitude, st), fmtSpeed(info.LastTrack.Groundspeed, st), info.LastTrack.Heading, t("l.state"), info.LastTrack.State), Inline: true},
			{Name: t("f.weather"), Value: fmt.Sprintf("**QNH:** %s\n**Squawk:** %04d", ivaoQnh(info, st), info.LastTrack.Transponder), Inline: true},
			{Name: t("f.flightplan"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.rules"), flightRules, t("l.dep"), dep, t("l.arr"), arr, t("l.alt"), alternate), Inline: true},
			{Name: t("f.timing"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.etd"), deptime, t("l.enroute"), enroute, t("l.fuel"), fuel), Inline: true},
			{Name: t("f.tech"), Value: fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s", t("l.ac"), aircraftFaa, t("l.onground"), yesNo(info.LastTrack.OnGround, st), t("f.server"), info.ServerId), Inline: true},
			{Name: t("f.remarks"), Value: fmt.Sprintf("```%s```", remarks), Inline: false},
		},
		Footer: footer("IVAO", st, info.CreatedAt, ""),
	}
}

func vatsimControllerEmbed(info vatsim.ControllerInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility) + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ATIS", Value: fmt.Sprintf("```%s```", strings.Join(info.TextPosition, "\n")), Inline: false},
			{Name: t("f.freq"), Value: info.Frequency + " MHz", Inline: true},
			{Name: t("f.range"), Value: fmtDistance(info.VisualRange, st), Inline: true},
			{Name: t("f.server"), Value: info.Server, Inline: true},
		},
		Footer: footer("VATSIM", st, info.LogonTime, info.LastUpdated),
	}
}

func vatsimAtisEmbed(info vatsim.AtisInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility) + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ATIS " + info.AtisCode, Value: fmt.Sprintf("```%s```", strings.Join(info.TextAtis, "\n")), Inline: false},
			{Name: t("f.freq"), Value: info.Frequency + " MHz", Inline: true},
			{Name: t("f.range"), Value: fmtDistance(info.VisualRange, st), Inline: true},
			{Name: t("f.server"), Value: info.Server, Inline: true},
		},
		Footer: footer("VATSIM", st, info.LogonTime, info.LastUpdated),
	}
}

func ivaoControllerAndAtisEmbed(info ivao.AtcInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	atisText := "N/A"
	atisTitle := "ATIS"
	if info.Atis != nil && len(info.Atis.Lines) > 0 {
		atisText = strings.Join(info.Atis.Lines, "\n")
		if info.Atis.Revision != "" {
			atisTitle = "ATIS " + info.Atis.Revision
		}
	}
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + info.AtcSession.Position + "  •  IVAO",
		Description: fmt.Sprintf("**%s %s** | ID: `%d` | %s", info.User.FirstName, info.User.LastName, info.UserId, info.User.Rating.AtcRating.ShortName),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: atisTitle, Value: fmt.Sprintf("```%s```", atisText), Inline: false},
			{Name: t("f.freq"), Value: fmt.Sprintf("%.3f MHz", info.AtcSession.Frequency), Inline: true},
			{Name: t("f.range"), Value: "N/A", Inline: true},
			{Name: t("f.server"), Value: info.ServerId, Inline: true},
		},
		Footer: footer("IVAO", st, info.CreatedAt, ""),
	}
}

func vatsimPrefileEmbed(info vatsim.PrefileInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + t("t.prefile") + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d`", info.Name, info.Cid),
		Color:       0xFEE75C,
		Fields: []*discordgo.MessageEmbedField{
			{Name: t("f.itinerary"), Value: fmt.Sprintf("`%s` → `%s`", info.FlightPlan.Departure, info.FlightPlan.Arrival), Inline: false},
			{Name: t("f.aircraft"), Value: info.FlightPlan.AircraftFaa, Inline: true},
			{Name: t("f.rules"), Value: info.FlightPlan.FlightRules, Inline: true},
			{Name: t("f.xpdr"), Value: "`" + info.FlightPlan.AssignedTransponder + "`", Inline: true},
			{Name: t("f.etd"), Value: formatHHMMZ(info.FlightPlan.Deptime), Inline: true},
			{Name: t("f.ete"), Value: info.FlightPlan.EnrouteTime, Inline: true},
			{Name: t("f.fuel"), Value: info.FlightPlan.FuelTime, Inline: true},
			{Name: t("f.route"), Value: fmt.Sprintf("```%s```", info.FlightPlan.Route), Inline: false},
			{Name: t("f.remarks"), Value: fmt.Sprintf("```%s```", info.FlightPlan.Remarks), Inline: false},
		},
		Footer: footer("VATSIM  •  "+t("t.prefile"), st, "", info.LastUpdated),
	}
}

func vatsimFacilityEmbed(info vatsim.FacilityInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Facility  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0xEB459E,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: t("f.shortname"), Value: info.ShortName, Inline: true},
			{Name: t("f.longname"), Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Facility"},
	}
}

func vatsimRatingEmbed(info vatsim.RatingInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  ATC Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: t("f.shortname"), Value: info.ShortName, Inline: true},
			{Name: t("f.longname"), Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  ATC Rating"},
	}
}

func vatsimPilotRatingEmbed(info vatsim.PilotRatingInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Pilot Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: t("f.shortname"), Value: info.ShortName, Inline: true},
			{Name: t("f.longname"), Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Pilot Rating"},
	}
}

func vatsimMilitaryRatingEmbed(info vatsim.MilitaryRatingInfo, st config.Settings) *discordgo.MessageEmbed {
	t := tr(st)
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Military Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: t("f.shortname"), Value: info.ShortName, Inline: true},
			{Name: t("f.longname"), Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Military Rating"},
	}
}

// Discord requires required options first, so the (now optional) network option
// comes after cid_or_callsign. No network = both VATSIM and IVAO are searched.
func networkAndIdentOptions(entity string) []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "cid_or_callsign",
			Description: "Find " + entity + " Info from CID or Callsign",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "network",
			Description: "Network to search (leave empty to search both VATSIM and IVAO)",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "VATSIM", Value: config.NetworkVatsim},
				{Name: "IVAO", Value: config.NetworkIvao},
			},
		},
	}
}

func idOrNameOptions(entity string) []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "id_or_name",
			Description: "Find " + entity + " from ID or short name (e.g. '5' or 'APP')",
			Required:    true,
		},
	}
}

func withMap(e *discordgo.MessageEmbed, mapURL string) *discordgo.MessageEmbed {
	if mapURL != "" {
		e.Image = &discordgo.MessageEmbedImage{URL: mapURL}
		e.Footer.Text += "  •  © Mapbox © OpenStreetMap"
	}
	return e
}

func ivaoMapURL(info ivao.PilotInfo) string {
	cur := &geo.Point{Lat: info.LastTrack.Latitude, Lon: info.LastTrack.Longitude}
	if info.FlightPlan == nil {
		return geo.FlightMapURL("", "", "", cur)
	}
	fp := info.FlightPlan
	return geo.FlightMapURL(fp.DepartureId, fp.ArrivalId, fp.Route, cur)
}

var Commands = []*discordgo.ApplicationCommand{
	{Name: "pilot", Description: "Get a Pilot info from VATSIM or IVAO", Options: networkAndIdentOptions("Pilot")},
	{Name: "controller", Description: "Get a Controller info from VATSIM or IVAO", Options: networkAndIdentOptions("Controller")},
	{Name: "atis", Description: "Get a Atis info of an atc from VATSIM or IVAO", Options: networkAndIdentOptions("Atis")},
	{Name: "prefile", Description: "Get a prefile info from VATSIM or IVAO", Options: networkAndIdentOptions("prefile")},
	{Name: "facility", Description: "Get a VATSIM facility by ID or short name", Options: idOrNameOptions("Facility")},
	{Name: "rating", Description: "Get a VATSIM ATC rating by ID or short name", Options: idOrNameOptions("ATC Rating")},
	{Name: "pilotrating", Description: "Get a VATSIM pilot rating by ID or short name", Options: idOrNameOptions("Pilot Rating")},
	{Name: "militaryrating", Description: "Get a VATSIM military rating by ID or short name", Options: idOrNameOptions("Military Rating")},
}

var CommandHandlers = map[string]func(session *discordgo.Session, interaction *discordgo.InteractionCreate){
	"pilot": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		opts := getOptions(interaction)
		cidOrCallsign := normalizeCallsign(opts["cid_or_callsign"].StringValue())
		wantVatsim, wantIvao := wantedNetworks(opts)

		var embeds []*discordgo.MessageEmbed
		var lastErr error

		if wantVatsim {
			var info vatsim.PilotInfo
			var err error
			if isPilotCallsign(cidOrCallsign) {
				info, err = vatsim.GetPilotFromCallsign(cidOrCallsign)
			} else {
				info, err = vatsim.GetPilotFromId(cidOrCallsign)
			}
			if err != nil {
				lastErr = err
			} else {
				fp := info.FlightPlan
				embeds = append(embeds, withMap(vatsimPilotEmbed(info, st),
					geo.FlightMapURL(fp.Departure, fp.Arrival, fp.Route,
						&geo.Point{Lat: info.Latitude, Lon: info.Longitude})))
			}
		}
		if wantIvao {
			var info ivao.PilotInfo
			var err error
			if isPilotCallsign(cidOrCallsign) {
				info, err = ivao.GetPilotFromCallsign(cidOrCallsign)
			} else {
				info, err = ivao.GetPilotFromUserId(cidOrCallsign)
			}
			if err != nil {
				lastErr = err
			} else {
				embeds = append(embeds, withMap(ivaoPilotEmbed(info, st), ivaoMapURL(info)))
			}
		}
		replyLookup(session, interaction, st, embeds, lastErr, wantVatsim && wantIvao)
	},

	"controller": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		opts := getOptions(interaction)
		cidOrCallsign := normalizeCallsign(opts["cid_or_callsign"].StringValue())
		wantVatsim, wantIvao := wantedNetworks(opts)

		var embeds []*discordgo.MessageEmbed
		var lastErr error

		if wantVatsim {
			var info vatsim.ControllerInfo
			var err error
			if isFacilityCallsign(cidOrCallsign) {
				info, err = vatsim.GetControllerFromCallsign(cidOrCallsign)
			} else {
				info, err = vatsim.GetControllerFromId(cidOrCallsign)
			}
			if err != nil {
				lastErr = err
			} else {
				embeds = append(embeds, vatsimControllerEmbed(info, st))
			}
		}
		if wantIvao {
			var info ivao.AtcInfo
			var err error
			if isFacilityCallsign(cidOrCallsign) {
				info, err = ivao.GetAtcFromCallsign(cidOrCallsign)
			} else {
				info, err = ivao.GetAtcFromUserId(cidOrCallsign)
			}
			if err != nil {
				lastErr = err
			} else {
				embeds = append(embeds, ivaoControllerAndAtisEmbed(info, st))
			}
		}
		replyLookup(session, interaction, st, embeds, lastErr, wantVatsim && wantIvao)
	},

	"atis": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		opts := getOptions(interaction)
		cidOrCallsign := normalizeCallsign(opts["cid_or_callsign"].StringValue())
		wantVatsim, wantIvao := wantedNetworks(opts)

		var embeds []*discordgo.MessageEmbed
		var lastErr error

		if wantVatsim {
			var info vatsim.AtisInfo
			var err error
			if isNumeric(cidOrCallsign) {
				info, err = vatsim.GetAtisFromId(cidOrCallsign)
			} else {
				info, err = vatsim.GetAtisFromCallsign(toAtisCallsign(cidOrCallsign))
			}
			if err != nil {
				lastErr = err
			} else {
				embeds = append(embeds, vatsimAtisEmbed(info, st))
			}
		}
		if wantIvao {
			// TODO : On IVAO ATC and ATIS are the same, but it will be great to just put the position name like LFBP insted of LFBP_TWR or LFBP_GND...
			info, err := ivao.GetAtcFromCallsign(strings.ToUpper(cidOrCallsign))
			switch {
			case err != nil:
				lastErr = err
			case info.Atis == nil:
				lastErr = errors.New(tr(st)("error.no_atis"))
			default:
				embeds = append(embeds, ivaoControllerAndAtisEmbed(info, st))
			}
		}
		replyLookup(session, interaction, st, embeds, lastErr, wantVatsim && wantIvao)
	},

	"prefile": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		opts := getOptions(interaction)
		wantVatsim, _ := wantedNetworks(opts)

		// IVAO has no prefiles: only an explicit network:IVAO is an error, "both" just returns VATSIM.
		if !wantVatsim {
			//TODO : Trouver un moyen pour les presets pour IVAO
			respondError(session, interaction, tr(st)("error.ivao_prefile"))
			return
		}

		cidOrCallsign := normalizeCallsign(opts["cid_or_callsign"].StringValue())
		var info vatsim.PrefileInfo
		var err error
		if isPilotCallsign(cidOrCallsign) {
			info, err = vatsim.GetPrefileFromCallsign(cidOrCallsign)
		} else {
			info, err = vatsim.GetPrefileFromId(cidOrCallsign)
		}
		if err != nil {
			fmt.Println(err)
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, withMap(vatsimPrefileEmbed(info, st),
			geo.FlightMapURL(info.FlightPlan.Departure, info.FlightPlan.Arrival, info.FlightPlan.Route, nil)))
	},

	"facility": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		query := getOptions(interaction)["id_or_name"].StringValue()
		var info vatsim.FacilityInfo
		var err error
		if isNumeric(query) {
			info, err = vatsim.GetFacilityFromId(query)
		} else {
			info, err = vatsim.GetFacilityFromShortName(query)
		}
		if err != nil {
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimFacilityEmbed(info, st))
	},

	"rating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		query := getOptions(interaction)["id_or_name"].StringValue()
		var info vatsim.RatingInfo
		var err error
		if isNumeric(query) {
			info, err = vatsim.GetRatingFromId(query)
		} else {
			info, err = vatsim.GetRatingFromShortName(query)
		}
		if err != nil {
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimRatingEmbed(info, st))
	},

	"pilotrating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		query := getOptions(interaction)["id_or_name"].StringValue()
		var info vatsim.PilotRatingInfo
		var err error
		if isNumeric(query) {
			info, err = vatsim.GetPilotRatingFromId(query)
		} else {
			info, err = vatsim.GetPilotRatingFromShortName(query)
		}
		if err != nil {
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimPilotRatingEmbed(info, st))
	},

	"militaryrating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		st := SettingsOf(interaction)
		query := getOptions(interaction)["id_or_name"].StringValue()
		var info vatsim.MilitaryRatingInfo
		var err error
		if isNumeric(query) {
			info, err = vatsim.GetMilitaryRatingFromId(query)
		} else {
			info, err = vatsim.GetMilitaryRatingFromShortName(query)
		}
		if err != nil {
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimMilitaryRatingEmbed(info, st))
	},
}
