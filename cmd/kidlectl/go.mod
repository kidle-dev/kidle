module github.com/orphaner/kidle/cmd/kidlectl

go 1.16

require (
	github.com/jessevdk/go-flags v1.5.0
	github.com/orphaner/kidle v0.0.0
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime v0.8.3
)

replace github.com/orphaner/kidle => ../../
