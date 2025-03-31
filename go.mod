module github.com/cirruslabs/cirrus-cli

go 1.23.0

toolchain go1.24.1

require (
	github.com/PaesslerAG/gval v1.2.2
	github.com/antihax/optional v1.0.0
	github.com/avast/retry-go/v4 v4.6.1
	github.com/breml/rootcerts v0.2.16
	github.com/cirruslabs/cirrus-ci-agent v1.134.1 // indirect
	github.com/cirruslabs/echelon v1.9.0
	github.com/cirruslabs/go-java-glob v0.1.0
	github.com/cirruslabs/podmanapi v0.3.1
	github.com/containers/image/v5 v5.34.2
	github.com/containers/storage v1.57.2 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6
	github.com/docker/cli v27.5.1+incompatible
	github.com/docker/docker v27.5.1+incompatible
	github.com/docker/go-units v0.5.0
	github.com/dustin/go-humanize v1.0.1
	github.com/getsentry/sentry-go v0.29.0
	github.com/go-git/go-billy/v5 v5.6.1
	github.com/go-git/go-git/v5 v5.13.1
	github.com/go-test/deep v1.1.0
	github.com/golang/protobuf v1.5.4
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/golang-lru v1.0.2
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c // indirect
	github.com/lestrrat-go/jsref v0.0.0-20211028120858-c0bcbb5abf20 // indirect
	github.com/lestrrat-go/jsschema v0.0.0-20181205002244-5c81c58ffcc3
	github.com/lestrrat-go/jsval v0.0.0-20181205002323-20277e9befc0 // indirect
	github.com/lestrrat-go/pdebug v0.0.0-20210111095411-35b07dbf089b // indirect
	github.com/lestrrat-go/structinfo v0.0.0-20210312050401-7f8bd69d6acb // indirect
	github.com/mitchellh/go-ps v1.0.0
	github.com/moby/buildkit v0.13.1
	github.com/moby/term v0.5.0 // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/otiai10/copy v1.14.0
	github.com/pkg/sftp v1.13.6
	github.com/qri-io/starlib v0.5.0
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/testify v1.10.0
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yudai/gojsondiff v1.0.0
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	go.starlark.net v0.0.0-20240314022150-ee8ed142361c
	golang.org/x/crypto v0.35.0
	golang.org/x/oauth2 v0.26.0 // indirect
	golang.org/x/sys v0.30.0
	golang.org/x/text v0.22.0
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v0.4.1
	github.com/IGLOU-EU/go-wildcard v1.0.3
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/aws/aws-sdk-go v1.55.5
	github.com/bmatcuk/doublestar v1.3.4
	github.com/cirruslabs/cirrus-ci-annotations v0.10.0
	github.com/cirruslabs/terminal v0.16.0
	github.com/docker/go-connections v0.5.0
	github.com/go-chi/render v1.0.3
	github.com/go-faster/errors v0.7.1
	github.com/go-faster/jx v1.1.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/go-github/v59 v59.0.0
	github.com/hashicorp/vault/api v1.12.2
	github.com/klauspost/pgzip v1.2.6
	github.com/ogen-go/ogen v1.4.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.15.1
	github.com/puzpuzpuz/xsync/v3 v3.1.0
	github.com/samber/lo v1.39.0
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/testcontainers/testcontainers-go v0.33.0
	github.com/tonistiigi/go-actions-cache v0.0.0-20240327122527-58651d5e11d6
	github.com/twitchtv/twirp v8.1.3+incompatible
	go.opentelemetry.io/contrib/bridges/otelzap v0.8.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.9.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0
	go.opentelemetry.io/otel/log v0.11.0
	go.opentelemetry.io/otel/metric v1.35.0
	go.opentelemetry.io/otel/sdk v1.35.0
	go.opentelemetry.io/otel/sdk/log v0.11.0
	go.opentelemetry.io/otel/sdk/metric v1.35.0
	go.opentelemetry.io/otel/trace v1.35.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67
	golang.org/x/sync v0.11.0
	golang.org/x/time v0.11.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.9 // indirect
	github.com/ProtonMail/go-crypto v1.1.3 // indirect
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/containerd/containerd v1.7.27 // indirect
	github.com/containerd/continuity v0.4.4 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/creack/pty v1.1.21 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go v1.5.1-1.0.20160303222718-d30aec9fd63c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fvbommel/sortorder v1.1.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/in-toto/in-toto-golang v0.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joshdk/go-junit v1.0.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20240513124658-fba389f38bae // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/capability v0.4.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.57.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.9.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/theupdateframework/notary v0.7.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/tonistiigi/fsutil v0.0.0-20240301111122-7525a1af2bb5 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
