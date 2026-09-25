# es-proxy

SSM-tunnelled, SigV4-signing proxy to a private OpenSearch domain. Pure Go: no `aws` CLI,
no `session-manager-plugin`, no Docker, no `fzf`.

```sh
go build -o bin/es-proxy ./cmd
es-proxy                      # pick domain, pick SSM instance, proxy on http://localhost:8080
es-proxy -r                   # ignore cache, reload domains and instances from AWS
es-proxy --host vpc-x.eu-west-1.es.amazonaws.com -t i-0123456789abcdef0
```

## How it works

1. Credentials come from the standard AWS chain (env, `--profile`, SSO...). `sts:GetCallerIdentity` gives the account ID.
2. Domains (`opensearch:ListDomainNames` + `DescribeDomains`) and SSM-online instances (`ssm:DescribeInstanceInformation`,
   names from `ec2:DescribeTags`) are cached
   per account + region in `$(os.UserCacheDir)/es-proxy/<account>-<region>.json`, refreshed after `--cache-ttl` (24h) or with `-r`.
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
