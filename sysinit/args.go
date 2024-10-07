package sysinit

import (
	"log"
	"os"
	"strings"

	"github.com/kballard/go-shellquote"
)

func ParseArgs(args []string) {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		log.Print("cmdline", err)
		return
	}

	words, err := shellquote.Split(string(data))
	if err != nil {
		log.Print("cmdline", err)
		return
	}

	for _, word := range words {
		key, val, opt := strings.Cut(word, "=")
		if opt {
			Args[key] = val
		} else {
			Args[key] = key
		}
	}

	for _, arg := range args {
		key, val, _ := strings.Cut(arg, "=")
		if strings.HasPrefix(key, "--") {
			Args[key[2:]] = val
		}
	}
}
