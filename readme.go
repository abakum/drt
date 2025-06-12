package readme

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

//go:embed README.md
var README string

func Print() {
	i := 0
	for _, s := range strings.Split(README, "\n") {
		if len(s) > i {
			i = len(s)
		}
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = i
	}
	r, _ := glamour.NewTermRenderer(
		// detect background color and pick either the default dark or light theme
		// glamour.WithAutoStyle(),
		glamour.WithStandardStyle("dracula"),
		// wrap output at specific width (default is 80)
		glamour.WithWordWrap(w-2),
	)
	result, _ := r.Render(README)
	fmt.Println(result)
}
