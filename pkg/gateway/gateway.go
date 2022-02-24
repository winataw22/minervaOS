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
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

var (
	domainRe        = regexp.MustCompile("^Host(?:SNI)?\\(`([^`]+)`\\)$")
	traefikBinRegex = regexp.MustCompile("/var/cache/modules/flistd/mountpoint/([a-z0-9:]+)/traefik")
)

const (
	traefikService = "traefik"
	// letsencrypt email need to customizable by he farmer.
	letsencryptEmail = "letsencrypt@threefold.tech"
	// certResolver must match the one defined in static config
	httpCertResolver = "resolver"
	dnsCertResolver  = "dnsresolver"
	validationPeriod = 1 * time.Hour
)

var (
	ErrTwinIDMismatch       = fmt.Errorf("twin id mismatch")
	ErrContractNotReserved  = fmt.Errorf("a name contract with the given name must be reserved first")
	ErrInvalidContractState = fmt.Errorf("the name contract must be in Created state")
)

type gatewayModule struct {
	cl       zbus.Client
	resolver *net.Resolver
	sub      substrate.Manager
	// maps domain to workload id
	reservedDomains map[string]string
	domainLock      sync.RWMutex

	proxyConfigPath  string
	staticConfigPath string
	binPath          string
	certScriptPath   string
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
	CertResolver string   `yaml:"certResolver,omitempty"`
	Domains      []Domain `yaml:"domains,omitempty"`
	Passthrough  string   `yaml:"passthrough,omitempty"`
}

type Domain struct {
	Sans []string `yaml:"sans,omitempty"`
	Main string   `yaml:"main,omitempty"`
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

// domainFromConfig returns workloadID, domain, error
func domainFromConfig(path string) (string, string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to read file")
	}

	var c ProxyConfig
	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to unmarshal yaml file")
	}
	var routers map[string]Router
	if c.TCP != nil {
		routers = c.TCP.Routers
	} else if c.Http != nil {
		routers = c.Http.Routers
	} else {
		return "", "", fmt.Errorf("yaml file doesn't contain valid http or tcp config %s", path)
	}
	if len(routers) > 1 {
		return "", "", fmt.Errorf("only one router expected, found more: %s", path)
	}

	for _, router := range routers {
		domain, err := domainFromRule(router.Rule)
		return router.Service, domain, err
	}

	return "", "", fmt.Errorf("no routes defined in: %s", path)
}

