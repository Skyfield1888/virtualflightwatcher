// Package events mirrors VATSIM / IVAO events into Discord scheduled events.
package events

import (
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	"github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/Skyfield1888/virtualflightwatcher/config"
	"github.com/Skyfield1888/virtualflightwatcher/i18n"
	"github.com/bwmarrin/discordgo"
)

const (
	syncEvery = 15 * time.Minute
	horizon   = 14 * 24 * time.Hour // only announce events starting within 2 weeks (Discord caps at 100 per server)
	maxName   = 100
	maxDesc   = 1000
	maxLoc    = 100
)

type Event struct {
	Key, Network, Name, Summary, Link string
	Airports, Organisers              []string
	Start, End                        time.Time
}

// Every created Discord event ends with "ref:<key>", which is how we detect already-created events.
var refRe = regexp.MustCompile(`ref:((?:vatsim|ivao)-[\w-]+)`)

var (
	cacheMu  sync.Mutex
	cached   []Event
	cachedAt time.Time

	guildMu sync.Mutex // serialises create / delete passes so they never fight each other
)

// Start launches the background sync (call once, after the Discord session is open).
func Start(s *discordgo.Session) {
	go func() {
		time.Sleep(20 * time.Second) // let the bot finish starting
		for {
			SyncAll(s)
			time.Sleep(syncEvery)
		}
	}()
}

func SyncAll(s *discordgo.Session) {
	guilds := config.EventGuilds()
	if len(guilds) == 0 {
		return
	}
	evs := fetchAll()
	for _, g := range guilds {
		syncGuild(s, g, evs)
	}
}

// ApplyGuild is called right after /config changed the events setting of a server:
// it deletes the bot's events of the networks that are no longer wanted (all of them when
// the mode is "none"), then creates the missing events of the wanted ones.
func ApplyGuild(s *discordgo.Session, guildID string) {
	pruneGuild(s, guildID)
	syncGuild(s, guildID, fetchAll())
}

func fetchAll() []Event {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cached != nil && time.Since(cachedAt) < 2*time.Minute {
		return cached
	}

	var out []Event
	if list, err := vatsim.FetchEvents(); err != nil {
		log.Println("events: vatsim:", err)
	} else {
		for _, e := range list {
			ev := Event{
				Key: fmt.Sprintf("vatsim-%d", e.Id), Network: config.NetworkVatsim, Name: e.Name,
				Summary: e.ShortDescription, Link: e.Link, Start: e.StartTime, End: e.EndTime,
			}
			if ev.Summary == "" {
				ev.Summary = e.Description
			}
			for _, a := range e.Airports {
				ev.Airports = append(ev.Airports, a.Icao)
			}
			for _, o := range e.Organisers {
				for _, v := range []string{o.OrganisationCode, o.Subdivision, o.Division, o.Region} {
					if v != "" {
						ev.Organisers = append(ev.Organisers, v)
						break
					}
				}
			}
			out = append(out, ev)
		}
	}
	if list, err := ivao.FetchEvents(); err != nil {
		log.Println("events: ivao:", err)
	} else {
		for _, e := range list {
			out = append(out, Event{
				Key: "ivao-" + e.Id, Network: config.NetworkIvao, Name: e.Title, Summary: e.Description,
				Link: e.Link, Airports: e.Airports, Start: e.Start, End: e.End,
			})
		}
	}

	now := time.Now()
	var keep []Event
	for _, e := range out {
		if e.Start.After(now.Add(time.Minute)) && e.Start.Before(now.Add(horizon)) {
			keep = append(keep, e)
		}
	}
	cached, cachedAt = keep, now
	return keep
}

// networkOf returns the network of a "vatsim-123" / "ivao-456" key.
func networkOf(key string) string {
	switch {
	case strings.HasPrefix(key, "vatsim-"):
		return config.NetworkVatsim
	case strings.HasPrefix(key, "ivao-"):
		return config.NetworkIvao
	}
	return ""
}

// pruneGuild deletes the events created by this bot whose network is not wanted (anymore).
// Events created by humans are never touched (creator check + "ref:" marker).
func pruneGuild(s *discordgo.Session, guildID string) {
	guildMu.Lock()
	defer guildMu.Unlock()

	mode := config.Get(guildID).Events
	existing, err := s.GuildScheduledEvents(guildID, false)
	if err != nil {
		log.Printf("events: guild %s: list failed: %v", guildID, err)
		return
	}
	for _, e := range existing {
		if e.CreatorID != s.State.User.ID {
			continue
		}
		m := refRe.FindStringSubmatch(e.Description)
		if m == nil || mode.Wants(networkOf(m[1])) {
			continue
		}
		if err := s.GuildScheduledEventDelete(guildID, e.ID); err != nil {
			log.Printf("events: guild %s: delete %s failed: %v", guildID, e.ID, err)
			continue
		}
		time.Sleep(1 * time.Second) // stay far from rate limits
	}
}

