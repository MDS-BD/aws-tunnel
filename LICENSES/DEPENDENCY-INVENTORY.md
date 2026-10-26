# Audited release dependency inventory

Exact dependency source licenses are archived under `LICENSES/_third-party/` at each module's exact selected version. This table records the distinct module/version union used to drive that package; a module may carry additional nested license files. Each row preserves the audit's source-license attribution and upstream retrieval address for provenance. Exact module versions are also pinned in `go.mod`/`go.sum`.

| Module and version | Identified license(s) | Source reference(s) |
|---|---|---|
| `github.com/alecthomas/kong@v1.16.1` | MIT | [MIT](https://github.com/alecthomas/kong/blob/v1.16.1/COPYING) |
| `github.com/alexbacchin/ssm-session-client@v1.3.0` | MIT | [MIT](https://github.com/alexbacchin/ssm-session-client/blob/v1.3.0/LICENSE) |
| `github.com/aws/aws-sdk-go-v2@v1.47.1` | Apache-2.0, BSD-3-Clause | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/v1.47.1/LICENSE.txt); [BSD-3-Clause](https://github.com/aws/aws-sdk-go-v2/blob/v1.47.1/internal/sync/singleflight/LICENSE) |
| `github.com/aws/aws-sdk-go-v2/config@v1.33.6` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/config/v1.33.6/config/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/credentials@v1.20.6` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/credentials/v1.20.6/credentials/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/feature/ec2/imds@v1.20.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/feature/ec2/imds/v1.20.1/feature/ec2/imds/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/feature/rds/auth@v1.7.4` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/feature/rds/auth/v1.7.4/feature/rds/auth/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/internal/configsources@v1.5.4` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/internal/configsources/v1.5.4/internal/configsources/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/internal/endpoints/v2@v2.8.4` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/internal/endpoints/v2.8.4/internal/endpoints/v2/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/internal/v4a@v1.5.4` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/internal/v4a/v1.5.4/internal/v4a/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/apigateway@v1.50.0` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/apigateway/v1.50.0/service/apigateway/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/ec2@v1.336.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/ec2/v1.336.1/service/ec2/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/eks@v1.101.0` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/eks/v1.101.0/service/eks/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/elasticache@v1.62.0` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/elasticache/v1.62.0/service/elasticache/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding@v1.13.19` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/internal/accept-encoding/v1.13.19/service/internal/accept-encoding/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/internal/presigned-url@v1.14.4` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/internal/presigned-url/v1.14.4/service/internal/presigned-url/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/kms@v1.61.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/kms/v1.61.1/service/kms/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/opensearch@v1.82.0` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/opensearch/v1.82.0/service/opensearch/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/rds@v1.129.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/rds/v1.129.1/service/rds/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/signin@v1.10.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/signin/v1.10.1/service/signin/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/ssm@v1.78.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/ssm/v1.78.1/service/ssm/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/sso@v1.38.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/sso/v1.38.1/service/sso/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/ssooidc@v1.43.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/ssooidc/v1.43.1/service/ssooidc/LICENSE.txt) |
| `github.com/aws/aws-sdk-go-v2/service/sts@v1.51.1` | Apache-2.0 | [Apache-2.0](https://github.com/aws/aws-sdk-go-v2/blob/service/sts/v1.51.1/service/sts/LICENSE.txt) |
| `github.com/aws/session-manager-plugin@v0.0.0-20260615221425-930a08e65d3a` | Apache-2.0 | [Apache-2.0](https://github.com/aws/session-manager-plugin/blob/930a08e65d3a/LICENSE) |
| `github.com/aws/smithy-go@v1.28.2` | Apache-2.0, BSD-3-Clause | [Apache-2.0](https://github.com/aws/smithy-go/blob/v1.28.2/LICENSE); [BSD-3-Clause](https://github.com/aws/smithy-go/blob/v1.28.2/internal/sync/singleflight/LICENSE); [BSD-3-Clause](https://github.com/aws/smithy-go/blob/v1.28.2/transport/http/protocol/internal/json/internal/stdlib/LICENSE) |
| `github.com/beorn7/perks@v1.0.1` | MIT | [MIT](https://github.com/beorn7/perks/blob/v1.0.1/LICENSE) |
| `github.com/cespare/xxhash/v2@v2.3.0` | MIT | [MIT](https://github.com/cespare/xxhash/blob/v2.3.0/LICENSE.txt) |
| `github.com/cihub/seelog@v0.0.0-20170130134532-f561c5e57575` | BSD-3-Clause | [BSD-3-Clause](https://github.com/cihub/seelog/blob/f561c5e57575/LICENSE.txt) |
| `github.com/clipperhouse/uax29/v2@v2.7.0` | MIT | [MIT](https://github.com/clipperhouse/uax29/blob/v2.7.0/LICENSE) |
| `github.com/davecgh/go-spew@v1.1.2-0.20180830191138-d8f796af33cc` | ISC | [ISC](https://github.com/davecgh/go-spew/blob/d8f796af33cc/LICENSE) |
| `github.com/fsnotify/fsnotify@v1.10.1` | BSD-3-Clause | [BSD-3-Clause](https://github.com/fsnotify/fsnotify/blob/v1.10.1/LICENSE) |
| `github.com/fxamacker/cbor/v2@v2.9.4` | MIT | [MIT](https://github.com/fxamacker/cbor/blob/v2.9.4/LICENSE) |
| `github.com/gdamore/encoding@v1.0.1` | Apache-2.0 | [Apache-2.0](https://github.com/gdamore/encoding/blob/v1.0.1/LICENSE) |
| `github.com/gdamore/tcell/v2@v2.13.10` | Apache-2.0 | [Apache-2.0](https://github.com/gdamore/tcell/blob/v2.13.10/LICENSE) |
| `github.com/go-logr/logr@v1.4.4` | Apache-2.0 | [Apache-2.0](https://github.com/go-logr/logr/blob/v1.4.4/LICENSE) |
| `github.com/gofrs/flock@v0.13.1` | BSD-3-Clause | [BSD-3-Clause](https://github.com/gofrs/flock/blob/v0.13.1/LICENSE) |
| `github.com/google/renameio/v2@v2.0.2` | Apache-2.0 | [Apache-2.0](https://github.com/google/renameio/blob/v2.0.2/LICENSE) |
| `github.com/google/uuid@v1.6.0` | BSD-3-Clause | [BSD-3-Clause](https://github.com/google/uuid/blob/v1.6.0/LICENSE) |
| `github.com/gorilla/websocket@v1.5.4-0.20250319132907-e064f32e3674` | BSD-2-Clause | [BSD-2-Clause](https://github.com/gorilla/websocket/blob/e064f32e3674/LICENSE) |
| `github.com/hashicorp/go-version@v1.9.0` | MPL-2.0 | [MPL-2.0](https://github.com/hashicorp/go-version/blob/v1.9.0/LICENSE) |
| `github.com/json-iterator/go@v1.1.12` | MIT | [MIT](https://github.com/json-iterator/go/blob/v1.1.12/LICENSE) |
| `github.com/ktr0731/go-ansisgr@v0.1.0` | MIT | [MIT](https://github.com/ktr0731/go-ansisgr/blob/v0.1.0/LICENSE) |
| `github.com/ktr0731/go-fuzzyfinder@v0.9.0` | MIT | [MIT](https://github.com/ktr0731/go-fuzzyfinder/blob/v0.9.0/LICENSE) |
| `github.com/lucasb-eyer/go-colorful@v1.4.1` | MIT | [MIT](https://github.com/lucasb-eyer/go-colorful/blob/v1.4.1/LICENSE) |
| `github.com/mattn/go-colorable@v0.1.15` | MIT | [MIT](https://github.com/mattn/go-colorable/blob/v0.1.15/LICENSE) |
| `github.com/mattn/go-isatty@v0.0.24` | MIT | [MIT](https://github.com/mattn/go-isatty/blob/v0.0.24/LICENSE) |
| `github.com/mattn/go-runewidth@v0.0.30` | MIT | [MIT](https://github.com/mattn/go-runewidth/blob/v0.0.30/LICENSE) |
| `github.com/modern-go/concurrent@v0.0.0-20180306012644-bacd9c7ef1dd` | Apache-2.0 | [Apache-2.0](https://github.com/modern-go/concurrent/blob/bacd9c7ef1dd/LICENSE) |
| `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` | Apache-2.0 | [Apache-2.0](https://github.com/modern-go/reflect2/blob/35a7c28c31ee/LICENSE) |
| `github.com/munnerz/goautoneg@v0.0.0-20191010083416-a7dc8b61c822` | BSD-3-Clause | [BSD-3-Clause](https://github.com/munnerz/goautoneg/blob/a7dc8b61c822/LICENSE) |
| `github.com/nsf/termbox-go@v1.1.2` | MIT | [MIT](https://github.com/nsf/termbox-go/blob/v1.1.2/LICENSE) |
| `github.com/opensearch-project/opensearch-go/v4@v4.8.0` | Apache-2.0 | [Apache-2.0](https://github.com/opensearch-project/opensearch-go/blob/v4.8.0/LICENSE.txt) |
| `github.com/pkg/errors@v0.9.1` | BSD-2-Clause | [BSD-2-Clause](https://github.com/pkg/errors/blob/v0.9.1/LICENSE) |
| `github.com/prometheus/client_golang@v1.24.1` | Apache-2.0 | [Apache-2.0](https://github.com/prometheus/client_golang/blob/v1.24.1/LICENSE) |
| `github.com/prometheus/client_model@v0.6.3` | Apache-2.0 | [Apache-2.0](https://github.com/prometheus/client_model/blob/v0.6.3/LICENSE) |
| `github.com/prometheus/common@v0.71.0` | Apache-2.0 | [Apache-2.0](https://github.com/prometheus/common/blob/v0.71.0/LICENSE) |
| `github.com/prometheus/procfs@v0.22.0` | Apache-2.0 | [Apache-2.0](https://github.com/prometheus/procfs/blob/v0.22.0/LICENSE) (exact tagged file inspected) |
| `github.com/rivo/uniseg@v0.4.7` | MIT | [MIT](https://github.com/rivo/uniseg/blob/v0.4.7/LICENSE.txt) |
| `github.com/rs/zerolog@v1.35.1` | MIT | [MIT](https://github.com/rs/zerolog/blob/v1.35.1/LICENSE) |
| `github.com/sirupsen/logrus@v1.10.2` | MIT | [MIT](https://github.com/sirupsen/logrus/blob/v1.10.2/LICENSE) |
| `github.com/spf13/afero@v1.15.0` | Apache-2.0 | [Apache-2.0](https://github.com/spf13/afero/blob/v1.15.0/LICENSE.txt) |
| `github.com/spf13/pflag@v1.0.10` | BSD-3-Clause | [BSD-3-Clause](https://github.com/spf13/pflag/blob/v1.0.10/LICENSE) |
| `github.com/stretchr/objx@v0.5.3` | MIT | [MIT](https://github.com/stretchr/objx/blob/v0.5.3/LICENSE) |
| `github.com/stretchr/testify@v1.12.1` | BSD-3-Clause, ISC, MIT | [BSD-3-Clause](https://github.com/stretchr/testify/blob/v1.12.1/internal/difflib/LICENSE); [ISC](https://github.com/stretchr/testify/blob/v1.12.1/internal/spew/LICENSE); [MIT](https://github.com/stretchr/testify/blob/v1.12.1/LICENSE) |
| `github.com/twinj/uuid@v1.0.0` | MIT | [MIT](https://github.com/twinj/uuid/blob/89173bcdda19/LICENSE) |
| `github.com/x448/float16@v0.8.4` | MIT | [MIT](https://github.com/x448/float16/blob/v0.8.4/LICENSE) |
| `github.com/xtaci/smux@v1.5.57` | MIT | [MIT](https://github.com/xtaci/smux/blob/v1.5.57/LICENSE) |
| `go.uber.org/multierr@v1.11.0` | MIT | [MIT](https://github.com/uber-go/multierr/blob/v1.11.0/LICENSE.txt) |
| `go.uber.org/zap@v1.28.0` | MIT | [MIT](https://github.com/uber-go/zap/blob/v1.28.0/LICENSE) |
| `go.yaml.in/yaml/v2@v2.4.4` | Apache-2.0 | [Apache-2.0](https://github.com/yaml/go-yaml/blob/v2.4.4/LICENSE) |
| `go.yaml.in/yaml/v3@v3.0.5` | MIT | [MIT](https://github.com/yaml/go-yaml/blob/v3.0.5/LICENSE) |
| `golang.org/x/crypto@v0.57.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/crypto/+/v0.57.0:LICENSE) |
| `golang.org/x/net@v0.59.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/net/+/v0.59.0:LICENSE) |
| `golang.org/x/oauth2@v0.37.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/oauth2/+/v0.37.0:LICENSE) |
| `golang.org/x/sync@v0.23.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/sync/+/v0.23.0:LICENSE) |
| `golang.org/x/sys@v0.48.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/sys/+/v0.48.0:LICENSE) |
| `golang.org/x/term@v0.46.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/term/+/v0.46.0:LICENSE) |
| `golang.org/x/text@v0.42.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/text/+/v0.42.0:LICENSE) |
| `golang.org/x/time@v0.16.0` | BSD-3-Clause | [BSD-3-Clause](https://cs.opensource.google/go/x/time/+/v0.16.0:LICENSE) |
| `google.golang.org/protobuf@v1.36.12` | BSD-3-Clause | [BSD-3-Clause](https://github.com/protocolbuffers/protobuf-go/blob/v1.36.12/LICENSE) |
| `gopkg.in/inf.v0@v0.9.1` | BSD-3-Clause | [BSD-3-Clause](https://github.com/go-inf/inf/blob/v0.9.1/LICENSE) |
| `gopkg.in/natefinch/lumberjack.v2@v2.2.1` | MIT | [MIT](https://github.com/natefinch/lumberjack/blob/v2.2.1/LICENSE) |
| `gopkg.in/yaml.v2@v2.4.0` | Apache-2.0 | [Apache-2.0](https://github.com/go-yaml/yaml/blob/v2.4.0/LICENSE) |
| `k8s.io/apimachinery@v0.37.1` | Apache-2.0, BSD-3-Clause | [Apache-2.0](https://github.com/kubernetes/apimachinery/blob/v0.37.1/LICENSE); [BSD-3-Clause](https://github.com/kubernetes/apimachinery/blob/v0.37.1/third_party/forked/golang/LICENSE) |
| `k8s.io/client-go@v0.37.1` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes/client-go/blob/v0.37.1/LICENSE) |
| `k8s.io/klog/v2@v2.140.0` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes/klog/blob/v2.140.0/LICENSE) |
| `k8s.io/kube-openapi@v0.0.0-20260911184034-7970a1e230da` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes/kube-openapi/blob/7970a1e230da/LICENSE) |
| `k8s.io/utils@v0.0.0-20260707023825-cf1189d6abe3` | Apache-2.0, BSD-3-Clause | [Apache-2.0](https://github.com/kubernetes/utils/blob/cf1189d6abe3/LICENSE); [BSD-3-Clause](https://github.com/kubernetes/utils/blob/cf1189d6abe3/internal/third_party/forked/golang/LICENSE) |
| `sigs.k8s.io/aws-iam-authenticator@v0.7.20` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes-sigs/aws-iam-authenticator/blob/v0.7.20/LICENSE) |
| `sigs.k8s.io/json@v0.0.0-20260909141634-11ed52e25bc5` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes-sigs/json/blob/11ed52e25bc5/LICENSE) |
| `sigs.k8s.io/randfill@v1.0.0` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes-sigs/randfill/blob/v1.0.0/LICENSE) |
| `sigs.k8s.io/structured-merge-diff/v6@v6.4.2` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes-sigs/structured-merge-diff/blob/v6.4.2/LICENSE) |
| `sigs.k8s.io/yaml@v1.6.0` | Apache-2.0 | [Apache-2.0](https://github.com/kubernetes-sigs/yaml/blob/v1.6.0/LICENSE) |
