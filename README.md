# aws-tunnel

SSM tunnels to private AWS services, starting with a SigV4-signing proxy to OpenSearch. Pure Go: no `aws` CLI,
no `session-manager-plugin`, no Docker, no `fzf`.

```sh
go build -o bin/aws-tunnel ./cmd
aws-tunnel opensearch           # pick domain, pick SSM instance, proxy on http://localhost:8080
aws-tunnel -r opensearch        # ignore cache, reload domains and instances from AWS
aws-tunnel -t i-0123456789abcdef0 opensearch --host vpc-x.eu-west-1.es.amazonaws.com
aws-tunnel rds                  # pick database + instance, print a psql/mysql command
aws-tunnel rds --iam --user app # also print an IAM auth token, refreshed every 10 minutes
aws-tunnel apigateway --sign    # pick private API stage, proxy on http://localhost:8080, SigV4-signed
aws-tunnel elasticache --iam --user app  # pick cache, print redis-cli command and IAM token
aws-tunnel eks                  # pick private cluster, then: export KUBECONFIG=<printed path>; kubectl get pods
aws-tunnel shell                # pick instance, interactive shell (replaces ssmw start)
aws-tunnel tunnel --local 18085 --remote 8085 --host service.example.com
                                # localhost:18085 -> host:8085 through the instance (replaces gossm/ssmw fwdrem)
```

## Install

```sh
export HOMEBREW_GITHUB_API_TOKEN=$(gh auth token)   # any token able to read MDS-BD repos
brew tap MDS-BD/tools git@github.com:MDS-BD/homebrew-tools.git
brew install --cask mds-bd/tools/aws-tunnel
```

