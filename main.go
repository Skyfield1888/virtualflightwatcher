package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	"github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/Skyfield1888/virtualflightwatcher/discord_utils/commands"
	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}

	if err = discord.Open(); err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commands.CommandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	for _, cmd := range commands.Commands {
		discord.ApplicationCommandCreate(discord.State.User.ID, "", cmd)
	}

	vatsim.StartVatsimApi()
	ivao.StartIvaoApi()
	defer discord.Close()

	fmt.Println("Bot is running. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

}
