package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Options are the cli main options for go-flags
type Options struct {
	Kubeconfig   string                  `long:"kubeconfig" env:"KUBECONFIG" description:"path to Kubernetes config file"`
	IdleCmd      idleCommandOptions      `command:"idle" alias:"i" description:"idle the referenced object of an IdlingResource"`
	WakeUpCmd    wakeupCommandOptions    `command:"wakeup" alias:"w" description:"wakeup the referenced object of an IdlingResource"`
	CreateCmd    createCommandOptions    `command:"create" alias:"c" description:"create an IdlingResource"`
	VersionCmd   versionCommandOptions   `command:"version" description:"show the kidle version information"`
}

// idleCommandOptions are the options of the idle command
type idleCommandOptions struct {
	Args struct {
		Name string `long:"name" env:"NAME" description:"idling resource name to idle"`
	} `positional-args:"yes" required:"1"`
	Namespace string `long:"namespace" env:"NAMESPACE" short:"n" description:"IdlingResource namespace"`
}

// wakeupCommandOptions are the options of the wakeup command
type wakeupCommandOptions struct {
	Args struct {
		Name string `long:"name" env:"NAME" description:"idling resource name to wakeup"`
	} `positional-args:"yes" required:"1"`
	Namespace string `long:"namespace" env:"NAMESPACE" short:"n" description:"IdlingResource namespace"`
}

// createCommandOptions are the options of the create command
type createCommandOptions struct {
	Args struct {
		Name string `long:"name" env:"NAME" description:"idling resource name to wakeup"`
	} `positional-args:"yes" required:"1"`
	Namespace string `long:"namespace" env:"NAMESPACE" short:"n" description:"IdlingResource namespace"`
	Idle      bool   `long:"idle" env:"IDLE" short:"i" description:"the desired state of idling, defaults to false"`
	Ref       string `long:"ref" env:"IDLE" short:"r" description:"the reference to the idle-able workload"`
}

// versionCommandOptions are the options of the version command
type versionCommandOptions struct {
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
		cmdIdle(opts.IdleCmd)
	case "wakeup":
		cmdWakeup(opts.WakeUpCmd)
	case "create":
		cmdCreate(opts.CreateCmd)
	case "version":
		cmdVersion()
	}
}
