FROM cassandra:5.0.6

ARG JMX_EXPORTER_VERSION=1.5.0

RUN mkdir /prometheus
ADD "https://github.com/prometheus/jmx_exporter/releases/download/$JMX_EXPORTER_VERSION/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar" /prometheus
RUN chmod 644 /prometheus/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar
ADD ./test/docker/cassandra/cassandra-jmx-exporter.yml /prometheus/cassandra-jmx-exporter.yml

# Rotate debug.log every minute, and compress it with gzip, not zip
# RUN sed -i -e 's~debug.log.%d{yyyy-MM-dd}.%i.zip~debug.log.%d{yyyy-MM-dd-HH-mm}.%i.gz~g' /etc/cassandra/logback.xml
RUN sed -i -e "s~debug.log.%d{yyyy-MM-dd}.%i.zip~debug.log.%d{\"yyyy-MM-dd'T'HH-mm\"}.%i.gz~g" /etc/cassandra/logback.xml

# Keep at least 60 min of debug.log and system.log history
RUN sed -i -e "s~<maxHistory>7</maxHistory>~<maxHistory>60</maxHistory>~g" /etc/cassandra/logback.xml

# "nodetool status" may complain about "Address already in use", do not pay attention to it, the exporter is working - see in Prometheus console
ENV JVM_OPTS "$JVM_OPTS -javaagent:/prometheus/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar=7070:/prometheus/cassandra-jmx-exporter.yml"

EXPOSE 7070
