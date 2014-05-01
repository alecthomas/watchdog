package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/alecthomas/flagutil"
	"github.com/alecthomas/pflag"
	"github.com/howeyc/fsnotify"
)

var (
	pathFlag  = pflag.StringP("path", "p", ".", "path to watch for changes")
	waitFlag  = pflag.DurationP("wait", "w", time.Millisecond*500, "duration to wait after changes before executing command")
	matchFlag = pflag.StringP("match", "m", "*", "changed files must match this glob pattern")
)

func main() {
	pflag.Usage = flagutil.MakeUsage("usage: watchdog <command>", "")
	pflag.Parse()

	if len(pflag.Args()) == 0 {
		flagutil.UsageErrorf("missing command")
	}

	command := pflag.Args()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		flagutil.Fatalf("failed to start fsnotify: %s", err)
	}

	done := make(chan bool)

	go func() {
		timer := time.NewTimer(time.Hour)
		timer.Stop()

		for {
			select {
			case <-timer.C:
				cmd := exec.Command(command[0], command[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				if err := cmd.Run(); err != nil {
					flagutil.Fatalf("failed to exec command: %s", err)
				}
				timer.Stop()

			case ev := <-watcher.Event:
				match, err := path.Match(*matchFlag, ev.Name)
				if err != nil {
					flagutil.Fatalf("match failed against %s: %s", *matchFlag, err)
				}
				if !ev.IsAttrib() && match {
					timer.Reset(*waitFlag)
				}

			case err := <-watcher.Error:
				fmt.Println("error:", err)
				done <- true
				return
			}
		}
	}()

	err = watcher.Watch(*pathFlag)
	if err != nil {
		flagutil.Fatalf("failed to watch path %s: %s", *pathFlag, err)
	}

	<-done

	watcher.Close()
}
