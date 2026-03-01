package commands

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	ivao "github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	vatsim "github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/bwmarrin/discordgo"
)

func isPilotCallsign(s string) bool {
	return regexp.MustCompile(`^[A-Z]{3}[A-Z0-9]{1,4}$`).MatchString(s)
}

func isFacilityCallsign(s string) bool {
	return regexp.MustCompile(`(?i)^[A-Z]{2,7}(_+[A-Z0-9]{1,3})*_+(DEL|GND|TWR|APP|DEP|CTR|FSS|ATIS|OBS|AFIS|INFO|RADAR|OCA)$`).MatchString(s)
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

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	respond(s, i, &discordgo.MessageEmbed{
		Title:       "Erreur",
		Description: message,
		Color:       0xFF0000,
	})
}

func getOptions(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range i.ApplicationCommandData().Options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

func vatsimPpilotEmbed(info vatsim.PilotInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + info.FlightPlan.Departure + " → " + info.FlightPlan.Arrival + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | %s | CID: `%d`", info.Name, info.FlightPlan.AircraftShort, info.Cid),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Route", Value: fmt.Sprintf("```%s```", info.FlightPlan.Route), Inline: false},
			{Name: "Position", Value: fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.Latitude, info.Longitude), Inline: true},
			{Name: "État", Value: fmt.Sprintf("**Alt:** %d ft\n**GS:** %d kts\n**HDG:** %d°", info.Altitude, info.Groundspeed, info.Heading), Inline: true},
			{Name: "Météo", Value: fmt.Sprintf("**QNH:** %.2f inHg\n**QNH:** %d hPa\n**Squawk:** %s", info.QnhInHg, info.QnhInmB, info.Transponder), Inline: true},
			{Name: "Plan de vol", Value: fmt.Sprintf("**Règles:** %s\n**Départ:** %s\n**Arrivée:** %s\n**Alternate:** %s", formatFlightRules(info.FlightPlan.FlightRules), info.FlightPlan.Departure, info.FlightPlan.Arrival, info.FlightPlan.Alternate), Inline: true},
			{Name: "Timing", Value: fmt.Sprintf("**Départ prévu:** %s\n**En route:** %s\n**Carburant:** %s", formatDuration(info.FlightPlan.Deptime), formatDuration(info.FlightPlan.EnrouteTime), formatDuration(info.FlightPlan.FuelTime)), Inline: true},
			{Name: "Technique", Value: fmt.Sprintf("**Avion (FAA):** %s\n**Squawk assigné:** %s\n**Serveur:** %s", info.FlightPlan.AircraftFaa, info.FlightPlan.AssignedTransponder, info.Server), Inline: true},
			{Name: "Remarques", Value: fmt.Sprintf("```%s```", info.FlightPlan.Remarks), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
		},
	}
}

func vatsimControllerEmbed(info vatsim.ControllerInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility) + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ATIS", Value: fmt.Sprintf("```%s```", strings.Join(info.TextPosition, "\n")), Inline: false},
			{Name: "Fréquence", Value: info.Frequency + " MHz", Inline: true},
			{Name: "Portée visuelle", Value: fmt.Sprintf("%d nm", info.VisualRange), Inline: true},
			{Name: "Serveur", Value: info.Server, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
		},
	}
}

func vatsimAtisEmbed(info vatsim.AtisInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility) + "  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ATIS " + info.AtisCode, Value: fmt.Sprintf("```%s```", strings.Join(info.TextAtis, "\n")), Inline: false},
			{Name: "Fréquence", Value: info.Frequency + " MHz", Inline: true},
			{Name: "Portée visuelle", Value: fmt.Sprintf("%d nm", info.VisualRange), Inline: true},
			{Name: "Serveur", Value: info.Server, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
		},
	}
}

