package commands

import (
	"fmt"
	"math"
	"strings"

	"github.com/Skyfield1888/virtualflightwatcher/config"
	"github.com/Skyfield1888/virtualflightwatcher/i18n"
	"github.com/bwmarrin/discordgo"
)

// ---------------------------------------------------------------------------
// Settings / translation
// ---------------------------------------------------------------------------

// settingsKey: settings are per server, or per user when the command is used in DMs.
func settingsKey(i *discordgo.InteractionCreate) string {
	if i.GuildID != "" {
		return i.GuildID
	}
	if i.User != nil {
		return "user:" + i.User.ID
	}
	if i.Member != nil && i.Member.User != nil {
		return "user:" + i.Member.User.ID
	}
	return "default"
}

func SettingsOf(i *discordgo.InteractionCreate) config.Settings {
	return config.Get(settingsKey(i))
}

// tr returns a translation function bound to the settings' language.
func tr(st config.Settings) func(string, ...any) string {
	return func(key string, args ...any) string { return i18n.T(st.Language, key, args...) }
}

// Dispatch is the single entry point for slash commands (used by main.go).
func Dispatch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	name := i.ApplicationCommandData().Name
	handler, ok := CommandHandlers[name]
	if !ok {
		return
	}

	st := SettingsOf(i)
	if name != "config" && st.AllowedChannelID != "" && i.ChannelID != st.AllowedChannelID {
		send(s, i, &discordgo.MessageEmbed{
			Title:       tr(st)("error.title"),
			Description: tr(st)("restricted", st.AllowedChannelID),
			Color:       0xFF0000,
		}, true)
		return
	}
	handler(s, i)
}

// ---------------------------------------------------------------------------
// Sending
// ---------------------------------------------------------------------------

func send(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, ephemeral bool) {
	sendMany(s, i, []*discordgo.MessageEmbed{embed}, ephemeral)
}

func sendMany(s *discordgo.Session, i *discordgo.InteractionCreate, embeds []*discordgo.MessageEmbed, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  flags,
		},
	})
}

// ---------------------------------------------------------------------------
// Multi-network lookups
// ---------------------------------------------------------------------------

// wantedNetworks: no "network" argument means both networks are queried.
func wantedNetworks(opts map[string]*discordgo.ApplicationCommandInteractionDataOption) (vatsimWanted, ivaoWanted bool) {
	if o, ok := opts["network"]; ok {
		switch o.StringValue() {
		case config.NetworkVatsim:
			return true, false
		case config.NetworkIvao:
			return false, true
		}
	}
	return true, true
}

// replyLookup sends every embed found (one per network), or an error if nothing was found.
func replyLookup(s *discordgo.Session, i *discordgo.InteractionCreate, st config.Settings, embeds []*discordgo.MessageEmbed, lastErr error, bothQueried bool) {
	if len(embeds) > 0 {
		sendMany(s, i, embeds, st.Ephemeral)
		return
	}
	if lastErr != nil {
		fmt.Println(lastErr)
	}
	if bothQueried || lastErr == nil {
		respondError(s, i, tr(st)("error.not_found_both"))
		return
	}
	respondError(s, i, lastErr.Error())
}

// ---------------------------------------------------------------------------
// Units & footer
// ---------------------------------------------------------------------------

func isMetric(st config.Settings) bool   { return st.Units == config.UnitsMetric }
func isImperial(st config.Settings) bool { return st.Units == config.UnitsImperial }

func fmtAltitude(ft int, st config.Settings) string {
	if isMetric(st) {
		return fmt.Sprintf("%d m", int(math.Round(float64(ft)*0.3048)))
	}
	return fmt.Sprintf("%d ft", ft)
}

func fmtSpeed(kts int, st config.Settings) string {
	if isMetric(st) {
		return fmt.Sprintf("%d km/h", int(math.Round(float64(kts)*1.852)))
	}
	if isImperial(st) {
		return fmt.Sprintf("%d mph", int(math.Round(float64(kts)*1.15078)))
	}
	return fmt.Sprintf("%d kts", kts)
}

func fmtDistance(nm int, st config.Settings) string {
	if isMetric(st) {
		return fmt.Sprintf("%d km", int(math.Round(float64(nm)*1.852)))
	}
	if isImperial(st) {
		return fmt.Sprintf("%d mi", int(math.Round(float64(nm)*1.15078)))
	}
	return fmt.Sprintf("%d nm", nm)
}

func fmtPressure(inHg float64, hPa int, st config.Settings) string {
	if isMetric(st) {
		return fmt.Sprintf("**QNH:** %d hPa", hPa)
	}
	if isImperial(st) {
		return fmt.Sprintf("**QNH:** %.2f inHg", inHg)
	}
	return fmt.Sprintf("**QNH:** %.2f inHg\n**QNH:** %d hPa", inHg, hPa)
}

func footer(network string, st config.Settings, logon, updated string) *discordgo.MessageEmbedFooter {
	t := tr(st)
	parts := []string{network}
	if logon != "" {
		parts = append(parts, t("footer.connected", logon))
	}
	if updated != "" {
		parts = append(parts, t("footer.updated", updated))
	}
	return &discordgo.MessageEmbedFooter{Text: strings.Join(parts, "  •  ")}
}
