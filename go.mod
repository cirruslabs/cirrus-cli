module github.com/cirruslabs/cirrus-cli

go 1.19

require (
	github.com/PaesslerAG/gval v1.2.2
	github.com/antihax/optional v1.0.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/certifi/gocertifi v0.0.0-20210507211836-431795d63e8d
	github.com/cirruslabs/cirrus-ci-agent v1.103.0
	github.com/cirruslabs/echelon v1.8.0
	github.com/cirruslabs/go-java-glob v0.1.0
	github.com/cirruslabs/podmanapi v0.3.0
	github.com/containers/image/v5 v5.24.2
	github.com/containers/storage v1.45.4 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/docker/cli v23.0.1+incompatible
	github.com/docker/docker v23.0.1+incompatible
	github.com/docker/go-units v0.5.0
	github.com/dustin/go-humanize v1.0.1
	github.com/go-git/go-billy/v5 v5.4.1
	github.com/go-git/go-git/v5 v5.6.0
	github.com/go-test/deep v1.0.1
	github.com/golang/protobuf v1.5.2
	github.com/google/go-github/v32 v32.1.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c // indirect
	github.com/lestrrat-go/jsref v0.0.0-20211028120858-c0bcbb5abf20 // indirect
	github.com/lestrrat-go/jsschema v0.0.0-20181205002244-5c81c58ffcc3
	github.com/lestrrat-go/jsval v0.0.0-20181205002323-20277e9befc0 // indirect
	github.com/lestrrat-go/pdebug v0.0.0-20210111095411-35b07dbf089b // indirect
	github.com/lestrrat-go/structinfo v0.0.0-20210312050401-7f8bd69d6acb // indirect
	github.com/mitchellh/go-ps v1.0.0
	github.com/moby/buildkit v0.11.4
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/otiai10/copy v1.9.0
	github.com/pkg/sftp v1.13.5
	github.com/qri-io/starlib v0.5.0
	github.com/sergi/go-diff v1.3.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/testify v1.8.2
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yudai/gojsondiff v1.0.0
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	go.starlark.net v0.0.0-20230302034142-4b1e35fe2254
	golang.org/x/crypto v0.6.0
	golang.org/x/oauth2 v0.5.0
	golang.org/x/sys v0.5.0
	golang.org/x/text v0.7.0
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/getsentry/sentry-go v0.18.0

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230217124315-7d5c6f04bbb8 // indirect
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.3.2 // indirect
	github.com/containerd/containerd v1.6.19 // indirect
	github.com/containerd/continuity v0.3.1-0.20230206214859-2a963a2f56e8 // indirect
	github.com/containerd/ttrpc v1.1.0 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go v1.5.1-1.0.20160303222718-d30aec9fd63c // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fvbommel/sortorder v1.0.2 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.16.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2 // indirect
	github.com/opencontainers/runc v1.1.5 // indirect
	github.com/opencontainers/runtime-spec v1.1.0-rc.1 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.41.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/skeema/knownhosts v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/theupdateframework/notary v0.7.0 // indirect
	github.com/tonistiigi/fsutil v0.0.0-20230214225802-a3696a2f1f27 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.40.0 // indirect
	go.opentelemetry.io/otel v1.14.0 // indirect
	go.opentelemetry.io/otel/metric v0.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.14.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20230301171018-9ab4bdc49ad5 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
