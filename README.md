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
```

## Install

```sh
export HOMEBREW_GITHUB_API_TOKEN=<token able to read MDS-BD repos>
brew tap MDS-BD/tools git@github.com:MDS-BD/homebrew-tools.git
brew install --cask aws-tunnel
```

Global flags (`--region`, `--profile`, `-t`, `-r`, `--local`) apply to every subcommand; run `aws-tunnel <command> --help` for the rest.

## Services

| Command | Lists | Auth |
| ------- | ----- | ---- |
| `opensearch` | `opensearch:ListDomainNames` + `DescribeDomains` | local proxy SigV4-signs every request |
| `rds` | `rds:DescribeDBClusters` (writer + reader endpoints) + `DescribeDBInstances` (standalone) | database password, or `--iam` token (`rds-db:connect`) |

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
| `HOMEBREW_TAP_GITHUB_TOKEN` | yes | Fine-grained token, *Contents: read & write* on `MDS-BD/homebrew-tools` |
| `MACOS_SIGN_P12` | no | base64 of the *Developer ID Application* certificate exported as `.p12` |
| `MACOS_SIGN_PASSWORD` | no | password of the `.p12` |
| `MACOS_NOTARY_KEY` | no | base64 of the App Store Connect API key (`.p8`) |
| `MACOS_NOTARY_KEY_ID` | no | ID of that key |
| `MACOS_NOTARY_ISSUER_ID` | no | issuer UUID of that key |

Without the `MACOS_*` secrets the darwin binaries ship unsigned; the cask strips the quarantine flag on install.

Local dry run: `goreleaser release --snapshot --clean --skip=publish`.
