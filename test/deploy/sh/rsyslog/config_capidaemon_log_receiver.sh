RSYSLOG_CAPIDAEMON_CONFIG_FILE=/etc/rsyslog.d/capidaemon_receiver.conf

sudo rm -f $RSYSLOG_CAPIDAEMON_CONFIG_FILE

sudo tee $RSYSLOG_CAPIDAEMON_CONFIG_FILE <<EOF
module(load="imudp")
ruleset(name="capidaemon"){action(type="omfile" DirCreateMode="0777" FileCreateMode="0644" file="/var/log/capidaemon/capidaemon.log")}
input(type="imudp" port="514" ruleset="capidaemon")
EOF

sudo systemctl restart rsyslog
