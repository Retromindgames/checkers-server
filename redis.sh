#!/bin/sh

ANNOUNCE_IP=$(hostname -i)
ANNOUNCE_PORT=$1
ANNOUNCE_BUS_PORT=$((ANNOUNCE_PORT + 10000))

CONF_FILE="/tmp/redis.conf"

cat > $CONF_FILE <<EOF
port $ANNOUNCE_PORT
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000
appendonly yes
loglevel notice
requirepass SUPER_SECRET_PASSWORD
masterauth SUPER_SECRET_PASSWORD
protected-mode no
cluster-announce-ip $ANNOUNCE_IP
cluster-announce-port $ANNOUNCE_PORT
cluster-announce-bus-port $ANNOUNCE_BUS_PORT
EOF

exec redis-server $CONF_FILE
