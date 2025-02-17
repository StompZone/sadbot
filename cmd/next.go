package cmd

import (
	"fmt"
	"github.com/stompzone/sadbot/utils"
)

// Next calls current guild's stream Next method. On success replies
// with current track name.
func Next(ctx Ctx) {
	err := requirePresence(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := ctx.stream().Next(); err != nil {
		utils.ErrorLogger.Println("Error nexting:", err)
		ctx.reply("Error nexting: " + err.Error())
	} else {
		ctx.reply("Now playing: " + ctx.stream().Current())
	}
}
