package cmd

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kidle-dev/kidle/cmd/kidlectl/pkg"
)

// WakeupCommandOptions are the options of the wakeup command
type WakeupCommandOptions struct {
	Args struct {
		Name string `long:"name" env:"NAME" description:"idling resource name to wakeup"`
	} `positional-args:"yes" required:"1"`
	Namespace string `long:"namespace" env:"NAMESPACE" short:"n" description:"IdlingResource namespace"`
}

// Wakeup executes the kidlectl wakeup command with given args
func Wakeup(opts WakeupCommandOptions) {
	kidle, err := pkg.NewKidleClient(opts.Namespace)
	if err != nil {
		logf.Log.Error(err, "unable to create kidle client")
		os.Exit(2)
	}
	logf.Log.V(0).Info("waking up", "namespace", kidle.Namespace, "name", opts.Args.Name)

	done, err := kidle.ApplyDesiredIdleState(false, &types.NamespacedName{
		Namespace: kidle.Namespace,
		Name:      opts.Args.Name,
	})
	if err != nil {
		logf.Log.Error(err, "unable to wake up")
		os.Exit(3)
	}

	if done {
		logf.Log.V(0).Info("woke up", "namespace", kidle.Namespace, "name", opts.Args.Name)
	} else {
		logf.Log.V(0).Info("already woke up", "namespace", kidle.Namespace, "name", opts.Args.Name)
	}
}
