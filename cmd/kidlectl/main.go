package main

import (
	"github.com/kidle-dev/kidle/cmd/kidlectl/cmd"
	"os"

	"github.com/jessevdk/go-flags"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Options are the cli main options for go-flags
type Options struct {
	Kubeconfig string                    `long:"kubeconfig" env:"KUBECONFIG" description:"path to Kubernetes config file"`
	IdleCmd    cmd.IdleCommandOptions    `command:"idle" alias:"i" description:"idle the referenced object of an IdlingResource"`
	WakeUpCmd  cmd.WakeupCommandOptions  `command:"wakeup" alias:"w" description:"wakeup the referenced object of an IdlingResource"`
	CreateCmd  cmd.CreateCommandOptions  `command:"create" alias:"c" description:"create an IdlingResource"`
	VersionCmd cmd.VersionCommandOptions `command:"version" description:"show the kidle version information"`
}

func main() {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	// parse flags
	opts := &Options{}
	p := flags.NewParser(opts, flags.Default)
	_, err := p.Parse()

	if err != nil {
		if flagErr, ok := err.(*flags.Error); ok {
			if flagErr.Type == flags.ErrCommandRequired || flagErr.Type == flags.ErrRequired {
				p.WriteHelp(os.Stdout)
			}
		}
		os.Exit(1)
	}

	// execute active command
	switch p.Active.Name {
	case "idle":
		cmd.Idle(opts.IdleCmd)
	case "wakeup":
		cmd.Wakeup(opts.WakeUpCmd)
	case "create":
		cmd.Create(opts.CreateCmd)
	case "version":
		cmd.Version()
	}
}
