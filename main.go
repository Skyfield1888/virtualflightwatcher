package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Skyfield1888/virtualflightwatcher/api/geo"
	"github.com/Skyfield1888/virtualflightwatcher/api/ivao"
	"github.com/Skyfield1888/virtualflightwatcher/api/vatsim"
	"github.com/Skyfield1888/virtualflightwatcher/config"
	"github.com/Skyfield1888/virtualflightwatcher/discord_utils/commands"
	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	settingsPath := os.Getenv("SETTINGS_PATH")
	if settingsPath == "" {
		settingsPath = "data/settings.json"
	}
	if err := config.Load(settingsPath); err != nil {
		fmt.Println("Error loading settings:", err)
		return
	}

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

	go func() {
		if err := geo.LoadNavdata("navdata"); err != nil {
			fmt.Println("Navdata load error:", err)
		}
	}()

	discord.AddHandler(commands.Dispatch)

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