func syncGuild(s *discordgo.Session, guildID string, evs []Event) {
	st := config.Get(guildID)
	var want []Event
	for _, e := range evs {
		if st.Events.Wants(e.Network) {
			want = append(want, e)
		}
	}
	if len(want) == 0 {
		return
	}

	guildMu.Lock()
	defer guildMu.Unlock()

	existing, err := s.GuildScheduledEvents(guildID, false)
	if err != nil {
		log.Printf("events: guild %s: list failed: %v", guildID, err)
		return
	}
	have := map[string]bool{}
	for _, e := range existing {
		if m := refRe.FindStringSubmatch(e.Description); m != nil {
			have[m[1]] = true
		}
	}

	t := func(key string, args ...any) string { return i18n.T(st.Language, key, args...) }
	for _, ev := range want {
		if have[ev.Key] {
			continue
		}
		// The setting may have been changed while this pass was running.
		if !config.Get(guildID).Events.Wants(ev.Network) {
			continue
		}
		end := ev.End
		if !end.After(ev.Start) {
			end = ev.Start.Add(2 * time.Hour) // external events require an end time
		}
		start := ev.Start
		loc := ev.Network + " (online)"
		if len(ev.Airports) > 0 {
			loc = strings.Join(ev.Airports, ", ")
		}
		_, err := s.GuildScheduledEventCreate(guildID, &discordgo.GuildScheduledEventParams{
			Name:               cut("["+ev.Network+"] "+stripHTML(ev.Name), maxName),
			Description:        describe(ev, t),
			ScheduledStartTime: &start,
			ScheduledEndTime:   &end,
			PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
			EntityType:         discordgo.GuildScheduledEventEntityTypeExternal,
			EntityMetadata:     &discordgo.GuildScheduledEventEntityMetadata{Location: cut(loc, maxLoc)},
		})
		if err != nil {
			// Most likely missing the "Manage Events" permission.
			log.Printf("events: guild %s: create %s failed: %v", guildID, ev.Key, err)
			return
		}
		time.Sleep(2 * time.Second) // stay far from rate limits
	}
}

func describe(ev Event, t func(string, ...any) string) string {
	var tail strings.Builder
	if len(ev.Airports) > 0 {
		tail.WriteString("**" + t("ev.airports") + ":** " + strings.Join(ev.Airports, ", ") + "\n")
	}
	if len(ev.Organisers) > 0 {
		tail.WriteString("**" + t("ev.organisers") + ":** " + strings.Join(ev.Organisers, ", ") + "\n")
	}
	if ev.Link != "" {
		tail.WriteString(t("ev.more_info") + ": " + ev.Link + "\n")
	}
	tail.WriteString("ref:" + ev.Key)

	budget := maxDesc - len([]rune(tail.String())) - 2
	summary := strings.TrimSpace(cut(stripHTML(ev.Summary), budget))
	if summary == "" {
		return tail.String()
	}
	return summary + "\n\n" + tail.String()
}

var (
	brRe      = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|h[1-6]|ul|ol|tr)>`)
	liRe      = regexp.MustCompile(`(?i)<li[^>]*>`)
	tagRe     = regexp.MustCompile(`(?s)<[^>]*>`)
	spacesRe  = regexp.MustCompile(`[ \t\x{00a0}]+`)
	newlineRe = regexp.MustCompile(`\n[ \t]*(\n[ \t]*)+`)
)

// stripHTML turns the HTML of the VATSIM / IVAO descriptions into plain text.
func stripHTML(s string) string {
	s = brRe.ReplaceAllString(s, "\n")
	s = liRe.ReplaceAllString(s, "\n• ")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s) // &amp; &nbsp; &#39; ...
	s = spacesRe.ReplaceAllString(s, " ")
	s = newlineRe.ReplaceAllString(s, "\n\n") // at most one blank line
	return strings.TrimSpace(s)
}

func cut(s string, n int) string {
	r := []rune(s)
	if n <= 0 {
		return ""
	}
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
