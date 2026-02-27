package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Skyfield1888/Vatsim/commands"

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

	// Enregistrer les commandes auprès de Discord
	for _, cmd := range commands.Commands {
		discord.ApplicationCommandCreate(discord.State.User.ID, "", cmd)
		// _, err := discord.ApplicationCommandCreate(discord.State.User.ID, "1407389812687769712", cmd)
		// if err != nil {
		// 	fmt.Println("Erreur enregistrement commande:", err) // log l'erreur !
		// }
	}

	defer discord.Close()

	fmt.Println("Bot is running. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
