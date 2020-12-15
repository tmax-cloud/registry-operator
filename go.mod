module github.com/tmax-cloud/registry-operator

go 1.13

require (
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bugsnag/bugsnag-go v1.7.0 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/cloudflare/cfssl v1.5.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/go-logr/logr v0.2.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/gorm v1.9.16 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lib/pq v1.8.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.5 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/operator-framework/operator-lib v0.1.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/theupdateframework/notary v0.6.2-0.20200804143915-84287fd8df4f
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/dancannon/gorethink.v3 v3.0.5 // indirect
	gopkg.in/fatih/pool.v2 v2.0.0 // indirect
	gopkg.in/gorethink/gorethink.v3 v3.0.5 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kube-aggregator v0.19.4
	knative.dev/pkg v0.0.0-20201127013335-0d896b5c87b8
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.8
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
)
