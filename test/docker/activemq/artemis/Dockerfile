FROM apache/activemq-artemis:2.43.0

# 1. Do not use /var/lib/artemis-instance/etc directly
# 2. Queue definition in broker.xml is surprisingly verbose: we had to tweak address-settings, queues and addresses
COPY ./test/docker/activemq/artemis/broker.xml /var/lib/artemis-instance/etc-override/broker.xml
# guest user
COPY ./test/docker/activemq/artemis/login.config /var/lib/artemis-instance/etc-override/login.config
COPY ./test/docker/activemq/artemis/artemis-roles.properties /var/lib/artemis-instance/etc-override/artemis-roles.properties
COPY ./test/docker/activemq/artemis/artemis-users.properties /var/lib/artemis-instance/etc-override/artemis-users.properties


