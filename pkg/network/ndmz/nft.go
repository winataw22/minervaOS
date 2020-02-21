package ndmz

import (
	"text/template"
)

var fwTmpl *template.Template

func init() {
	fwTmpl = template.Must(template.New("").Parse(_nft))
}

var _nft = `
flush ruleset

table inet nat {
  chain prerouting {
    type nat hook prerouting priority dstnat; policy accept;
  }

  chain input {
    type nat hook input priority 100; policy accept;
  }

  chain output {
    type nat hook output priority -100; policy accept;
  }

  chain postrouting {
    type nat hook postrouting priority srcnat; policy accept;
    oifname "npub4" masquerade fully-random;
    oifname "npub6" masquerade fully-random;
  }
}

table inet filter {

  chain base_checks {
    # allow established/related connections
    ct state {established, related} accept
    # early drop of invalid connections
    ct state invalid drop
  }
  
  chain input {
    type filter hook input priority 0; policy accept;
    jump base_checks
    ip6 nexthdr icmpv6 accept
    iifname "npub6" counter drop
    iifname "npub4" counter drop
  }

  chain forward {
    type filter hook forward priority 0; policy accept;
    # is there already an existing stream? (outgoing)
    jump base_checks
    # if not, verify if it's new and coming in from the br4-gw network
    # if it is, drop it
    iifname "npub6" counter drop
    iifname "npub4" counter drop
  }

  chain output {
    type filter hook output priority 0; policy accept;
  }
}
`
