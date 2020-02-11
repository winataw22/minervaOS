#!/bin/bash
#set -Eeuo pipefail
set -Eeo pipefail
DEFAULT_NETWORK_RANGE="173.30"
CIDR_16="${DEFAULT_NETWORK_RANGE}.0.0/16"
CUR_DIR=$PWD
echo "CUR_DIR:" $CUR_DIR
echo "---Building tfuser cli...."
cd ../../tools
TOOL_DIR=$PWD
make tfuser &>/dev/null
TFUSER_PATH="${TOOL_DIR}/tfuser/tfuser"
echo "TFUSER_PATH:" $TFUSER_PATH
cd $CUR_DIR

#--------------------#
# Arguments          #
#--------------------#
echo "---Analysing arguments..."
if [ -z "$1" ]; then
    echo "You need to specify your gihub account as first argument to link your ssh keys to the virtual machine"
    exit
else
    GITHUB_ACCOUNT_OR_PUBKEY=$1
    echo "Public ssh key or github Account \"$GITHUB_ACCOUNT_OR_PUBKEY\" will be pushed to the VM's authorized_keys"
fi

if [ -z "$2" ]; then
    echo "Network range /16 CIDR not specified...\"${CIDR_16}\" will be used"
else
    DEFAULT_NETWORK_RANGE=$2
    CIDR_16="${DEFAULT_NETWORK_RANGE}.0.0/16"
    echo "Network range /16 CIDR \"${CIDR_16}\" will be used"
fi

if [ -z "$3" ]; then
    CLUSTER_DURATION=1
    echo "No duration for the cluster set... cluster TTL set for 1 day"
else
    CLUSTER_DURATION=$3
    echo "duration for the cluster set to \"$CLUSTER_DURATION\""
fi

NODE_COUNT=3
if [ -z "$4" ]; then
    WORKER_NODES=2
    echo "No node numbers set... will deploy $NODE_COUNT nodes : 1 master and $WORKER_NODES workers"
else
    NODE_COUNT=$4
    WORKER_NODES=$(($4 - 1))
    echo "Will deploy \"$NODE_COUNT\" nodes : 1 master and \"$WORKER_NODES\" workers"
fi

if [ -z "$5" ]; then
    VM_SIZE=1
    echo "VM size not set... will deploy vm with size "$VM_SIZE
else
    VM_SIZE=$5
    echo "Will deploy VM with size"$VM_SIZE
fi
# We use this static node to get external access and to be the master node
FIRST_NODE="qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3"
FIRST_NODE_SUBNET=$DEFAULT_NETWORK_RANGE.2.0/24
ADD_ACCESS_SUBNET=$DEFAULT_NETWORK_RANGE.245.0/24
NODE_ID[0]="3NAkUYqm5iPRmNAnmLfjwdreqDssvsebj4uPUt9BxFPm"
SUBNET[0]=$DEFAULT_NETWORK_RANGE.3.0/24
NODE_ID[1]="FTothsg9ZuJubAEzZByEgQQUmkWM637x93YH1QSJM242"
SUBNET[1]=$DEFAULT_NETWORK_RANGE.4.0/24

# check if we can get all dev nodes
which jq &>/dev/null
retval_jq=$?
which curl &>/dev/null
retval_curl=$?

if [ $retval_jq -ne 0 ] && [ $retval_curl -ne 0 ]; then
    echo "jq or curl not found ...will use static nodeid"
