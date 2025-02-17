package cmd

import (
	"time"
	"github.com/stompzone/sadbot/utils"
	"strings"
)

// Play joins bot to voice if it is not in one. If no arguments passed calls
// current stream Unpause method. Otherwise processes given query, calls current
// stream Add method for each processed track. When done replies with currnet
// queue. Then calls for Play method.
func Play(ctx Ctx) {
	// Get the voice state for the given guild and user
	_, err := ctx.S.State.VoiceState(ctx.M.GuildID, ctx.M.Author.ID)

	// if err means user is not connected to a voice channel
	if err != nil {
		ctx.reply("Must be connected to voice channel to use bot")
		return
	}

	// join voice in case bot is not in one
	if ctx.stream().V == nil {
		err := joinWithRetry(ctx, 3)
		if err != nil {
			utils.ErrorLogger.Println("Failed to join voice channel:", err)
			return
		}
	}

	args := strings.TrimSpace(ctx.Args)

	if args == "" {
		ctx.stream().Unpause()
		return
	}

	res, err := utils.ProcessQuery(args)
	if err != nil {
		ctx.reply(err.Error())
	}

	for _, t := range res {
		ctx.stream().Add(t.Url, t.Title)
	}

	go Queue(ctx)

	if err := ctx.stream().Play(); err != nil {
		utils.ErrorLogger.Println("Error streaming:", err)
		ctx.reply("Error streaming: " + err.Error())
	}
}

// joinWithRetry attempts to join the voice channel with retries.
func joinWithRetry(ctx Ctx, retries int) error {
	for i := 0; i < retries; i++ {
		err := join(ctx)
		if err == nil {
			return nil
		}
		utils.ErrorLogger.Printf("Failed to join voice channel (attempt %d/%d): %v", i+1, retries, err)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for voice")
}
