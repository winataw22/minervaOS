# Gateway Module

## ZBus

Gateway module is available on zbus over the following channel

| module  | object                | version |
| ------- | --------------------- | ------- |
| gateway | [gateway](#interface) | 0.0.1   |

## Home Directory

gateway keeps some data in the following locations
| directory | path                         |
| --------- | ---------------------------- |
| root      | `/var/cache/modules/gateway` |

The directory `/var/cache/modules/gateway/proxy` contains the route information used by traefik to forward traffic.
## Introduction

The gateway modules is used to register traefik routes and services to act as a reverse proxy. It's the backend supporting two kinds of workloads: `gateway-fqdn-proxy` and `gateway-name-proxy`.

For the FQDN type, it receives the domain and a list of backends in the form `http://ip:port` or `https://ip:port` and registers a route for this domain forwarding traffic to these backends. It's a requirement that the domain resolves to the gateway public ip. The `tls_passthrough` parameter determines whether the tls termination happens on the gateway or in the backends. When it's true, the backends must be in the form `https://ip:port`, and the backends must be https-enabled servers.

The name type is the same as the FQDN type except that the `name` parameter is added as a prefix to the gatweay domain to determine the fqdn. It's forbidden to use a FQDN type workload to reserve a domain managed by the gateway. 

The fqdn type is enabled only if there's a public config on the node. The name type works only if a domain exists in the public config. To make a full-fledged gateway node, these DNS records are required:
```
gatwaydomain.com                   A     ip.of.the.gateway
*.gatewaydomain.com                CNAME gatewaydomain.com
__acme-challenge.gatewaydomain.com NS    gatdwaydomain.com
```

### zinit unit

```yaml
exec: gateway --broker unix:///var/run/redis.sock --root /var/cache/modules/gateway
after:
  - boot
```
## Implementation details

Traefik is used as the reverse proxy forwarding traffic to upstream servers. All worklaods deployed on the node is associated with a domain that resolves to the node IP. In the name workload case, it's a subdomain of the gateway main domain. In the FQDN case, the user must create a DNS A record pointing it to the node IP. The node by default redirects all http traffic to https.

When an https request reaches the node, it looks at the domain and determines the correct service that should handle the request. The services defintions are in `/var/cache/modules/gateway/proxy/` and is hot-reloaded by traefik every time a service is added/removed to/from it. Zos currently supports enabling `tls_passthrough` in which case the https request is passed as is to the backend (at the TCP level). The default is `tls_passthrough` is false which means the node terminates the TLS traffic and then forwards the request as http to the backend. 
Example of a FQDN service definition with tls_passthrough enabled:
```yaml
tcp:
  routers:
    37-2039-testname-route:
      rule: HostSNI(`remote.omar.grid.tf`)
      service: 37-2039-testname
      tls:
        passthrough: "true"
  services:
    37-2039-testname:
      loadbalancer:
        servers:
        - address: 137.184.106.152:443
```
Example of a "name" service definition with tls_passthrough disabled:
```yaml
http:
  routers:
    37-1976-workloadname-route:
      rule: Host(`workloadname.gent01.dev.grid.tf`)
      service: 40-1976-workloadname
      tls:
        certResolver: dnsresolver
        domains:
        - sans:
          - '*.gent01.dev.grid.tf'
  services:
    40-1976-workloadname:
      loadbalancer:
        servers:
        - url: http://[backendip]:9000
```

The `certResolver` option has two valid values, `resolver` and `dnsresolver`. The `resolver` is an http resolver and is used in FQDN services with `tls_passthrough` disabled. It uses the http challenge to generate a single-domain certificate. The `dnsresolver` is used for name services with `tls_passthrough` disabled. The `dnsresolver` is responsible for generating a wildcard certificate to be used for all subdomains of the gateway domain. Its flow is described below.

The CNAME record is used to make all subdomains (reserved or not) resolve to the ip of the gateway. Generating a wildcard certificate requires adding a TXT record at `__acme-challenge.gatewaydomain.com`. The NS record is used to delegate this specific subdomain to the node. So if someone did `dig TXT __acme-challenge.gatewaydomain.com`, the query is served by the node, not the DNS provider used for the gateway domain.

Traefik has, as a config parameter, multiple dns [providers](https://doc.traefik.io/traefik/https/acme/#providers) to communicate with when it wants to add the required TXT record. For non-supported providers, a bash script can be provided to do the record generation and clean up (i.e. External program). The bash [script](https://github.com/threefoldtech/zos/blob/main/pkg/gateway/static/cert.sh) starts dnsmasq managing a dns zone for the `__acme-challenge` subdomain with the given TXT record. It then kills the dnsmasq process and removes the config file during cleanup.
## Interface

```go
type Backend string

// GatewayFQDNProxy definition. this will proxy name.<zos.domain> to backends
type GatewayFQDNProxy struct {
	// FQDN the fully qualified domain name to use (cannot be present with Name)
	FQDN string `json:"fqdn"`

	// Passthroug whether to pass tls traffic or not
	TLSPassthrough bool `json:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `json:"backends"`
}


// GatewayNameProxy definition. this will proxy name.<zos.domain> to backends
type GatewayNameProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string `json:"name"`

	// Passthroug whether to pass tls traffic or not
	TLSPassthrough bool `json:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `json:"backends"`
}

type Gateway interface {
	SetNamedProxy(wlID string, prefix string, backends []string, TLSPassthrough bool) (string, error)
	SetFQDNProxy(wlID string, fqdn string, backends []string, TLSPassthrough bool) error
	DeleteNamedProxy(wlID string) error
	Metrics() (GatewayMetrics, error)
}
```
