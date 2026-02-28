package commands

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

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

func respondIVAO(s *discordgo.Session, i *discordgo.InteractionCreate) {
	respondError(s, i, "IVAO n'est pas encore supporté.")
}

func getOptions(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range i.ApplicationCommandData().Options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

func pilotEmbed(info vatsim.PilotInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + info.FlightPlan.Departure + " → " + info.FlightPlan.Arrival,
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

func controllerEmbed(info vatsim.ControllerInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility),
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

func atisEmbed(info vatsim.AtisInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  " + formatFacility(info.Facility),
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

func prefileEmbed(info vatsim.PrefileInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       info.Callsign + "  •  Préfile",
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

func facilityEmbed(info vatsim.FacilityInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Facility", info.ShortName),
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

func ratingEmbed(info vatsim.RatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  ATC Rating", info.ShortName),
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

func pilotRatingEmbed(info vatsim.PilotRatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Pilot Rating", info.ShortName),
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

func militaryRatingEmbed(info vatsim.MilitaryRatingInfo) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s  •  Military Rating", info.ShortName),
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
	{Name: "pilot", Description: "Get a Pilot info from VATSIM", Options: networkAndIdentOptions("Pilot")},
	{Name: "controller", Description: "Get a Controller info from VATSIM", Options: networkAndIdentOptions("Controller")},
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
		if opts["network"].StringValue() == "IVAO" {
			respondIVAO(session, interaction)
			return
		}
		cidOrCallsign := opts["cid_or_callsign"].StringValue()
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
		respond(session, interaction, pilotEmbed(info))
	},

	"controller": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)
		if opts["network"].StringValue() == "IVAO" {
			respondIVAO(session, interaction)
			return
		}
		cidOrCallsign := opts["cid_or_callsign"].StringValue()
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
		respond(session, interaction, controllerEmbed(info))
	},

	"atis": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)
		if opts["network"].StringValue() == "IVAO" {
			respondIVAO(session, interaction)
			return
		}
		cidOrCallsign := toAtisCallsign(opts["cid_or_callsign"].StringValue())
		var info vatsim.AtisInfo
		var err error
		if isFacilityCallsign(cidOrCallsign) {
			info, err = vatsim.GetAtisFromCallsign(cidOrCallsign)
		} else {
			info, err = vatsim.GetAtisFromId(cidOrCallsign)
		}
		if err != nil {
			fmt.Println(err)
			respondError(session, interaction, err.Error())
			return
		}
		respond(session, interaction, atisEmbed(info))
	},

	"prefile": func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		opts := getOptions(interaction)
		if opts["network"].StringValue() == "IVAO" {
			respondIVAO(session, interaction)
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
		respond(session, interaction, prefileEmbed(info))
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
		respond(session, interaction, facilityEmbed(info))
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
		respond(session, interaction, ratingEmbed(info))
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
		respond(session, interaction, pilotRatingEmbed(info))
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
		respond(session, interaction, militaryRatingEmbed(info))
	},
}
