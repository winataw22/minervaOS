package nr

import (
	"text/template"
)

var fwTmpl *template.Template

func init() {
	fwTmpl = template.Must(template.New("nrfw").Parse(_nft))
}

var _nft = `
flush ruleset
# The rules in the NR are there to make sure direct communication between
# the NRs behind exitpoints is impossible

table inet filter {
    chain base_checks {
        # allow established/related connections
        ct state {established, related} accept
        # early drop of invalid connections
        ct state invalid drop
    }
  chain input {
    type filter hook input priority 0; policy accept;
  }

  chain forward {
    type filter hook forward priority 0; policy accept;
        # is there already an existing stream? (outgoing)
        jump base_checks
        # if not, verify if it's new and coming in from the br4-gw network
        # if it is, drop it
        iifname "{{.Iifname}}" counter drop
  }

  chain output {
    type filter hook output priority 0; policy accept;
  }
}
`
