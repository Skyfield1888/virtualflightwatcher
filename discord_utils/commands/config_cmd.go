package commands

import (
	"fmt"

	"github.com/Skyfield1888/virtualflightwatcher/config"
	"github.com/Skyfield1888/virtualflightwatcher/discord_utils/events"
	"github.com/Skyfield1888/virtualflightwatcher/i18n"
	"github.com/bwmarrin/discordgo"
)

func init() {
	Commands = append(Commands, configCommand())
	CommandHandlers["config"] = handleConfig
}

func configCommand() *discordgo.ApplicationCommand {
	// Hidden by default for members without "Manage Server" (admins can still change it in Discord's UI).
	perm := int64(discordgo.PermissionManageGuild)

	var langChoices []*discordgo.ApplicationCommandOptionChoice
	for _, l := range i18n.Supported {
		langChoices = append(langChoices, &discordgo.ApplicationCommandOptionChoice{Name: l.Name, Value: string(l.Code)})
	}

	return &discordgo.ApplicationCommand{
		Name:                     "config",
		Description:              "Configure the bot for this server",
		DefaultMemberPermissions: &perm,
		// Every option is optional: no option = show the configuration, several options = change them all at once.
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString, Name: "language",
				Description: "Bot language", Choices: langChoices,
			},
			{
				Type: discordgo.ApplicationCommandOptionBoolean, Name: "ephemeral",
				Description: "Make replies visible only to the person who ran the command",
			},
			{
				Type: discordgo.ApplicationCommandOptionString, Name: "units",
				Description: "Units used for altitude, speed, pressure and distance",
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Aviation (ft, kts, inHg, nm)", Value: config.UnitsAviation},
					{Name: "Metric (m, km/h, hPa, km)", Value: config.UnitsMetric},
					{Name: "Imperial (ft, mph, inHg, mi)", Value: config.UnitsImperial},
				},
			},
			{
				Type: discordgo.ApplicationCommandOptionChannel, Name: "channel",
				Description:  "Restrict bot commands to this channel",
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
			},
			{
				Type: discordgo.ApplicationCommandOptionBoolean, Name: "all_channels",
				Description: "Allow bot commands in every channel again (removes the channel restriction)",
			},
			{
				Type: discordgo.ApplicationCommandOptionString, Name: "events",
				Description: "Create Discord events for the upcoming events of a network (or none)",
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "VATSIM", Value: string(config.EventsVatsim)},
					{Name: "IVAO", Value: string(config.EventsIvao)},
					{Name: "VATSIM + IVAO", Value: string(config.EventsBoth)},
					{Name: "None (remove the events created by the bot)", Value: string(config.EventsNone)},
				},
			},
			{
				Type: discordgo.ApplicationCommandOptionBoolean, Name: "reset",
				Description: "Reset every setting to its default value (applied before the other options)",
			},
		},
	}
}

func handleConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// In a server, require "Manage Server". In DMs, settings only affect the user themself.
	if i.Member != nil && i.Member.Permissions&discordgo.PermissionManageGuild == 0 {
		respondError(s, i, tr(SettingsOf(i))("cfg.no_perm"))
		return
	}

	opts := getOptions(i)

	// No argument: just show the configuration.
	if len(opts) == 0 {
		send(s, i, configEmbed(i), true)
		return
	}

	key := settingsKey(i)
	resetOnly := len(opts) == 1 && opts["reset"] != nil

	eventsBefore := config.Get(key).Events

	var err error
	if o, ok := opts["reset"]; ok && o.BoolValue() {
		err = config.Reset(key)
	}
	if err == nil {
		err = config.Update(key, func(c *config.Settings) {
			if o, ok := opts["language"]; ok {
				if lang := i18n.Lang(o.StringValue()); i18n.Valid(lang) {
					c.Language = lang
				}
			}
			if o, ok := opts["ephemeral"]; ok {
				c.Ephemeral = o.BoolValue()
			}
			if o, ok := opts["units"]; ok {
				c.Units = o.StringValue()
			}
			if o, ok := opts["all_channels"]; ok && o.BoolValue() {
				c.AllowedChannelID = ""
			}
			if o, ok := opts["channel"]; ok {
				c.AllowedChannelID, _ = o.Value.(string) // channel options carry the channel ID
			}
			if o, ok := opts["events"]; ok {
				if mode := config.EventsMode(o.StringValue()); mode.Valid() {
					c.Events = mode
				}
			}
		})
	}
	if err != nil {
		fmt.Println("config error:", err)
		respondError(s, i, err.Error())
		return
	}

	// Events mode changed in a server: remove the bot's events of the networks no longer wanted and
	// create the missing ones right away. Comparing before/after also covers "reset".
	if i.GuildID != "" && config.Get(key).Events != eventsBefore {
		go events.ApplyGuild(s, i.GuildID)
	}

	// Re-read settings so the confirmation is already in the newly selected language.
	t := tr(SettingsOf(i))
	msg := t("cfg.saved")
	if resetOnly {
		msg = t("cfg.reset")
	}
	embed := configEmbed(i)
	embed.Description = msg + "\n\n" + embed.Description
	send(s, i, embed, true)
}

func configEmbed(i *discordgo.InteractionCreate) *discordgo.MessageEmbed {
	st := SettingsOf(i)
	t := tr(st)

	lang := string(st.Language)
	for _, l := range i18n.Supported {
		if l.Code == st.Language {
			lang = l.Name
		}
	}

	onOff := t("cfg.off")
	if st.Ephemeral {
		onOff = t("cfg.on")
	}

	evOnOff := t("cfg.off") // "none"
	switch st.Events {
	case config.EventsVatsim:
		evOnOff = "VATSIM"
	case config.EventsIvao:
		evOnOff = "IVAO"
	case config.EventsBoth:
		evOnOff = "VATSIM + IVAO"
	}

	channel := t("cfg.all_channels")
	if st.AllowedChannelID != "" {
		channel = "<#" + st.AllowedChannelID + ">"
	}

	scope := t("cfg.scope.guild")
	if i.GuildID == "" {
		scope = t("cfg.scope.user")
	}

	return &discordgo.MessageEmbed{
		Title:       "⚙️  " + t("cfg.title"),
		Description: "*" + scope + "*",
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: t("cfg.language"), Value: lang, Inline: true},
			{Name: t("cfg.ephemeral"), Value: onOff, Inline: true},
			{Name: t("cfg.units"), Value: t("cfg.units." + st.Units), Inline: true},
			{Name: t("cfg.channel"), Value: channel, Inline: true},
			{Name: t("cfg.events"), Value: evOnOff, Inline: true},
		},
	}
}
