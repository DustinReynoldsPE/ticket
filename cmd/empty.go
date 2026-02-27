package cmd

import (
	_ "embed"
	"fmt"
	"math/rand"
	"strings"
)

//go:embed messages.txt
var messagesRaw string

func printEmptyMessage() {
	if jsonOutput {
		fmt.Println("[]")
		return
	}

	lines := strings.Split(strings.TrimSpace(messagesRaw), "\n")
	if len(lines) == 0 {
		return
	}
	fmt.Println(lines[rand.Intn(len(lines))])
}
