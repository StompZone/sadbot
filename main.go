package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/stompzone/sadbot/session"
	"github.com/stompzone/sadbot/utils"
)

func main() {
	// Initialize logger
	utils.InitLog()

	// parse config file
	config, err := utils.GetConfig()
	if err != nil {
		utils.ErrorLogger.Fatalf("Failed to parse config file: %s", err)
	}

	// Create new Discord Session
	session, err := session.OpenSession("Bot "+config.Token, config.Prefix)
	if err != nil {
		utils.ErrorLogger.Fatalf("Failed to create a discord session: %s", err)
	}

	// ensure that session will be gracefully closed whenever the function exits
	defer session.Close()

	// run until code is terminated
	utils.InfoLogger.Println("sadbot is now running. Press Ctrl-C to exit.")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-c
}
