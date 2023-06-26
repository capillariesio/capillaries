# Make it as idempotent as possible, it can be called over and over

if [ "$INTERNAL_BASTION_IP" = "" ]; then
  echo Error, missing: INTERNAL_BASTION_IP=10.5.0.10
 exit 1
fi

RSYSLOG_CAPIDAEMON_CONFIG_FILE=/etc/rsyslog.d/capidaemon_sender.conf

sudo rm -f $RSYSLOG_CAPIDAEMON_CONFIG_FILE

sudo tee $RSYSLOG_CAPIDAEMON_CONFIG_FILE <<EOF
module(load="imfile")
input(type="imfile" File="/var/log/capidaemon/capidaemon.log" Tag="capidaemon" Severity="info" ruleset="udp_bastion")
ruleset(name="udp_bastion") {action(type="omfwd" target="$INTERNAL_BASTION_IP" Port="514" Protocol="udp") }
EOF

sudo systemctl restart rsyslog
