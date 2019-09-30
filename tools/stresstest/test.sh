#!/bin/bash
set -e

# default fallback values
dfarmid="CemYjciEmuvYVKDFXYaZLdGsCdLDRp4U1Xu1LPPrQNkK"
dtnodb="https://bcdb.dev.grid.tf"
dredis="10.4.0.250"

# initializing variables
nodeid=""
farmid=""
tnodb=""

# default duration of provisioning
duration="20m"

# forward logs to this redis server
redislog=""
redischan="debug-$(date +%s)"

# debug will enable lot of verbosity
# values are 'true' or 'false'
debug="false"

blue="\033[1;34m"
green="\033[1;32m"
red="\033[1;31m"
nc="\033[0m"

dependencies() {
    if ! which curl > /dev/null 2>&1; then
        echo "[-] missing command: curl"
        exit 1
    fi

    if ! which jq > /dev/null 2>&1; then
        echo "[-] missing command: jq"
        exit 1
    fi
}

setup() {
    if [ "$debug" == "true" ]; then
        echo "[+] enabling debugging"
        set -x
    fi

    schemas="${PWD}/schemas"

    echo "[+] setting up environment"
    rm -rf ${schemas}
    mkdir -p ${schemas}

    # updating internal arguments
    tfubin="${PWD}/../tfuser/tfuser"
    tfubin="${tfubin} --tnodb ${tnodb} --provision ${tnodb}"
    seed="${schemas}/user.seed"

    # initialize tests array
    tests=()
    testsname=()
}

identity() {
    echo "[+] generating identity"
    $tfubin id -o ${schemas}/user.seed > /dev/null

    identity=$($tfubin id -o ${schemas}/user.seed | grep 'identity:' | awk '{ print $2 }')

    echo "[+] identity: ${identity}"
}

select_node() {
    if [ "$nodeid" == "" ]; then
        echo "[+] selecting one node in the farm"
        fnodesjson=$(curl -s "${tnodb}/nodes?farm=${farmid}")
        node=$(echo "$fnodesjson" | jq -r '.[0].node_id')

    else
        echo "[+] using preselected node: $nodeid"
        node=$nodeid
    fi
}

generate_network() {
    echo "[+] fetching nodes list"
    nodesjson=$(curl -s "${tnodb}/nodes")

    echo "[+] selecting one exit node"
    exitnode=$(echo "$nodesjson" | jq -r '.[] | select(.exit_node > 0) | .node_id' | head -1)

    echo "[+] exit node selected: $exitnode"

    # echo "[+] creating a new network"
    # $tfubin generate network create --node $exitnode > ${schemas}/net-init.json
    # netid=$(cat ${schemas}/net-init.json | python -m json.tool | grep network_id | awk -F'"' '{ print $4 }')

    # echo "[+] add the node into the network"
    # $tfubin generate --schema ${schemas}/net-init.json network add-node --node $node

    # echo "[+] adding user to network"
    # wgkey=$($tfubin generate --schema ${schemas}/net-init.json network add-user --user ${identity} | head -1 | awk '{ print $4 }')

    # echo "[+] generating wireguard config"
    # $tfubin generate --schema ${schemas}/net-init.json network wg --user ${identity} --key ${wgkey} > ${schemas}/wg.conf
}

generate_debug() {
    echo "[+]   generating debug mode (redis ${redislog} -> ${redischan})"
    $tfubin generate debug --endpoint "${redislog}:6379" --channel ${redischan} > ${schemas}/debug-node.json
}

generate_containers() {
    echo "[+]   generating container"
    $tfubin generate container --flist https://hub.grid.tf/maxux/busybox-latest.flist --entrypoint /bin/ash --corex --network ${netid} --envs hello=world > ${schemas}/busybox-corex.json
}

generate_zdb() {
    echo "[+]   generating zdb profiles"
    $tfubin generate storage zdb --size 10 --type SSD --mode user > ${schemas}/zdb-ssd-10.json
    $tfubin generate storage zdb --size 100 --type SSD --mode user > ${schemas}/zdb-ssd-100.json
    $tfubin generate storage zdb --size 200 --type SSD --mode user > ${schemas}/zdb-ssd-200.json
    $tfubin generate storage zdb --size 100 --type HDD --mode user > ${schemas}/zdb-hdd-100.json
    $tfubin generate storage zdb --size 400 --type HDD --mode user > ${schemas}/zdb-hdd-400.json
}

provision() {
    testname="$1"
    echo -n "[+]   provisioning: $testname ... "
    response=$($tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/${testname}.json)

    resource=$(echo "$response" | grep Resource | awk '{ print $2 }')
    testsname+=($testname)
    tests+=($resource)

    echo "$resource"
}

provision_network() {
    provision net-init
}

provision_debug() {
    provision debug-node
}

provision_containers() {
    provision busybox-corex
}

provision_zdb() {
    provision zdb-ssd-10
    provision zdb-ssd-100
    provision zdb-ssd-200
    provision zdb-hdd-100
    provision zdb-hdd-400
}

teststatus() {
    echo "[+]"
    echo "[+] waiting for tests result"
    echo "[+]"

    for index in "${!tests[@]}"; do
        echo -en "[+] ${blue}"
        printf "%-14s: " "${testsname[$index]}"

        while : ; do
            status=$(curl -s ${tnodb}${tests[$index]})
            if echo "$status" | jq -e '.Result == null' > /dev/null; then
                # result not yet available
                sleep 1
                continue
            fi

            if echo "$status" | jq -e '.Result.error == ""' > /dev/null; then
                echo -en "${green}"
                echo -en "success${nc}, data: "
                echo "$status" | jq '.Result.data'

            else
                echo -en "${red}"
                echo -en "failed${nc}, error: "
                echo "$status" | jq -r '.Result.error'
            fi

            break
        done

        echo -en "\033[0m"
    done
}

usage() {
    echo "Usage: $0 [-n nodeid] [-f farmid] [-t tnourl] [-r redis-endpoint]"
    echo ""
    echo "Default values:"
    echo "  farmid: $dfarmid"
    echo "  nodeid: (random within the farm)"
    echo "   tnodb: $dtnodb"
    echo "   redis: $dredis"
}

options() {
    while getopts "n:f:t:r:" arg; do
        case "${arg}" in
            n)
                nodeid=${OPTARG} ;;
            f)
                farmid=${OPTARG} ;;
            t)
                tnodb=${OPTARG} ;;
            r)
                redislog=${OPTARG} ;;
            *)
                usage
                exit 1
                ;;
        esac
    done
    shift $((OPTIND-1))

    farmid=${farmid:-$dfarmid}
    tnodb=${tnodb:-$dtnodb}
    redislog=${redislog:-$dredis}

    echo -e "[+] farm id: ${blue}${farmid}${nc}"
    echo -e "[+] tnodb url: ${blue}${tnodb}${nc}"
    echo -e "[+] redis link: ${blue}${redislog}${nc} / ${blue}${redischan}${nc}"
}

main() {
    echo "[+] initializing stress test"

    dependencies
    options $@

    setup
    identity
    select_node

    echo "[+]"
    echo "[+] generating schemas"
    echo "[+]"

    generate_debug
    # generate_network
    # generate_containers
    generate_zdb

    echo "[+]"
    echo "[+] sending provisioning"
    echo "[+]"

    provision_debug
    # provision_network
    # provision_containers
    provision_zdb

    teststatus

    echo "[+]"
    echo "[+] stress test done"
}

main $@
