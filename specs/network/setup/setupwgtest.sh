#!/usr/bin/bash

# setup NUM network namespaces that are connecte via a circular mesh:
# i.e. every namespace has an encrypted tunnel to the other, with the
# associated route

NUM=5
# setup 2 network namespaces, generate keys for 2 wg's

function genkeys() {
	for i in $(seq 1 $NUM); do
		wg genkey | tee wg${i}.priv | wg pubkey >wg${i}.pub
	done
}

function genconf() {
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		PRIV=$(cat wg${i}.priv)
		h=$(printf '%x' $i)
		port=$((16000 + $i))
		cat <<EOF >wg${i}.conf
# WG${i}
[Interface]
ListenPort = ${port}
PrivateKey = $PRIV
EOF
		for wg in $(seq 1 $NUM); do
			if [ "$wg" -ne "$i" ]; then
				port=$((16000 + $wg))
				PUB=$(cat wg${wg}.pub)
				h=$(printf '%x' $wg)
				cat <<EEOF >>wg${i}.conf

# Config for --- WG${wg} ---
[Peer]
PublicKey = $PUB
Endpoint = 127.0.0.1:${port}
AllowedIPs = fe80::${h},192.168.255.${wg},2001:1:1:${h}::/64
PersistentKeepalive = 20
EEOF
				if [ "$wg" -eq "1" ]; then
					cat <<EEEOF >>wg${i}.conf
AllowedIPs = fe80::${h},192.168.255.${wg},2001:1:1:${h}::/64,::/0,0.0.0.0/0
EEEOF
				else
					cat <<EEEOF >>wg${i}.conf
AllowedIPs = fe80::${h},192.168.255.${wg},2001:1:1:${h}::/64
EEEOF
				fi
				cat <<EEEOF >>wg${i}.conf
PersistentKeepalive = 20
EEEOF
			fi
		done
	done
	echo
}

function ns() {
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		h=$(printf '%x' $i)
		ip netns add wg${i}

		ip link add wg${i} type wireguard
		ip link set wg${i} netns wg${i}

		ip -n wg${i} link set lo up
		ip -n wg${i} link set wg${i} up

		ip netns exec wg${i} wg setconf wg${i} wg${i}.conf
		# enable forwarding in the NS (not on by default)
		ip netns exec wg${i} sysctl -w net.ipv6.conf.all.forwarding=1


		ip -n wg${i} addr add fe80::${h}/64 dev wg${i}
		ip -n wg${i} addr add 192.168.255.${i}/24 dev wg${i}

		# create an ethernet pair and send it int another NS so we can test all allowed
		ip netns add cont${i}
		ip -n cont${i} link set lo up
		ip link add br-${i} type bridge
		ip link set br-${i} up
		ip link add wg2br${i} type veth peer name br2wg${i}
		ip link add cont2br${i} type veth peer name br2cont${i}
		ip link set br2wg${i} master br-${i}
		ip link set br2cont${i} master br-${i}
		ip link set br2wg${i} up
		ip link set br2cont${i} up
		ip link set wg2br${i} netns wg${i}
		ip link set cont2br${i} netns cont${i}
		ip -n wg${i} link set wg2br${i} up
		ip -n cont${i} link set cont2br${i} up
		ip -n wg${i} addr add 2001:1:1:${h}::1/64 dev wg2br${i}
		ip -n cont${i} addr add 2001:1:1:${h}::200/64 dev cont2br${i}
		ip -n cont${i} route add default via 2001:1:1:${h}::1


	done
	echo
}

function addroutes() {
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		for wg in $(seq 1 $NUM); do
			if [ "$wg" -ne "$i" ]; then
				h=$(printf '%x' $wg)
				# echo ip -n wg${i} route add 2001:1:1:${h}::/64 via fe80::${h} dev wg${i}
				ip -n wg${i} route add 2001:1:1:${h}::/64 via fe80::${h} dev wg${i}
			fi
		done
		# default route is via wg1 for all except wg1 itself
		[ "$i" -ne "1" ] && ip -n wg${i} route add default via fe80::1 dev wg${i}
	done
}

function exitNR() {
	# the first is the exit NR
	ip link add exit1 type veth peer name exit1tobr
	ip link set exit1 netns wg1
	ip -n wg1 link set exit1 up
	ip link set exit1tobr up
	ip netns exec wg1 setconf wg1 wg1.conf
}
function deleteall() {
	for i in $(seq 1 $NUM); do
	    ip -n wg${i} link del wg2br${i}
		ip -n cont${i} link del cont2br${i}
		ip link del br-${i}
		ip netns del wg${i}
		ip netns del cont${i}
	done
	rm -f wg*

}

function main() {
	genkeys
	echo
	genconf
	echo
	ns
	echo
	addroutes
	echo
}

# main
