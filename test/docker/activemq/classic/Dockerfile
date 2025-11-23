FROM apache/activemq-classic:6.1.8

# 1. Enable redeliveryPlugin
# 2. No queue definition added
COPY ./test/docker/activemq/classic/activemq.xml /opt/apache-activemq/conf/activemq.xml
# Add guest user
COPY ./test/docker/activemq/classic/login.config /opt/apache-activemq/conf/login.config
COPY ./test/docker/activemq/classic/users.properties /opt/apache-activemq/conf/users.properties
COPY ./test/docker/activemq/classic/groups.properties /opt/apache-activemq/conf/groups.properties

