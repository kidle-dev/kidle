package cmd

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kidle-dev/kidle/cmd/kidlectl/pkg"
)

// CreateCommandOptions are the options of the create command
type CreateCommandOptions struct {
	Args struct {
		Name string `long:"name" env:"NAME" description:"idling resource name to wakeup"`
	} `positional-args:"yes" required:"1"`
	Namespace string `long:"namespace" env:"NAMESPACE" short:"n" description:"IdlingResource namespace"`
	Idle      bool   `long:"idle" env:"IDLE" short:"i" description:"the desired state of idling, defaults to false"`
	Ref       string `long:"ref" env:"IDLE" short:"r" description:"the reference to the idle-able workload"`
}

// Create executes the kidlectl create command with given args
func Create(opts CreateCommandOptions) {
	// create a new kidle client
	kidle, err := pkg.NewKidleClient(opts.Namespace)
	if err != nil {
		logf.Log.Error(err, "unable to create kidle client")
		os.Exit(2)
	}
	logf.Log.V(0).Info("creating the idling resource", "namespace", kidle.Namespace, "name", opts.Args.Name, "ref", opts.Ref)

	// create an IdlingResource
	done, err := kidle.CreateIdlingResource(opts.Idle, opts.Ref, &types.NamespacedName{
		Namespace: kidle.Namespace,
		Name:      opts.Args.Name,
	})
	if err != nil {
		logf.Log.Error(err, "unable to create an idling resource")
		os.Exit(3)
	}

	// handle execution result
	if done {
		logf.Log.V(0).Info("idling resource created", "namespace", kidle.Namespace, "name", opts.Args.Name, "ref", opts.Ref)
	} else {
		logf.Log.V(0).Info("creation of the idling resource failed", "namespace", kidle.Namespace, "name", opts.Args.Name, "ref", opts.Ref)
	}
}
