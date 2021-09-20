#!/bin/sh

DOMAIN="$2"
DOMAIN=${DOMAIN#"*."}
DOMAIN=${DOMAIN%%"."}
TOKEN="$3"
IP=`ip -4 addr show public | grep inet | tr -s ' ' | cut -d' ' -f3 | cut -d'/' -f1 | tail -n 1`
PIDFILE="/tmp/certbot-dnsmasq.pid"
CONFFILE="/tmp/dnsmasq.conf"

if [ -z "$IP" ]; then
    echo no public ip found
    exit 1
fi

if [ $1 == "present" ]; then
    if [ -f $PIDFILE ]; then
        PID=`cat $PIDFILE`
        kill $PID
        while kill -0 $PID; do 
            sleep 1
        done
    fi
    cp %[1]s/dnsmasq.conf $CONFFILE
    sed -i "s/DOMAIN/$DOMAIN/g" $CONFFILE
    sed -i "s/TOKEN/$TOKEN/g" $CONFFILE
    sed -i "s/IP/$IP/g" $CONFFILE
    dnsmasq -C $CONFFILE
elif [ $1 == "timeout" ]; then
    echo '{"timeout": 30, "interval": 5}'
elif [ $1 == "cleanup" ]; then
    # kill dnsmasq and remove related files
    [ -f $PIDFILE ] && kill `cat $PIDFILE`
    rm -rf $CONFFILE $PIDFILE
fi