See [homebrew-tools](https://github.com/MDS-BD/homebrew-tools) for token options and the harmless ``postflight``
deprecation warning.

Global flags (`--region`, `--profile`, `-t`, `-r`, `--local`) apply to every subcommand; run `aws-tunnel <command> --help` for the rest.

## Services

| Command | Lists | Auth |
| ------- | ----- | ---- |
| `opensearch` | `opensearch:ListDomainNames` + `DescribeDomains` | local proxy SigV4-signs every request |
| `apigateway` | `apigateway:GET` on `/restapis` and their stages, private APIs only | none, or `--sign` for `AWS_IAM` methods (`execute-api:Invoke`) |
| `eks` | `eks:ListClusters` + `DescribeCluster`, private-endpoint clusters only | local proxy adds the IAM bearer token; your identity needs an EKS access entry (or `aws-auth` mapping) |
| `elasticache` | `elasticache:DescribeReplicationGroups` + `DescribeServerlessCaches` | AUTH token / RBAC password, or `--iam` token (`elasticache:Connect`) |
| `tunnel` | instance picker only; `--host` omitted reaches a port on the instance | none, plain TCP |
| `shell` | instance picker only | `ssm:StartSession` on the instance |
| `rds` | `rds:DescribeDBClusters` (writer + reader endpoints) + `DescribeDBInstances` (standalone) | database password, or `--iam` token (`rds-db:connect`) |

API Gateway: the proxy calls `https://<api-id>.execute-api.<region>.amazonaws.com/<stage>`, which resolves to the
`execute-api` VPC endpoint only when that endpoint has private DNS enabled. The API resource policy must also allow
the VPC endpoint.

EKS: the proxy verifies the API server against the cluster CA and injects the token, so the printed kubeconfig
points at `http://localhost:8443` with no credentials and no `aws eks get-token` exec plugin. `exec`, `attach`,
`port-forward`, `logs -f` and watches work through it. The kubeconfig is deleted on exit.

ElastiCache: cluster-mode-enabled groups answer `MOVED`/`ASK` with node addresses the single tunnel does not forward,
so only single-slot commands without cluster redirects work (plain `redis-cli`, no `-c`). Serverless caches and
cluster-mode-disabled groups work fully. TLS is kept but not host-verified (`--tls --insecure --sni <host>`).

RDS TLS: the client connects to `localhost`, so certificate host verification fails. Use `sslmode=require` /
`--ssl-mode=REQUIRED` (encrypted, not verified), or map the RDS host to `127.0.0.1` in `/etc/hosts` and verify
against the [RDS CA bundle](https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem) with `--local` set
to the database port.

## How it works

1. Credentials come from the standard AWS chain (env, `--profile`, SSO...). `sts:GetCallerIdentity` gives the account ID.
2. Domains (`opensearch:ListDomainNames` + `DescribeDomains`) and SSM-online instances (`ssm:DescribeInstanceInformation`,
   names from `ec2:DescribeTags`) are cached
   per account + region in `$(os.UserCacheDir)/aws-tunnel/<account>-<region>.json`, refreshed after `--cache-ttl` (24h) or with `-r`.
3. Pickers rank entries by frecency: every pick adds 1 to a score that halves every 14 days, so often *and* recently
   used entries come first.
4. An SSM `AWS-StartPortForwardingSessionToRemoteHost` session is opened in-process
   ([ssm-session-client](https://github.com/alexbacchin/ssm-session-client), multiplexed for agents >= 3.0.196.0).
   The picker shows each instance's SSM agent version and flags agents older than 3.0.196.0, which serve one
   connection at a time: update them with the `AWS-UpdateSSMAgent` Run Command document.
5. A local reverse proxy signs every request (opensearch-go `signer/awsv2`) and dials the tunnel, keeping the real
   domain name for Host/SNI: TLS is verified and no signing-host override is needed.

## Test

```sh
go test ./... -cover
```

## Release

Push a `v*` tag: the `release` workflow runs GoReleaser, which builds darwin/linux × amd64/arm64, publishes the GitHub
release and pushes the cask to [MDS-BD/homebrew-tools](https://github.com/MDS-BD/homebrew-tools).

```sh
git tag v0.1.0 && git push origin v0.1.0
```

Repository secrets:

| Secret | Required | Content |
| ------ | -------- | ------- |
| `HOMEBREW_RELEASER_APP_ID` | yes | App ID or Client ID of the releaser GitHub App, see below |
| `HOMEBREW_RELEASER_APP_PRIVATE_KEY` | yes | Private key (`.pem`) of that app |
| `MACOS_SIGN_P12` | no | base64 of the *Developer ID Application* certificate exported as `.p12` |
| `MACOS_SIGN_PASSWORD` | no | password of the `.p12`; omit it when the file has none |
| `MACOS_NOTARY_KEY` | no | base64 of the App Store Connect API key (`.p8`) |
| `MACOS_NOTARY_KEY_ID` | no | ID of that key |
| `MACOS_NOTARY_ISSUER_ID` | no | issuer UUID of that key |

The tap is pushed with a GitHub App instead of a personal token: tokens last one hour and can only write to
`MDS-BD/homebrew-tools`. One-time setup (org admin):

1. Organization settings → Developer settings → GitHub Apps → *New GitHub App*. Any name and homepage URL; untick
   *Webhook → Active*; *Repository permissions → Contents: Read and write*; *Only on this account*.
2. On the app page, copy the **App ID** (or the Client ID, both work), then *Generate a private key* (downloads a
   `.pem`).
3. *Install App* → MDS-BD → *Only select repositories* → `homebrew-tools`.
4. Store both in this repo:
   ```sh
   gh secret set HOMEBREW_RELEASER_APP_ID -R MDS-BD/aws-tunnel --body <app-id>
   gh secret set HOMEBREW_RELEASER_APP_PRIVATE_KEY -R MDS-BD/aws-tunnel < ./app-name.private-key.pem
   ```
   then delete the local `.pem`.

Without the `MACOS_*` secrets the darwin binaries ship unsigned; the cask strips the quarantine flag on install.

Local dry run: `goreleaser release --snapshot --clean --skip=publish`.
