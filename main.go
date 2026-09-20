// Command agent-skills-validator validates skill directories against the
// public Agent Skills specification.
package main

import (
	"os"

	"github.com/coolapso/agent-skills-validator/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