else
    echo "jq and curl found listing node id..."
    # if updated difference in minutes with now is less then 10 minutes, node is up
    CUR_DATE=$(date +"%s")
    MINIMUM_UPDATED_TIME=$(($CUR_DATE - 600))
    NODES=$(curl -s -X POST "https://explorer.devnet.grid.tf/nodes/list" | jq '.nodes[] | if .updated > '${MINIMUM_UPDATED_TIME}' then .node_id else "" end' | tr "\n" ";")
    if [ $(echo $NODES | grep -c $FIRST_NODE) -ne 1 ]; then
        echo "bootsratp node $FIRST_NODE is down contact an administrator"
        #exit
    fi
    IFS=';' read -ra ADDR <<<"$NODES"
    sub_const=3
    for i in "${!ADDR[@]}"; do
        if [ "${ADDR[i]}" != "\"\"" ]; then
            NODE_ID[$i]=${ADDR[i]//\"/}
            SUBNET[$i]=$DEFAULT_NETWORK_RANGE.$(($i + $sub_const)).0/24
            printf 'NODE_ID[%s]=%s\n' "$i" "${NODE_ID[i]}"
            printf 'SUBNET[%s]=%s\n' "$i" "${SUBNET[i]}"
        fi

    done
    # master node will be the last node in the array
    #last_id=$((${#NODE_ID[@]} - 1))
    #echo $last_id
    #FIRST_NODE=${NODE_ID[$last_id]}
    #FIRST_NODE_SUBNET=${SUBNET[$last_id]}
    #echo $FIRST_NODE
    #echo $FIRST_NODE_SUBNET
fi

echo "---Setting up the network ..."
mkdir -p cluster
cd cluster
CONFIG_PATH=$PWD
NETWORK_NAME=kube-$RANDOM
SECRET=secret-$RANDOM-$RANDOM-$RANDOM

$TFUSER_PATH generate --schema network.json network create --name $NETWORK_NAME --cidr $CIDR_16
echo "Network $NETWORK_NAME : $CIDR_16  Created"

$TFUSER_PATH generate --schema network.json network add-node --node $FIRST_NODE --subnet $FIRST_NODE_SUBNET
echo "Master node $FIRST_NODE with subnet : $FIRST_NODE_SUBNET  Added to the network"
NODES_ARGUMENT=" --node $FIRST_NODE"

for ((i = 0; $WORKER_NODES - $i; i++)); do
    $TFUSER_PATH generate --schema network.json network add-node --node ${NODE_ID[$i]} --subnet ${SUBNET[$i]}
    NODES_ARGUMENT="$NODES_ARGUMENT --node ${NODE_ID[$i]}"
    echo "node $i id:${NODE_ID[$i]} subnet:${SUBNET[$i]} Added to the network"
done

echo "---Adding external access"
$TFUSER_PATH generate --schema network.json network add-access --node $FIRST_NODE --subnet $ADD_ACCESS_SUBNET --ip4 >wg.conf
echo "access granted to master node $FIRST_NODE with subnet : $ADD_ACCESS_SUBNET  wireguard configuration written to wg.conf"

IDENTITY_FILE=$CONFIG_PATH/user.seed
if [ -f "$IDENTITY_FILE" ]; then
    echo "Will use identity found at $IDENTITY_FILE"
else
    echo "Identity not found at $IDENTITY_FILE ...generating an identity"
    $TFUSER_PATH id
fi

echo "---Provisioning the network"
$TFUSER_PATH provision --schema network.json --duration $CLUSTER_DURATION --seed $IDENTITY_FILE $NODES_ARGUMENT
echo "waiting for provisioning..."
sleep 5
retval_live=$($TFUSER_PATH live --seed $IDENTITY_FILE --end 3000 | grep -c "not deployed yet" || true)
if [ $retval_live -gt 0 ]; then
    echo "waiting again for provisioning..."
    sleep 5
fi
echo "---Provisioning the Kubernetes VM"
MASTER_IP=${FIRST_NODE_SUBNET//.0\/24/}.2
$TFUSER_PATH generate --schema first_node.json kubernetes --size $VM_SIZE --network-id $NETWORK_NAME --ip $MASTER_IP --secret $SECRET --node $FIRST_NODE --ssh-keys "$GITHUB_ACCOUNT_OR_PUBKEY"
$TFUSER_PATH -d provision --schema first_node.json --duration $CLUSTER_DURATION --seed $IDENTITY_FILE --node $FIRST_NODE
echo "provisioning master node..."

for ((i = 0; $WORKER_NODES - $i; i++)); do
    IP=${SUBNET[$i]//.0\/24/}.2
    echo " - Ip:"$IP
    VM_IPS=${VM_IPS}" "${IP}
    $TFUSER_PATH generate --schema node$i.json kubernetes --size $VM_SIZE --network-id $NETWORK_NAME --ip $IP --master-ips $MASTER_IP --secret $SECRET --node ${NODE_ID[$i]} --ssh-keys "$GITHUB_ACCOUNT_OR_PUBKEY"
    $TFUSER_PATH -d provision --schema node$i.json --duration $CLUSTER_DURATION --seed $IDENTITY_FILE --node ${NODE_ID[$i]}
    echo "provisioning worker node $i with ip:$IP..."
done

echo master node ip:$MASTER_IP workers ip:$VM_IPS
echo "waiting for provisioning..."
sleep 10
wg-quick down $CONFIG_PATH/wg.conf &>/dev/null || true
wg-quick up $CONFIG_PATH/wg.conf
printf "%s" "waiting for Master node $MASTER_IP to answer ..."
while ! timeout 0.2 ping -c 1 -n $MASTER_IP &>/dev/null; do
    printf "%c" "."
done
printf "\n%s\n" "Master is online"
sleep 3
echo "Retrieving information needed to setup the cluster from the master node..."
ssh -o "StrictHostKeyChecking no" rancher@$MASTER_IP 'k3s kubectl get nodes'
scp -o "StrictHostKeyChecking no" rancher@$MASTER_IP:/etc/rancher/k3s/k3s.yaml ./kube-config.yaml
echo "\nkubeconfig file has been written to kube-config.yaml edit your ~/.kube/config accordingly"
echo "to ssh into your master node execute this command"
printf "\nssh rancher@%s\n" $MASTER_IP

exit
