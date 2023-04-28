// Package watcher is a command line tool inspired by fresh (https://github.com/pilu/fresh) and used
// for watching .go file changes, and restarting the app in case of an update/delete/add operation.
// After you installed it, you can run your apps with their default parameters as:
// watcher -c config -p 7000 -h localhost
package watcher

import (
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/fatih/color"
)

// Runner listens for the change events and depending on that kills
// the obsolete process, and runs a new one
type Runner struct {
	start     chan string
	done      chan struct{}
	cmd       *exec.Cmd
	interrupt bool
}

// NewRunner creates a new Runner instance and returns its pointer
func NewRunner(params *Params) *Runner {
	var interrupt bool
	var err error
	flagInterrupt := params.Get("interrupt-signal")

	if flagInterrupt != "" {
		interrupt, err = strconv.ParseBool(params.Get("interrupt-signal"))
		if err != nil {
			log.Println("interrupt-signal should be 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False")
			os.Exit(1)
		}
	}

	return &Runner{
		start:     make(chan string),
		done:      make(chan struct{}),
		interrupt: interrupt,
	}
}

// Run initializes runner with given parameters.
func (r *Runner) Run(p *Params) {
	for fileName := range r.start {

		color.Green("Running %s...\n", p.Get("run"))

		cmd, err := runCommand(fileName, p.Package...)
		if err != nil {
			log.Printf("Could not run the go binary: %s \n", err)
			r.stop(cmd)

			continue
		}

		r.cmd = cmd
		removeFile(fileName)

		go func(cmd *exec.Cmd) {
			if err := cmd.Wait(); err != nil {
				log.Printf("process interrupted: %s \n", err)
				r.stop(cmd)
			}
		}(r.cmd)
	}
}

// Restart kills or interrupt the process, removes the old binary and
// restarts the new process
func (r *Runner) restart(fileName string) {
	r.stop(r.cmd)

	r.start <- fileName
}

func (r *Runner) stop(cmd *exec.Cmd) {
	if cmd != nil {
		if r.interrupt {
			log.Println("sending interrupt signal...")
			cmd.Process.Signal(os.Interrupt)
		} else {
			cmd.Process.Kill()
		}
	}
}

func (r *Runner) Close() {
	close(r.start)
	r.stop(r.cmd)
	close(r.done)
}

func (r *Runner) Wait() {
	<-r.done
}
