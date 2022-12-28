# Expecting
# BASTION_IP=10.5.0.2

RSYSLOG_CAPIDAEMON_CONFIG_FILE=/etc/rsyslog.d/capidaemon.conf

sudo rm -f $RSYSLOG_CAPIDAEMON_CONFIG_FILE

sudo tee $RSYSLOG_CAPIDAEMON_CONFIG_FILE <<EOF
module(load="imfile")
input(type="imfile" File="/var/log/capidaemon/capidaemon.log" Tag="capidaemon" Severity="info" ruleset="udp_bastion")
ruleset(name="udp_bastion") {action(type="omfwd" target="$BASTION_IP" Port="514" Protocol="udp") }
EOF

sudo systemctl restart rsyslog
