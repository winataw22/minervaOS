package gateway

import (
	"text/template"
)

var fwTmpl *template.Template

func init() {
	fwTmpl = template.Must(template.New("").Parse(_nft))
}

var _nft = `
flush ruleset

# until kernel 5.2, we use ip nat
table ip nat {
  chain prerouting {
    type nat hook prerouting priority -100; policy accept;
  }

  chain input {
    type nat hook input priority 100; policy accept;
  }

  chain output {
    type nat hook output priority -100; policy accept;
  }

  chain postrouting {
    type nat hook postrouting priority 100; policy accept;
    ip saddr 10.0.0.0/8 masquerade fully-random
  }
}
# future placeholder
table inet filter {
  chain input {
    type filter hook input priority 0; policy accept;
  }

  chain forward {
    type filter hook forward priority 0; policy accept;
  }

  chain output {
    type filter hook output priority 0; policy accept;
  }
}
`
