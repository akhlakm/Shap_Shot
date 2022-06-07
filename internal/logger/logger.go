package logger

import (
	"fmt"
	"os"
)

const trace bool = false
const info bool = false

func Trace(process string, item string) {
	if trace {
		line := fmt.Sprintf("%s < %s ...", process, shorten_message(item))
		fmt.Println(line)
	}
}

func Done(process string, item string) {
	if info {
		line := fmt.Sprintf("OK  -- %s > %s", process, shorten_message(item))
		fmt.Println(line)
	}
}

func Info(item string) {
	if info {
		line := fmt.Sprintf("\t%s", shorten_message(item))
		fmt.Println(line)
	}
}

func Error(process string, item string, message string) {
	line := fmt.Sprintf("ERR -- %s > %s", process, shorten_message(item))
	fmt.Println(line)
	fmt.Println("       " + message)
	os.Exit(10)
}

func Print(message string) {
	fmt.Println(message)
}

func shorten_message(msg string) string {
	maxlen := 200
	lefthalf := int(maxlen / 2)
	if len(msg) > maxlen {
		msg = msg[:lefthalf] + "..." + msg[len(msg)-lefthalf:]
	}
	return msg
}
