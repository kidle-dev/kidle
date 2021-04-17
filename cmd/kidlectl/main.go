package main

import (
	"github.com/jessevdk/go-flags"
	"k8s.io/apimachinery/pkg/types"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// options are the cli main options for go-flags
type options struct {
	Kubeconfig string               `long:"kubeconfig" env:"KUBECONFIG" description:"path to Kubernetes config file"`
	IdleCmd    idleCommandOptions   `command:"idle" alias:"i" description:"idle the referenced object of an IdlingResource"`
	WakeUpCmd  wakeupCommandOptions `command:"wakeup" alias:"w" description:"wakeup the referenced object of an IdlingResource"`
}

// idleCommandOptions are the options of the idle command
type idleCommandOptions struct {
	PosArgs   positionalArgs `positional-args:"yes"`
	Namespace string         `long:"namespace" env:"NAMESPACE" required:"1" short:"n" description:"IdlingResource namespace"`
}

// wakeupCommandOptions are the options of the wakeup command
type wakeupCommandOptions struct {
	PosArgs   positionalArgs `positional-args:"yes"`
	Namespace string         `long:"namespace" env:"NAMESPACE" required:"1" short:"n" description:"IdlingResource namespace"`
}

type positionalArgs struct {
	Name string `long:"name" env:"NAME" required:"1" description:"idling resource name"`
}

func main() {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	// parse flags
	opts := &options{}
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

	kidle, err := NewKidleClient(opts.Kubeconfig)
	if err != nil {
		logf.Log.Error(err, "unable to create kidle client")
		os.Exit(2)
	}

	// execute active command
	switch p.Active.Name {
	case "idle":
		logf.Log.V(0).Info("idling", "namespace", opts.IdleCmd.Namespace, "name", opts.IdleCmd.PosArgs.Name)

		req := &types.NamespacedName{
			Namespace: opts.IdleCmd.Namespace,
			Name:      opts.IdleCmd.PosArgs.Name,
		}

		done, err := kidle.applyDesiredIdleState(true, req)
		if err != nil {
			logf.Log.Error(err, "unable to idle")
			os.Exit(3)
		}

		if done {
			logf.Log.V(0).Info("scaled to 0", "namespace", opts.IdleCmd.Namespace, "name", opts.IdleCmd.PosArgs.Name)
		} else {
			logf.Log.V(0).Info("already idled", "namespace", opts.IdleCmd.Namespace, "name", opts.IdleCmd.PosArgs.Name)
		}

	case "wakeup":
		logf.Log.V(0).Info("waking up", "namespace", opts.WakeUpCmd.Namespace, "name", opts.WakeUpCmd.PosArgs.Name)

		req := &types.NamespacedName{
			Namespace: opts.WakeUpCmd.Namespace,
			Name:      opts.WakeUpCmd.PosArgs.Name,
		}

		done, err := kidle.applyDesiredIdleState(true, req)
		if err != nil {
			logf.Log.Error(err, "unable to wake up")
			os.Exit(3)
		}

		if done {
			logf.Log.V(0).Info("waked up", "namespace", opts.WakeUpCmd.Namespace, "name", opts.WakeUpCmd.PosArgs.Name)
		} else {
			logf.Log.V(0).Info("already waked up", "namespace", opts.WakeUpCmd.Namespace, "name", opts.WakeUpCmd.PosArgs.Name)
		}
	}
}
