# dynamic-dns-cloudflare

dynamic-dns-cloudflare is a small application that will update DNS records for
machines with dynamic IP addresses. Written to allow remote access to home test
machines.

It requires a record for `-domain` already exists in `-zone`.

Thanks to [tgross/dynamic-dns-route53][tgross-route53] for the idea to use DNS
to resolve the IP.

## Usage

### Environment Variables

| Name | Description |
| ---- | ----------- |
| `CF_API_KEY` | The cloudflare api token to use, it can be found [here][cf-account-page] |
| `CF_API_EMAIL` | The cloudflare login email |

### CLI Arguments

| Name | Description |
| ---- | ----------- |
| `-zone` | The zone the domain is registered against |
| `-domain` | The domain name to update records for |
| `-interval` | The interval to update records at, default: run once |

[cf-account-page]: https://dash.cloudflare.com/profile
[tgross-route53]: https://github.com/tgross/dynamic-dns-route53
