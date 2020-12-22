module github.com/tmax-cloud/registry-operator

go 1.13

require (
	github.com/Nerzal/gocloak/v7 v7.8.0
	github.com/bugsnag/bugsnag-go v1.7.0 // indirect
	github.com/cloudflare/cfssl v1.5.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/go-logr/logr v0.2.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gorilla/mux v1.7.4
	github.com/jinzhu/gorm v1.9.16 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lib/pq v1.8.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.5 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/operator-framework/operator-lib v0.1.0
	github.com/theupdateframework/notary v0.6.2-0.20200804143915-84287fd8df4f
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kube-aggregator v0.19.4
	knative.dev/pkg v0.0.0-20201127013335-0d896b5c87b8
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.8
)
