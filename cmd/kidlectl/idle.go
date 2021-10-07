package main

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kidle-dev/kidle/cmd/kidlectl/pkg"
)

// cmdIdle executes the kidlectl idle command with given args
func cmdIdle(opts idleCommandOptions) {
	kidle, err := pkg.NewKidleClient(opts.Namespace)
	if err != nil {
		logf.Log.Error(err, "unable to create kidle client")
		os.Exit(2)
	}
	logf.Log.V(0).Info("idling", "namespace", kidle.Namespace, "name", opts.Args.Name)

	done, err := kidle.ApplyDesiredIdleState(true, &types.NamespacedName{
		Namespace: kidle.Namespace,
		Name:      opts.Args.Name,
	})
	if err != nil {
		logf.Log.Error(err, "unable to idle")
		os.Exit(3)
	}

	if done {
		logf.Log.V(0).Info("scaled to 0", "namespace", kidle.Namespace, "name", opts.Args.Name)
	} else {
		logf.Log.V(0).Info("already idled", "namespace", kidle.Namespace, "name", opts.Args.Name)
	}
}
