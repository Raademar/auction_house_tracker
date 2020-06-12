package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/raademar/auction_house_tracker/config"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type AuctionItem struct {
	Slug       string `json:"slug"`
	ItemID     int    `json:"itemId"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Timerange  int    `json:"timerange"`
	Data       []struct {
		MarketValue int       `json:"marketValue"`
		MinBuyout   int       `json:"minBuyout"`
		Quantity    int       `json:"quantity"`
		ScannedAt   time.Time `json:"scannedAt"`
	} `json:"data"`
}

func goDotEnvVariable(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return os.Getenv(key)
}

func main() {
	// Load the bot token from .env
	Token := goDotEnvVariable("DISCORD_BOT_TOKEN")
	// BlizzardAPIToken := goDotEnvVariable("BLIZZARD_API_TOKEN")

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	channel, err := s.State.Channel(m.ChannelID)
	userChannel, err := s.UserChannelCreate(m.Author.ID)
	// Add the newly created userChannel to the global state
	s.State.ChannelAdd(userChannel)

	if m.Content == "!track" {
		if err != nil {
			log.Fatal("Could not find the channel,", err)
			return
		}
		userStateChannel, err := s.State.Channel(userChannel.ID)
		if err != nil {
			log.Fatal("Could not find the channel,", err)
			return
		}

		s.ChannelMessageSend(userStateChannel.ID, "What item would you like to track?")
	}

	if channel.ID == userChannel.ID {
		fmt.Println(m.Content)
		searchQuery := strings.ReplaceAll(m.Content, " ", "-")
		res, err := http.Get(config.NexusHubAPI + "items/ashbringer-horde/" + searchQuery + "/prices")
		if err != nil {
			log.Fatal("Error getting the thing")
		}
		defer res.Body.Close()

		var auctionItem AuctionItem
		err = json.NewDecoder(res.Body).Decode(&auctionItem)
		if err != nil {
			log.Fatal(err)
		}
		currentPrice := auctionItem.Data[0]
		formattedPrice := strconv.Itoa(currentPrice.MinBuyout)
		lengthOfPrice := len(formattedPrice)
		copperFormat := formattedPrice[lengthOfPrice-2 : lengthOfPrice]
		silverFormat := formattedPrice[lengthOfPrice-4 : lengthOfPrice-2]
		goldFormat := formattedPrice[0 : lengthOfPrice-4]
		s.ChannelMessageSend(channel.ID, "Cheapest auction of "+auctionItem.Name+" is currently at "+goldFormat+"g "+silverFormat+"s "+copperFormat+"c")

	}

}
