package readme

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"golang.org/x/term"
)

//go:embed README.md
var README string

func Print() {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		for _, s := range strings.Split(README, "\n") {
			if len(s) > w {
				w = len(s)
			}
		}
	}
	result := markdown.Render(README, w, 0)
	fmt.Println(string(result))
}
