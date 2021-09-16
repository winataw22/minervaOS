package gateway

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

var (
	domainRe = regexp.MustCompile("^Host(?:SNI)?\\(`([^`]+)`\\)$")
)

const (
	traefikService = "traefik"
	// letsencrypt email need to customizable by he farmer.
	letsencryptEmail = "letsencrypt@threefold.tech"
	// certResolver must match the one defined in static config
	certResolver = "resolver"
)

type gatewayModule struct {
	cl       zbus.Client
	resolver *net.Resolver

	reservedDomains  map[string]struct{}
	proxyConfigPath  string
	staticConfigPath string
	binPath          string
}

type ProxyConfig struct {
	Http *HTTPConfig `yaml:"http,omitempty"`
	TCP  *HTTPConfig `yaml:"tcp,omitempty"`
}

type HTTPConfig struct {
	Routers  map[string]Router
	Services map[string]Service
}

type Router struct {
	Rule    string
	Service string
	Tls     *TlsConfig `yaml:"tls,omitempty"`
}
type TlsConfig struct {
	CertResolver string `yaml:"certResolver,omitempty"`
	Passthrough  string `yaml:"passthrough,omitempty"`
}

type Service struct {
	LoadBalancer LoadBalancer
}

type LoadBalancer struct {
	Servers []Server
}

type Server struct {
	Url     string `yaml:"url,omitempty"`
	Address string `yaml:"address,omitempty"`
}

// domainFromRule gets domain from rules in the form Host(`domain`) or HostSNI(`domain`)
func domainFromRule(rule string) (string, error) {
	m := domainRe.FindStringSubmatch(rule)
	if len(m) == 2 {
		return m[1], nil
	}
	// no match
	return "", fmt.Errorf("failed to extract domain from routing rule '%s'", rule)
}

func domainFromConfig(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	var c ProxyConfig
	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal yaml file")
	}
	var routers map[string]Router
	if c.TCP != nil {
		routers = c.TCP.Routers
	} else if c.Http != nil {
		routers = c.Http.Routers
	} else {
		return "", fmt.Errorf("yaml file doesn't contain valid http or tcp config %s", path)
	}
	if len(routers) > 1 {
		return "", fmt.Errorf("only one router expected, found more: %s", path)
	}

	for _, router := range routers {
		return domainFromRule(router.Rule)
	}

	return "", fmt.Errorf("no routes defined in: %s", path)
}

func loadDomains(ctx context.Context, dir string) (map[string]struct{}, error) {
	domains := make(map[string]struct{})
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read dir")
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			path := filepath.Join(dir, entry.Name())
			domain, err := domainFromConfig(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to load domain from config file")
				continue
			}
			domains[domain] = struct{}{}
		}
	}
	return domains, nil
}
func New(ctx context.Context, cl zbus.Client, root string) (pkg.Gateway, error) {
	// where should service-restart/node-reboot recovery be handled?
	configPath := filepath.Join(root, "proxy")
	err := os.MkdirAll(configPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make gateway config dir")
	}

	traefikMetadata := filepath.Join(root, "traefik")
	err = os.MkdirAll(traefikMetadata, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make traefik metadata directory")
	}

	bin, err := ensureTraefikBin(ctx, cl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ensure traefik binary")
	}

	staticCfgPath := filepath.Join(root, "traefik.yaml")
	updated, err := staticConfig(staticCfgPath, root, letsencryptEmail)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create static config")
	}
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}
	domains, err := loadDomains(ctx, configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load old domains")
	}
	gw := &gatewayModule{
		cl:               cl,
		proxyConfigPath:  configPath,
		staticConfigPath: staticCfgPath,
		binPath:          bin,
		resolver:         r,
		reservedDomains:  domains,
	}

	// in case there are already active configurations we should always try to ensure running traefik
	if _, err := gw.ensureGateway(ctx, updated); err != nil {
		log.Error().Err(err).Msg("gateway is not supported")
		// this is not a failure because supporting of the gateway can happen
		// later if the farmer set the correct network configuration!
	}

	return gw, nil
}

func (g *gatewayModule) isTraefikStarted(z *zinit.Client) (bool, error) {
	traefikStatus, err := z.Status(traefikService)
	if errors.Is(err, zinit.ErrUnknownService) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check traefik status")
	}

	return traefikStatus.State.Is(zinit.ServiceStateRunning), nil
}

// ensureGateway makes sure that gateway infrastructure is in place and
// that it is supported.
func (g *gatewayModule) ensureGateway(ctx context.Context, forceResstart bool) (pkg.PublicConfig, error) {
	var (
		networker = stubs.NewNetworkerStub(g.cl)
	)
	cfg, err := networker.GetPublicConfig(ctx)
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "gateway is not supported on this node")
	}

	z, err := zinit.Default()
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "failed to connect to zinit")
	}
	defer z.Close()
	running, err := g.isTraefikStarted(z)
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "failed to check traefik status")
	}
	if running && forceResstart {
		// note: a kill is basically a singal to traefik process to
		// die. but zinit will restart it again anyway. so this is
		// enough to force restart it.
		if err := z.Kill(traefikService, zinit.SIGTERM); err != nil {
			return pkg.PublicConfig{}, errors.Wrap(err, "failed to restart traefik")
		}
	}

	if running {
		return cfg, nil
	}

	//other wise we start traefik
	return cfg, g.startTraefik(z)
}
func (g *gatewayModule) verifyDomainDestination(ctx context.Context, cfg pkg.PublicConfig, domain string) error {
	ips, err := g.resolver.LookupHost(ctx, domain)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if ip == cfg.IPv4.IP.String() || ip == cfg.IPv6.IP.String() {
			return nil
		}
	}
	return errors.New("host doesn't point to the gateway ip")
}

