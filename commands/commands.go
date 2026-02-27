package commands

import (
	"fmt"
	"regexp"
	"strings"

	vatsim "github.com/Skyfield1888/Vatsim/api/vatsim"
	"github.com/bwmarrin/discordgo"
)

func isPilotCallsign(s string) bool {
	re := regexp.MustCompile(`^[A-Z]{3}[A-Z0-9]{1,4}$`)

	return re.MatchString(s)
}

func isFacilityCallsign(s string) bool {
	re := regexp.MustCompile(`(?i)^[A-Z]{2,7}(_+[A-Z0-9]{1,3})*_+(DEL|GND|TWR|APP|DEP|CTR|FSS|ATIS|OBS|AFIS|INFO|RADAR|OCA)$`)

	return re.MatchString(s)
}

func formatflightRules(code string) string {
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

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "pilot",
		Description: "Get a Pilot info from VATSIM",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "network",
				Description: "On which network do you want to get the Pilot Info",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "VATSIM",
						Value: "VATSIM",
					},
					{
						Name:  "IVAO",
						Value: "IVAO",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cid_or_callsign",
				Description: "Find Pilot Info from CID or Callsign",
				Required:    true,
			},
		},
	},
	{
		Name:        "controller",
		Description: "Get a Controller info from VATSIM",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "network",
				Description: "On which network do you want to get the Controller Info",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "VATSIM",
						Value: "VATSIM",
					},
					{
						Name:  "IVAO",
						Value: "IVAO",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cid_or_callsign",
				Description: "Find Controller Info from CID or Callsign",
				Required:    true,
			},
		},
	},
	{
		Name:        "atis",
		Description: "Get a Atis info of an atc from VATSIM or IVAO",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "network",
				Description: "On which network do you want to get the Atis Info",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "VATSIM",
						Value: "VATSIM",
					},
					{
						Name:  "IVAO",
						Value: "IVAO",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cid_or_callsign",
				Description: "Find Atis Info from CID or Callsign of an atc",
				Required:    true,
			},
		},
	},
	{
		Name:        "prefile",
		Description: "Get a prefile info of an atc from VATSIM or IVAO",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "network",
				Description: "On which network do you want to get the prefile Info",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "VATSIM",
						Value: "VATSIM",
					},
					{
						Name:  "IVAO",
						Value: "IVAO",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cid_or_callsign",
				Description: "Find prefile Info from CID or Callsign",
				Required:    true,
			},
		},
	},
}

var CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"pilot": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		network := optionMap["network"].StringValue()
		cidOrCallsign := optionMap["cid_or_callsign"].StringValue()

		isCallsign := isPilotCallsign(cidOrCallsign)

		switch network {
		case "VATSIM":
			var info vatsim.PilotInfo
			var err error
			if isCallsign {
				info, err = vatsim.GetPilotFromCallsign(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			} else {
				info, err = vatsim.GetPilotFromId(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			}
			embed := &discordgo.MessageEmbed{
				Title:       info.Callsign + "  •  " + info.FlightPlan.Departure + " → " + info.FlightPlan.Arrival,
				Description: fmt.Sprintf("**%s** | %s | CID: `%d`", info.Name, info.FlightPlan.AircraftShort, info.Cid),
				Color:       0x5865F2,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Route",
						Value:  fmt.Sprintf("```%s```", info.FlightPlan.Route),
						Inline: false,
					},
					{
						Name:   "Position",
						Value:  fmt.Sprintf("**Lat:** %.4f °\n**Lon:** %.4f °", info.Latitude, info.Longitude),
						Inline: true,
					},
					{
						Name:   "État",
						Value:  fmt.Sprintf("**Alt:** %d ft\n**GS:** %d kts\n**HDG:** %d°", info.Altitude, info.Groundspeed, info.Heading),
						Inline: true,
					},
					{
						Name:   "Météo",
						Value:  fmt.Sprintf("**QNH:** %.2f inHg\n**QNH:** %d hPa\n**Squawk:** %s", info.QnhInHg, info.QnhInmB, info.Transponder),
						Inline: true,
					},
					{
						Name:   "Plan de vol",
						Value:  fmt.Sprintf("**Règles:** %s\n**Départ:** %s\n**Arrivée:** %s\n**Alternate:** %s", formatflightRules(info.FlightPlan.FlightRules), info.FlightPlan.Departure, info.FlightPlan.Arrival, info.FlightPlan.Alternate),
						Inline: true,
					},
					{
						Name:   "Timing",
						Value:  fmt.Sprintf("**Départ prévu:** %s\n**En route:** %s\n**Carburant:** %s", formatDuration(info.FlightPlan.Deptime), formatDuration(info.FlightPlan.EnrouteTime), formatDuration(info.FlightPlan.FuelTime)),
						Inline: true,
					},
					{
						Name:   "Technique",
						Value:  fmt.Sprintf("**Avion (FAA):** %s\n**Squawk assigné:** %s\n**Serveur:** %s", info.FlightPlan.AircraftFaa, info.FlightPlan.AssignedTransponder, info.Server),
						Inline: true,
					},
					{
						Name:   "Remarques",
						Value:  fmt.Sprintf("```%s```", info.FlightPlan.Remarks),
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
				},
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		case "IVAO":
			embed := &discordgo.MessageEmbed{
				Title:       "error",
				Description: "IVAO Is Not Supported for now",
				Color:       0xFF0000,
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	},
	"controller": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		network := optionMap["network"].StringValue()
		cidOrCallsign := optionMap["cid_or_callsign"].StringValue()

		isCallsign := isFacilityCallsign(cidOrCallsign)

		switch network {
		case "VATSIM":
			var info vatsim.ControllerInfo
			var err error
			if isCallsign {
				info, err = vatsim.GetControllerFromCallsign(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			} else {
				info, err = vatsim.GetControllerFromId(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			}
			embed := &discordgo.MessageEmbed{
				Title:       info.Callsign + "  •  " + formatFacility(info.Facility),
				Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
				Color:       0x57F287,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "ATIS",
						Value:  fmt.Sprintf("```%s```", strings.Join(info.TextPosition, "\n")),
						Inline: false,
					},
					{
						Name:   "Fréquence",
						Value:  info.Frequency + " MHz",
						Inline: true,
					},
					{
						Name:   "Portée visuelle",
						Value:  fmt.Sprintf("%d nm", info.VisualRange),
						Inline: true,
					},
					{
						Name:   "Serveur",
						Value:  info.Server,
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
				},
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		case "IVAO":
			embed := &discordgo.MessageEmbed{
				Title:       "error",
				Description: "IVAO Is Not Supported for now",
				Color:       0xFF0000,
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	},
	"atis": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		network := optionMap["network"].StringValue()
		cidOrCallsign := optionMap["cid_or_callsign"].StringValue()

		isCallsign := isFacilityCallsign(cidOrCallsign)

		switch network {
		case "VATSIM":
			var info vatsim.AtisInfo
			var err error
			if isCallsign {
				info, err = vatsim.GetAtisFromCallsign(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			} else {
				info, err = vatsim.GetAtisFromId(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			}
			embed := &discordgo.MessageEmbed{
				Title:       info.Callsign + "  •  " + formatFacility(info.Facility),
				Description: fmt.Sprintf("**%s** | CID: `%d` | %s", info.Name, info.Cid, formatRating(info.Rating)),
				Color:       0x57F287,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "ATIS " + info.AtisCode,
						Value:  fmt.Sprintf("```%s```", strings.Join(info.TextAtis, "\n")),
						Inline: false,
					},
					{
						Name:   "Fréquence",
						Value:  info.Frequency + " MHz",
						Inline: true,
					},
					{
						Name:   "Portée visuelle",
						Value:  fmt.Sprintf("%d nm", info.VisualRange),
						Inline: true,
					},
					{
						Name:   "Serveur",
						Value:  info.Server,
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "VATSIM  •  Connecté depuis " + info.LogonTime + "  •  Mis à jour " + info.LastUpdated,
				},
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		case "IVAO":
			embed := &discordgo.MessageEmbed{
				Title:       "error",
				Description: "IVAO Is Not Supported for now",
				Color:       0xFF0000,
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	},
	"prefile": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		network := optionMap["network"].StringValue()
		cidOrCallsign := optionMap["cid_or_callsign"].StringValue()

		isCallsign := isFacilityCallsign(cidOrCallsign)

		switch network {
		case "VATSIM":
			var info vatsim.PrefileInfo
			var err error
			if isCallsign {
				info, err = vatsim.GetPrefileFromCallsign(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			} else {
				info, err = vatsim.GetPrefileFromId(cidOrCallsign)
				if err != nil {
					fmt.Println(err)
					embed := &discordgo.MessageEmbed{
						Title:       "error",
						Description: err.Error(),
						Color:       0xFF0000,
					}
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{embed},
						},
					})
					return
				}
			}
			embed := &discordgo.MessageEmbed{
				Title:       info.Callsign + "  •  Préfile",
				Description: fmt.Sprintf("**%s** | CID: `%d`", info.Name, info.Cid),
				Color:       0xFEE75C,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Itinéraire",
						Value:  fmt.Sprintf("`%s` → `%s`", info.FlightPlan.Departure, info.FlightPlan.Arrival),
						Inline: false,
					},
					{
						Name:   "Aéronef",
						Value:  info.FlightPlan.AircraftFaa,
						Inline: true,
					},
					{
						Name:   "Règles de vol",
						Value:  info.FlightPlan.FlightRules,
						Inline: true,
					},
					{
						Name:   "Transpondeur",
						Value:  "`" + info.FlightPlan.AssignedTransponder + "`",
						Inline: true,
					},
					{
						Name:   "Départ prévu",
						Value:  info.FlightPlan.Deptime + " Z",
						Inline: true,
					},
					{
						Name:   "Temps de route",
						Value:  info.FlightPlan.EnrouteTime,
						Inline: true,
					},
					{
						Name:   "Carburant",
						Value:  info.FlightPlan.FuelTime,
						Inline: true,
					},
					{
						Name:   "Route",
						Value:  fmt.Sprintf("```%s```", info.FlightPlan.Route),
						Inline: false,
					},
					{
						Name:   "Remarques",
						Value:  fmt.Sprintf("```%s```", info.FlightPlan.Remarks),
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "VATSIM  •  Préfile  •  Mis à jour " + info.LastUpdated,
				},
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		case "IVAO":
			embed := &discordgo.MessageEmbed{
				Title:       "error",
				Description: "IVAO Is Not Supported for now",
				Color:       0xFF0000,
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	},
}
