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

	// execute active command
	switch p.Active.Name {
	case "idle":
		kidle, err := NewKidleClient(opts.IdleCmd.Namespace)
		if err != nil {
			logf.Log.Error(err, "unable to create kidle client")
			os.Exit(2)
		}
		logf.Log.V(0).Info("idling", "namespace", kidle.Namespace, "name", opts.IdleCmd.Args.Name)

		done, err := kidle.applyDesiredIdleState(true, &types.NamespacedName{
			Namespace: kidle.Namespace,
			Name:      opts.IdleCmd.Args.Name,
		})
		if err != nil {
			logf.Log.Error(err, "unable to idle")
			os.Exit(3)
		}

		if done {
			logf.Log.V(0).Info("scaled to 0", "namespace", kidle.Namespace, "name", opts.IdleCmd.Args.Name)
		} else {
			logf.Log.V(0).Info("already idled", "namespace", kidle.Namespace, "name", opts.IdleCmd.Args.Name)
		}

	case "wakeup":
		kidle, err := NewKidleClient(opts.WakeUpCmd.Namespace)
		if err != nil {
			logf.Log.Error(err, "unable to create kidle client")
			os.Exit(2)
		}
		logf.Log.V(0).Info("waking up", "namespace", kidle.Namespace, "name", opts.WakeUpCmd.Args.Name)

		done, err := kidle.applyDesiredIdleState(false, &types.NamespacedName{
			Namespace: kidle.Namespace,
			Name:      opts.WakeUpCmd.Args.Name,
		})
		if err != nil {
			logf.Log.Error(err, "unable to wake up")
			os.Exit(3)
		}

		if done {
			logf.Log.V(0).Info("waked up", "namespace", kidle.Namespace, "name", opts.WakeUpCmd.Args.Name)
		} else {
			logf.Log.V(0).Info("already waked up", "namespace", kidle.Namespace, "name", opts.WakeUpCmd.Args.Name)
		}
	}
}
