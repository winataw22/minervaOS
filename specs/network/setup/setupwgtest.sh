#!/usr/bin/bash

# setup 2 network namespaces, generate keys for 2 wg's

NUM=5
function genkeys() {
	for i in $(seq 1 $NUM); do
		wg genkey | tee wg${i}.priv | wg pubkey >wg${i}.pub
	done
}

function genconf() {
	for i in $(seq 1 $NUM); do
		PRIV=$(cat wg${i}.priv)
		cat <<EOF >wg${i}.conf
# WG${i}
[Interface]
ListenPort = 1234${i}
PrivateKey = $PRIV
EOF

		for wg in $(seq 1 $NUM); do
			if [ "$wg" -ne "$i" ]; then
				PUB=$(cat wg${wg}.pub)
				cat <<EEOF >>wg${i}.conf

# Config for --- WG${wg} ---
[Peer]
PublicKey = $PUB
Endpoint = 127.0.0.1:1234${wg}
AllowedIPs = fe80::${wg}/128, 192.168.255.${wg}, ::/0
PersistentKeepalive = 20
EEOF
			fi
		done
	done
}

function ns() {
	for i in $(seq 1 $NUM); do
		ip netns add wg${i}

		ip link add wg${i} type wireguard

		ip link set wg${i} netns wg${i}

		ip link add int${i} type dummy
		ip link set int${i} netns wg${i}

		ip -n wg${i} link set lo up
		ip -n wg${i} link set wg${i} up
		ip -n wg${i} link set int${i} up

		ip netns exec wg${i} wg setconf wg${i} wg${i}.conf

		ip -n wg${i} addr add fe80::${i}/64 dev wg${i}
		ip -n wg0 addr add 192.168.255.${i}/24 dev wg${i}

		ip -n wg0 addr add 2001:1:1:${i}::1/64 dev int${i}
	done
}

function main() {
	genkeys
	genconf
	ns
}