func loadDomains(ctx context.Context, dir string) (map[string]string, error) {
	domains := make(map[string]string)
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read dir")
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			path := filepath.Join(dir, entry.Name())
			wlID, domain, err := domainFromConfig(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to load domain from config file")
				continue
			}
			domains[domain] = wlID
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

	dnsmasqCfgPath := filepath.Join(root, "dnsmasq.conf")
	err = dnsmasqConfig(dnsmasqCfgPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dnsmasq config")
	}
	certScriptPath := filepath.Join(root, "cert.sh")
	err = updateCertScript(certScriptPath, root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cert script")
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
	sub, err := environment.GetSubstrate()
	if err != nil {
		return nil, err
	}

	gw := &gatewayModule{
		cl:               cl,
		resolver:         r,
		sub:              sub,
		proxyConfigPath:  configPath,
		staticConfigPath: staticCfgPath,
		certScriptPath:   certScriptPath,
		binPath:          bin,
		reservedDomains:  domains,
		domainLock:       sync.RWMutex{},
	}

	// in case there are already active configurations we should always try to ensure running traefik
	if _, err := gw.ensureGateway(ctx, updated); err != nil {
		log.Error().Err(err).Msg("gateway is not supported")
		// this is not a failure because supporting of the gateway can happen
		// later if the farmer set the correct network configuration!
	}
	go gw.nameContractsValidator()
	return gw, nil
}

func (g *gatewayModule) getReservedDomain(domain string) (string, bool) {
	g.domainLock.RLock()
	defer g.domainLock.RUnlock()
	v, ok := g.reservedDomains[domain]
	return v, ok
}

func (g *gatewayModule) setReservedDomain(domain string, wlID string) {
	g.domainLock.Lock()
	defer g.domainLock.Unlock()
	log.Debug().
		Str("domain", domain).
		Str("wlID", wlID).
		Msg("setting domain")
	g.reservedDomains[domain] = wlID
}

func (g *gatewayModule) deleteReservedDomain(domain string) {
	g.domainLock.Lock()
	defer g.domainLock.Unlock()
	log.Debug().
		Str("domain", domain).
		Msg("deleting domain")
	delete(g.reservedDomains, domain)
}

func (g *gatewayModule) copyReservedDomain() map[string]string {
	g.domainLock.RLock()
	defer g.domainLock.RUnlock()
	res := make(map[string]string, len(g.reservedDomains))
	for k, v := range g.reservedDomains {
		res[k] = v
	}
	return res
}

func (g *gatewayModule) validateNameContracts() error {
	ctx, cancel := context.WithTimeout(context.Background(), validationPeriod/2)
	defer cancel()
	e := stubs.NewProvisionStub(g.cl)
	networker := stubs.NewNetworkerStub(g.cl)
	cfg, err := networker.GetPublicConfig(ctx)
	if err != nil {
		return nil
	}

	baseDomain := cfg.Domain
	if baseDomain == "" {
		// domain doesn't exist so no name workloads exist
		// or the domain was unset and name wokrloads will never be deleted
		// should iterate over workloads instead?
		return nil
	}
	reservedDomains := g.copyReservedDomain()
	for domain, id := range reservedDomains {
		wlID := gridtypes.WorkloadID(id)
		twinID, _, _, err := wlID.Parts()
		if err != nil {
			log.Error().
				Err(err).
				Msgf("failed to parse wlID %s parts", id)
			continue
		}
		if !strings.HasSuffix(domain, baseDomain) {
			// a fqdn workload, skip validating it
			continue
		}
		name := strings.TrimSuffix(domain, fmt.Sprintf(".%s", baseDomain))
		err = g.validateNameContract(name, twinID)
		if errors.Is(err, ErrContractNotReserved) || errors.Is(err, ErrInvalidContractState) || errors.Is(err, ErrTwinIDMismatch) {
			log.Debug().
				Str("reason", err.Error()).
				Str("wlID", id).
				Str("name", name).
				Msg("removing domain in name contract validation")
			if err := e.DecommissionCached(ctx, id, err.Error()); err != nil {
				log.Error().
					Err(err).
					Msgf("failed to decommission invalid gateway name workload %s", id)
			}
		} else if err != nil {
			log.Error().
				Str("reason", err.Error()).
				Str("wlID", id).
				Str("name", name).
				Msg("validating name contract failed because of a non-user error")
		}
	}
	return nil
}

func (g *gatewayModule) nameContractsValidator() {
	// no context?
	ticker := time.NewTicker(validationPeriod)
	defer ticker.Stop()
	for range ticker.C {
		if err := g.validateNameContracts(); err != nil {
			log.Error().Err(err).Msg("a round of failed name contract validation")
		}
	}
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

func (g *gatewayModule) traefikBinary(ctx context.Context, z *zinit.Client) (string, string, error) { // path, name, err
	info, err := z.Get(traefikService)
	if err != nil {
		return "", "", err
	}
	matches := traefikBinRegex.FindAllStringSubmatch(info.Exec, -1)
	if len(matches) != 1 {
		return "", "", errors.Wrapf(err, "find %d matches in %s", len(matches), info.Exec)
	}

	return matches[0][0], matches[0][1], nil
}

// ensureGateway makes sure that gateway infrastructure is in place and
// that it is supported.
func (g *gatewayModule) ensureGateway(ctx context.Context, forceResstart bool) (pkg.PublicConfig, error) {
	var (
		networker = stubs.NewNetworkerStub(g.cl)
		flistd    = stubs.NewFlisterStub(g.cl)
	)
	cfg, err := networker.GetPublicConfig(ctx)
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "gateway is not supported on this node")
	}

	z := zinit.Default()
	running, err := g.isTraefikStarted(z)
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "failed to check traefik status")
	}
	exists, err := z.Exists(traefikService)
	if err != nil {
		return pkg.PublicConfig{}, errors.Wrap(err, "couldn't get traefik service status")
	}
	if exists {
		path, name, err := g.traefikBinary(ctx, z)
		if err != nil {
			return pkg.PublicConfig{}, errors.Wrap(err, "failed to get old traefik binary path")
		}
		if path != g.binPath {
			if err := z.StopWait(10*time.Second, traefikService); err != nil {
				return pkg.PublicConfig{}, errors.Wrap(err, "failed to stop old traefik")
			}
			running = false
			if err := flistd.Unmount(ctx, name); err != nil {
				log.Error().Err(err).Msg("failed to unmount old traefik")
			}
		}

		if !running {
			if err := z.Forget(traefikService); err != nil {
				return pkg.PublicConfig{}, errors.Wrap(err, "failed to forget old traefik")
			}
		}
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
		Env: map[string]string{
			"EXEC_PATH": g.certScriptPath,
		},
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

func (g *gatewayModule) validateNameContract(name string, twinID uint32) error {
	sub, err := g.sub.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()
	contractID, err := sub.GetContractIDByNameRegistration(string(name))
	if errors.Is(err, substrate.ErrNotFound) {
		return ErrContractNotReserved
	}
	if err != nil {
		return err
	}
	contract, err := sub.GetContract(contractID)
	if errors.Is(err, substrate.ErrNotFound) {
		return fmt.Errorf("contract by name returned %d, but retreiving it results in 'not found' error", contractID)
	}
	if err != nil {
		return err
	}
	if !contract.State.IsCreated {
		return ErrInvalidContractState
	}
	if uint32(contract.TwinID) != twinID {
		return ErrTwinIDMismatch
	}
	return nil
}

func (g *gatewayModule) SetNamedProxy(wlID string, prefix string, backends []string, TLSPassthrough bool, twinID uint32) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cfg, err := g.ensureGateway(ctx, false)
	if err != nil {
		return "", err
	}
	if cfg.Domain == "" {
		return "", errors.New("node doesn't support name proxy (doesn't have a domain)")
	}
	if err := g.validateNameContract(prefix, twinID); err != nil {
		return "", errors.Wrap(err, "failed to verify name contract")
	}
	fqdn := fmt.Sprintf("%s.%s", prefix, cfg.Domain)

	gatewayTLSConfig := TlsConfig{
		CertResolver: dnsCertResolver,
		Domains: []Domain{
			{
				Sans: []string{fmt.Sprintf("*.%s", cfg.Domain)},
			},
		},
	}
	if err := g.setupRouting(wlID, fqdn, backends, gatewayTLSConfig, TLSPassthrough); err != nil {
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
	gatewayTLSConfig := TlsConfig{
		CertResolver: httpCertResolver,
		Domains: []Domain{
			{
				Main: fqdn,
			},
		},
	}
	return g.setupRouting(wlID, fqdn, backends, gatewayTLSConfig, TLSPassthrough)
}
func (g *gatewayModule) setupRouting(wlID string, fqdn string, backends []string, tlsConfig TlsConfig, TLSPassthrough bool) error {
	if _, ok := g.getReservedDomain(fqdn); ok {
		return errors.New("domain already registered")
	}
	var rule string
	if TLSPassthrough {
		rule = fmt.Sprintf("HostSNI(`%s`)", fqdn)
		tlsConfig = TlsConfig{
			Passthrough: "true",
		}
	} else {
		rule = fmt.Sprintf("Host(`%s`)", fqdn)
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
				Tls:     &tlsConfig,
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
	g.setReservedDomain(fqdn, wlID)
	return nil
}

func (g *gatewayModule) DeleteNamedProxy(wlID string) error {
	path := g.configPath(wlID)
	_, domain, err := domainFromConfig(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to load domain from config file")
	}
	if err := os.Remove(path); err != nil {
		return errors.Wrap(err, "couldn't remove config file")
	}
	if domain != "" {
		g.deleteReservedDomain(domain)
	}
	return nil
}