func vatsimPrefileEmbed(info vatsim.PrefileInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  Préfile  •  VATSIM",
		Description: fmt.Sprintf("**%s** | CID: `%d`", info.Name, info.Cid),
		Color:       0xFEE75C,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Itinéraire", Value: fmt.Sprintf("`%s` → `%s`", info.FlightPlan.Departure, info.FlightPlan.Arrival), Inline: false},
			{Name: "Aéronef", Value: info.FlightPlan.AircraftFaa, Inline: true},
			{Name: "Règles de vol", Value: info.FlightPlan.FlightRules, Inline: true},
			{Name: "Transpondeur", Value: "`" + info.FlightPlan.AssignedTransponder + "`", Inline: true},
			{Name: "Départ prévu", Value: info.FlightPlan.Deptime + " Z", Inline: true},
			{Name: "Temps de route", Value: info.FlightPlan.EnrouteTime, Inline: true},
			{Name: "Carburant", Value: info.FlightPlan.FuelTime, Inline: true},
			{Name: "Route", Value: fmt.Sprintf("```%s```", info.FlightPlan.Route), Inline: false},
			{Name: "Remarques", Value: fmt.Sprintf("```%s```", info.FlightPlan.Remarks), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VATSIM  •  Préfile  •  Mis à jour " + info.LastUpdated,
		},
	}
}

func vatsimFacilityEmbed(info vatsim.FacilityInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Facility  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0xEB459E,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: "Nom court", Value: info.ShortName, Inline: true},
			{Name: "Nom complet", Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Facility"},
	}
}

func vatsimRatingEmbed(info vatsim.RatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  ATC Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0xED4245,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: "Nom court", Value: info.ShortName, Inline: true},
			{Name: "Nom complet", Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  ATC Rating"},
	}
}

func vatsimPilotRatingEmbed(info vatsim.PilotRatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Pilot Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: "Nom court", Value: info.ShortName, Inline: true},
			{Name: "Nom complet", Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Pilot Rating"},
	}
}

func vatsimMilitaryRatingEmbed(info vatsim.MilitaryRatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Military Rating  •  VATSIM", info.ShortName),
		Description: info.LongName,
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ID", Value: fmt.Sprintf("`%d`", info.Id), Inline: true},
			{Name: "Nom court", Value: info.ShortName, Inline: true},
			{Name: "Nom complet", Value: info.LongName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "VATSIM  •  Military Rating"},
	}
}

func ivaoPilotEmbed(info ivao.PilotInfo) *discordgo.MessageEmbed {
	dep, arr := "N/A", "N/A"
	route, remarks, aircraft, flightRules := "N/A", "N/A", "N/A", "N/A"
	if info.FlightPlan != nil {
		dep = info.FlightPlan.DepartureId
		arr = info.FlightPlan.ArrivalId
		route = info.FlightPlan.Route
		remarks = info.FlightPlan.Remarks
		aircraft = info.FlightPlan.AircraftId
		flightRules = formatFlightRules(info.FlightPlan.FlightRules)
	}
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + dep + " → " + arr + "  •  IVAO",
		Description: fmt.Sprintf("**%s %s** | %s | ID: `%d`", info.User.FirstName, info.User.LastName, aircraft, info.UserId),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Route", Value: fmt.Sprintf("```%s```", route), Inline: false},
			{Name: "Position", Value: fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.LastTrack.Latitude, info.LastTrack.Longitude), Inline: true},
			{Name: "État", Value: fmt.Sprintf("**Alt:** %d ft\n**GS:** %d kts\n**HDG:** %d°\n**État:** %s", info.LastTrack.Altitude, info.LastTrack.Groundspeed, info.LastTrack.Heading, info.LastTrack.State), Inline: true},
			{Name: "Technique", Value: fmt.Sprintf("**Squawk:** %d\n**Au sol:** %t\n**Règles:** %s", info.LastTrack.Transponder, info.LastTrack.OnGround, flightRules), Inline: true},
			{Name: "Remarques", Value: fmt.Sprintf("```%s```", remarks), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "IVAO  •  Connecté depuis " + info.CreatedAt,
		},
	}
}

func ivaoControllerAndAtisEmbed(info ivao.AtcInfo) *discordgo.MessageEmbed {
	atisText := "N/A"
	atisRevision := ""
	if info.Atis != nil && len(info.Atis.Lines) > 0 {
		atisText = strings.Join(info.Atis.Lines, "\n")
		atisRevision = " " + info.Atis.Revision
	}
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + info.AtcSession.Position + "  •  IVAO",
		Description: fmt.Sprintf("**%s %s** | ID: `%d` | %s", info.User.FirstName, info.User.LastName, info.UserId, info.User.Rating.AtcRating.ShortName),
		Color:       0x57F287,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "ATIS" + atisRevision, Value: fmt.Sprintf("```%s```", atisText), Inline: false},
			{Name: "Fréquence", Value: fmt.Sprintf("%.3f MHz", info.AtcSession.Frequency), Inline: true},
			{Name: "Position", Value: fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.LastTrack.Latitude, info.LastTrack.Longitude), Inline: true},
			{Name: "Serveur", Value: info.ServerId, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "IVAO  •  Connecté depuis " + info.CreatedAt,
		},
	}
}