func (g *gatewayModule) startTraefik(z *zinit.Client) error {
	cmd := fmt.Sprintf(
		"ip netns exec public %s --configfile %s",
		g.binPath,
		g.staticConfigPath,
	)

	if err := zinit.AddService(traefikService, zinit.InitService{
		Exec: cmd,
	}); err != nil {
		return errors.Wrap(err, "failed to add traefik to zinit")
	}

	if err := z.Monitor(traefikService); err != nil {
		return errors.Wrap(err, "couldn't monitor traefik service")
	}

	if err := z.StartWait(time.Second*20, traefikService); err != nil {
		return errors.Wrap(err, "waiting for trafik start timed out")
	}

	return nil
}

func (g *gatewayModule) configPath(name string) string {
	return filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", name))
}

func (g *gatewayModule) SetNamedProxy(wlID string, prefix string, backends []string, TLSPassthrough bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cfg, err := g.ensureGateway(ctx, false)
	if err != nil {
		return "", err
	}
	if cfg.Domain == "" {
		return "", errors.New("node doesn't support name proxy (doesn't have a domain)")
	}
	fqdn := fmt.Sprintf("%s.%s", prefix, cfg.Domain)
	if err := g.setupRouting(wlID, fqdn, backends, TLSPassthrough); err != nil {
		return "", err
	} else {
		return fqdn, nil
	}
}

func (g *gatewayModule) SetFQDNProxy(wlID string, fqdn string, backends []string, TLSPassthrough bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cfg, err := g.ensureGateway(ctx, false)
	if err != nil {
		return err
	}

	if cfg.Domain != "" && strings.HasSuffix(fqdn, cfg.Domain) {
		return errors.New("can't create a fqdn workload with a subdomain of the gateway's managed domain")
	}
	if err := g.verifyDomainDestination(ctx, cfg, fqdn); err != nil {
		return errors.Wrap(err, "failed to verify domain dns record")
	}
	return g.setupRouting(wlID, fqdn, backends, TLSPassthrough)
}
func (g *gatewayModule) setupRouting(wlID string, fqdn string, backends []string, TLSPassthrough bool) error {
	if _, ok := g.reservedDomains[fqdn]; ok {
		return errors.New("domain already registered")
	}
	var rule string
	var tlsConfig *TlsConfig
	if TLSPassthrough {
		rule = fmt.Sprintf("HostSNI(`%s`)", fqdn)
		tlsConfig = &TlsConfig{
			Passthrough: "true",
		}
	} else {
		rule = fmt.Sprintf("Host(`%s`)", fqdn)
		tlsConfig = &TlsConfig{
			CertResolver: certResolver,
		}
	}
	servers := make([]Server, len(backends))
	for idx, backend := range backends {
		if TLSPassthrough {
			u, err := url.Parse(backend)
			log.Debug().Str("hostname", u.Host).Str("backend", backend).Msg("tls passthrough")
			if err != nil {
				return errors.Wrap(err, "couldn't parse backend host")
			}
			if u.Scheme != "https" {
				return errors.New("enabling tls passthrough requires backends to have https scheme")
			}
			servers[idx] = Server{
				Address: u.Host,
			}
		} else {
			servers[idx] = Server{
				Url: backend,
			}
		}
	}
	route := fmt.Sprintf("%s-route", wlID)
	proxyConfig := ProxyConfig{}

	routingconfig := &HTTPConfig{
		Routers: map[string]Router{
			route: {
				Rule:    rule,
				Service: wlID,
				Tls:     tlsConfig,
			},
		},
		Services: map[string]Service{
			wlID: {
				LoadBalancer: LoadBalancer{
					Servers: servers,
				},
			},
		},
	}
	if TLSPassthrough {
		proxyConfig.TCP = routingconfig
	} else {
		proxyConfig.Http = routingconfig
	}
	yamlString, err := yaml.Marshal(&proxyConfig)
	if err != nil {
		return errors.Wrap(err, "failed to convert config to yaml")
	}
	log.Debug().Str("yaml-config", string(yamlString)).Msg("configuration file")
	if err = os.WriteFile(g.configPath(wlID), yamlString, 0644); err != nil {
		return errors.Wrap(err, "couldn't open config file for writing")
	}
	g.reservedDomains[fqdn] = struct{}{}
	return nil
}

func (g *gatewayModule) DeleteNamedProxy(wlID string) error {
	path := g.configPath(wlID)
	domain, err := domainFromConfig(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to load domain from config file")
	}
	if err := os.Remove(path); err != nil {
		return errors.Wrap(err, "couldn't remove config file")
	}
	if domain != "" {
		delete(g.reservedDomains, domain)
	}
	return nil
}
