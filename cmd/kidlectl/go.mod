module github.com/kidle-dev/kidle/cmd/kidlectl

go 1.16

require (
	github.com/jessevdk/go-flags v1.5.0
	github.com/kidle-dev/kidle v0.0.0
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)

replace github.com/kidle-dev/kidle => ../../
