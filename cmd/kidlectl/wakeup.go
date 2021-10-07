package main

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kidle-dev/kidle/cmd/kidlectl/pkg"
)

// cmdWakeup executes the kidlectl wakeup command with given args
func cmdWakeup(opts wakeupCommandOptions) {
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
