# Gateway Module

## ZBus

Gateway module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| gateway|[gateway](#interface)| 0.0.1|

## Home Directory

gateway keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/gateway`|

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
exec: gateway --broker unix://var/run/redis.sock --root /var/cache/modules/gateway
after:
  - boot
```

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
