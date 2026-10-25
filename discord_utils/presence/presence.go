// Package presence keeps the bot's Discord status up to date with the number of
// pilots online on VATSIM and IVAO ("Watching 1234 pilots on VATSIM • 567 on IVAO").
package presence

import (
	"fmt"
	"log"
	"time"

	"github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	"github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/bwmarrin/discordgo"
)

const every = time.Minute // Discord limits presence updates, once a minute is plenty

// Start launches the background updater (call once, after the session is open
// and after the VATSIM / IVAO APIs have started).
func Start(s *discordgo.Session) {
	go func() {
		for {
			update(s)
			time.Sleep(every)
		}
	}()
}

func update(s *discordgo.Session) {
	v, i := vatsim.PilotCount(), ivao.PilotCount()
	if v == 0 && i == 0 { // feeds not loaded yet
		return
	}
	err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{{
			Name: fmt.Sprintf("%d pilots on VATSIM • %d on IVAO", v, i),
			Type: discordgo.ActivityTypeWatching,
		}},
	})
	if err != nil {
		log.Println("presence:", err)
	}
}
