#!/usr/bin/env bash

# setup connection namespaces with a dummy and add an ip
NUM=5

function prepare(){
	for i in $(seq 1 $NUM) ; do
		# ExitPoint and their IPv4
		ip netns add z${i}
		ip -n z${i} link set lo up
		ip -n z${i} link add zone${i} type dummy
		ip -n z${i} link set zone${i} up
		ip -n z${i} addr add 10.10.0.1/24 dev zone${i}
	done

	# a public IPv4 in a Nat container
	# 
	ip netns add vrf

	ip link add ivrf type veth peer name ovrf
	ip link set ivrf netns vrf
	ip -n vrf link set ivrf up
	ip -n vrf link set lo up
	ip -n vrf addr add 172.18.0.254/24 dev ivrf
	ip -n vrf link add cvrf type dummy

	for i in $(seq 1 $NUM) ; do
		ip link add oz${i} type veth peer name iz${i}
		ip link set iz${i} netns z${i}
		ip -n z${i} link set iz${i} up
		ip -n z${i} addr add 172.16.0.1/24 dev iz${i}
		ip -n z${i} route add default via 172.16.0.254
		ip link set oz${i} netns vrf
		ip -n vrf link set oz${i} up
	done

	ip link set ovrf up
}	

function delete(){
	for i in $(seq 1 $NUM) ; do
		ip netns del z${i}
		ip link del oz${i}
	done
	ip netns del vrf
	ip link del ovrf

}

