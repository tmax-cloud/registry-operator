package scan

import (
	"github.com/gorilla/mux"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

func TestGetScanResult(t *testing.T) {
	imgUrl := "127.0.0.1:32222/test"
	imgTag := "test"
	regUrl := "http://127.0.0.1:32222"
	clairUrl := regUrl

	launchTestServer(t)

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if err := os.Setenv("CLAIR_URL", clairUrl); err != nil {
		t.Fatal(err)
	}

	img, err := trust.NewImage(imgUrl, regUrl, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	img.Tag = imgTag

	res, err := GetScanResult(img)
	if err != nil {
		t.Fatal(err)
	}

	for s, vl := range res {
		log.Info("======")
		log.Info(s)
		for _, v := range vl {
			log.Info(v.Name)
		}
		log.Info("======")
	}
}

func launchTestServer(t *testing.T) {
	router := mux.NewRouter()
	// Registry
	router.HandleFunc("/v2", func(w http.ResponseWriter, req *http.Request) {})
	router.HandleFunc("/v2/test/manifests/test", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Header().Add("Docker-Content-Digest", "sha256:e8797cf5f16003f70e808c6e0b571cb4eb6393ef58de6f2c88a028bf6d925bff")
		w.Header().Add("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Add("Etag", "\"sha256:e8797cf5f16003f70e808c6e0b571cb4eb6393ef58de6f2c88a028bf6d925bff\"")
		w.Header().Add("X-Content-Type-Options", "nosniff")

		body := "{\"schemaVersion\":2,\"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\",\"config\":{\"mediaType\":\"application/vnd.docker.container.image.v1+json\",\"size\":12310,\"digest\":\"sha256:0019173091d74047c924799d4469f738458c8df3d4dde5e74567c40d5b13a0f1\"},\"layers\":[{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":75515164,\"digest\":\"sha256:71d1b80d640e2d963088bf3a6346137a8ec65b961be299feda2b632407ee574b\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":264,\"digest\":\"sha256:4805af504e5875409ab56b1da4c22a42302f920a6e57bba8902b1fc15c6a06b5\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":73091545,\"digest\":\"sha256:2cab6e57631cb7d41d6e1cfffee81d82b79a177a7da43ec7e21dd013ec120cc2\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":2007,\"digest\":\"sha256:1642919186bf4e183888eb98a54083f5b30830d8caea8b589458f807ad4d4ea4\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":122367327,\"digest\":\"sha256:f17bfe00d7e7a34e9bf512eecb2a9f379d4b208b01a0f9fb3d795e35c8484694\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":1033,\"digest\":\"sha256:e6881ed20c3e53bc09739b563ff52879f79f2d99fa8706a675b465750c9e8484\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":2585,\"digest\":\"sha256:f9d2c8b4aa6159c1c6ca79361ec3a6796efc9bb4387ba769b41ebfc55a1c0cea\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":10618260,\"digest\":\"sha256:dcb63e6d969d5d58b6106381fd573af66524d02ca43cb651c2ab8be0beb04dbf\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":272230,\"digest\":\"sha256:c4f79e3b5317c99f7db8c4e85ad3c94f3b049f912379270041203ec2a92db7a1\"}]}"

		_, err := w.Write([]byte(body))
		if err != nil {
			t.Fatal(err)
		}
	})

	// Clair
	router.HandleFunc("/v1/layers/{layerId}", func(w http.ResponseWriter, req *http.Request) {
		body := "{\"Layer\":{\"Name\":\"sha256:4805af504e5875409ab56b1da4c22a42302f920a6e57bba8902b1fc15c6a06b5\",\"NamespaceName\":\"centos:7\",\"ParentName\":\"sha256:71d1b80d640e2d963088bf3a6346137a8ec65b961be299feda2b632407ee574b\",\"IndexedByVersion\":3,\"Features\":[{\"Name\":\"openssl-libs\",\"NamespaceName\":\"centos:7\",\"VersionFormat\":\"rpm\",\"Version\":\"1:1.0.1e-60.el7\",\"Vulnerabilities\":[{\"Name\":\"RHSA-2019:2304\",\"NamespaceName\":\"centos:7\",\"Description\":\"OpenSSL is a toolkit that implements the Secure Sockets Layer (SSL) and Transport Layer Security (TLS) protocols, as well as a full-strength general-purpose cryptography library. Security Fix(es): * openssl: 0-byte record padding oracle (CVE-2019-1559) * openssl: timing side channel attack in the DSA signature algorithm (CVE-2018-0734) For more details about the security issue(s), including the impact, a CVSS score, acknowledgments, and other related information, refer to the CVE page(s) listed in the References section. Additional Changes: For detailed information on changes in this release, see the Red Hat Enterprise Linux 7.7 Release Notes linked from the References section.\",\"Link\":\"https:\\/\\/access.redhat.com\\/errata\\/RHSA-2019:2304\",\"Severity\":\"Medium\",\"FixedBy\":\"1:1.0.2k-19.el7\"},{\"Name\":\"RHBA-2017:1929\",\"NamespaceName\":\"centos:7\",\"Description\":\"OpenSSL is a toolkit that implements the Secure Sockets Layer (SSL) and Transport Layer Security (TLS) protocols, as well as a full-strength general-purpose cryptography library. For detailed information on changes in this release, see the Red Hat Enterprise Linux 7.4 Release Notes linked from the References section. Users of openssl are advised to upgrade to these updated packages.\",\"Link\":\"https:\\/\\/access.redhat.com\\/errata\\/RHBA-2017:1929\",\"Severity\":\"Medium\",\"FixedBy\":\"1:1.0.2k-8.el7\"}],\"AddedBy\":\"sha256:71d1b80d640e2d963088bf3a6346137a8ec65b961be299feda2b632407ee574b\"}]}}"

		_, err := w.Write([]byte(body))
		if err != nil {
			t.Fatal(err)
		}
	})

	srv := &http.Server{Addr: "0.0.0.0:32222", Handler: router}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	}()
}