func networkAndIdentOptions(entity string) []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "network",
			Description: "On which network do you want to get the " + entity + " Info",
			Required:    true,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "VATSIM", Value: "VATSIM"},
				{Name: "IVAO", Value: "IVAO"},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "cid_or_callsign",
			Description: "Find " + entity + " Info from CID or Callsign",
			Required:    true,
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
		opts := getOptions(interaction)
		cidOrCallsign := opts["cid_or_callsign"].StringValue()

		if opts["network"].StringValue() == "IVAO" {
			var info ivao.PilotInfo
			var err error
			if isPilotCallsign(cidOrCallsign) {
				info, err = ivao.GetPilotFromCallsign(cidOrCallsign)
			} else {
				info, err = ivao.GetPilotFromUserId(cidOrCallsign)
			}
			if err != nil {
				fmt.Println(err)
				respondError(session, interaction, err.Error())
				return
			}
			respond(session, interaction, ivaoPilotEmbed(info))
			return
		}

		var info vatsim.PilotInfo
		var err error
		if isPilotCallsign(cidOrCallsign) {
			info, err = vatsim.GetPilotFromCallsign(cidOrCallsign)
		} else {
			info, err = vatsim.GetPilotFromId(cidOrCallsign)
		}
		if err != nil {
			fmt.Println(err)
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimPpilotEmbed(info))
	},

	"controller": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)
		cidOrCallsign := opts["cid_or_callsign"].StringValue()

		if opts["network"].StringValue() == "IVAO" {
			var info ivao.AtcInfo
			var err error
			if isFacilityCallsign(cidOrCallsign) {
				info, err = ivao.GetAtcFromCallsign(cidOrCallsign)
			} else {
				info, err = ivao.GetAtcFromUserId(cidOrCallsign)
			}
			if err != nil {
				fmt.Println(err)
				respondError(session, interaction, err.Error())
				return
			}
			respond(session, interaction, ivaoControllerAndAtisEmbed(info))
			return
		}

		var info vatsim.ControllerInfo
		var err error
		if isFacilityCallsign(cidOrCallsign) {
			info, err = vatsim.GetControllerFromCallsign(cidOrCallsign)
		} else {
			info, err = vatsim.GetControllerFromId(cidOrCallsign)
		}
		if err != nil {
			fmt.Println(err)
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimControllerEmbed(info))
	},

	"atis": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)
		cidOrCallsign := opts["cid_or_callsign"].StringValue()

		if opts["network"].StringValue() == "IVAO" {
			info, err := ivao.GetAtcFromCallsign(strings.ToUpper(cidOrCallsign))
			if err != nil {
				fmt.Println(err)
				respondError(session, interaction, err.Error())
				return
			}
			if info.Atis == nil {
				respondError(session, interaction, "Aucun ATIS disponible pour ce contrôleur.")
				return
			}
			respond(session, interaction, ivaoControllerAndAtisEmbed(info))
			return
		}

		atisCallsign := toAtisCallsign(cidOrCallsign)
		var info vatsim.AtisInfo
		var err error
		if isFacilityCallsign(atisCallsign) {
			info, err = vatsim.GetAtisFromCallsign(atisCallsign)
		} else {
			info, err = vatsim.GetAtisFromId(atisCallsign)
		}
		if err != nil {
			fmt.Println(err)
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, vatsimAtisEmbed(info))
	},

	"prefile": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)

		if opts["network"].StringValue() == "IVAO" {
			//TODO : Trouver un moyen pour les presets pour IVAO
			respondError(session, interaction, "Les préfiles ne sont pas disponibles sur IVAO.")
			return
		}

		cidOrCallsign := opts["cid_or_callsign"].StringValue()
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
		respond(session, interaction, vatsimPrefileEmbed(info))
	},

	"facility": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
		respond(session, interaction, vatsimFacilityEmbed(info))
	},

	"rating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
		respond(session, interaction, vatsimRatingEmbed(info))
	},

	"pilotrating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
		respond(session, interaction, vatsimPilotRatingEmbed(info))
	},

	"militaryrating": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
		respond(session, interaction, vatsimMilitaryRatingEmbed(info))
	},
}
