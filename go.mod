module github.com/cirruslabs/cirrus-cli

go 1.14

// https://github.com/go-yaml/yaml/pull/364
replace gopkg.in/yaml.v2 => github.com/cirruslabs/yaml v0.0.0-20201223192638-1984d0f98f29

require (
	github.com/Microsoft/hcsshim v0.8.10 // indirect
	github.com/PaesslerAG/gval v1.1.0
	github.com/antihax/optional v1.0.0
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/bmatcuk/doublestar v1.3.2
	github.com/certifi/gocertifi v0.0.0-20200922220541-2c3bb06c6054
	github.com/cirruslabs/cirrus-ci-agent v1.25.1
	github.com/cirruslabs/echelon v1.4.0
	github.com/cirruslabs/podmanapi v0.1.0
	github.com/containerd/containerd v1.4.1 // indirect
	github.com/containerd/continuity v0.0.0-20201119173150-04c754faca46 // indirect
	github.com/containers/image/v5 v5.8.1
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/docker/cli v20.10.0-rc1+incompatible
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/go-units v0.4.0
	github.com/go-git/go-billy/v5 v5.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-test/deep v1.0.7
	github.com/golang/protobuf v1.4.3
	github.com/google/go-github/v32 v32.1.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hashicorp/go-version v1.2.1
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c // indirect
	github.com/lestrrat-go/jsref v0.0.0-20181205001954-1b590508f37d // indirect
	github.com/lestrrat-go/jsschema v0.0.0-20181205002244-5c81c58ffcc3
	github.com/lestrrat-go/jsval v0.0.0-20181205002323-20277e9befc0 // indirect
	github.com/lestrrat-go/pdebug v0.0.0-20200204225717-4d6bd78da58d // indirect
	github.com/lestrrat-go/structinfo v0.0.0-20190212233437-acd51874663b // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pkg/sftp v1.12.0
	github.com/qri-io/starlib v0.4.2
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yudai/gojsondiff v1.0.0
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	go.starlark.net v0.0.0-20201118183435-e55f603d8c79
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201022231255-08b38378de70 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sys v0.0.0-20201022201747-fb209a7c41cd
	golang.org/x/text v0.3.3
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20201022181438-0ff5f38871d5 // indirect
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)
