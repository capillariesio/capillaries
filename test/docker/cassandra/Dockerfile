FROM cassandra:5.0.2

ARG JMX_EXPORTER_VERSION=1.0.1

RUN mkdir /prometheus
ADD "https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/$JMX_EXPORTER_VERSION/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar" /prometheus
RUN chmod 644 /prometheus/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar
ADD ./test/docker/cassandra/cassandra-jmx-exporter.yml /prometheus/cassandra-jmx-exporter.yml

ENV JVM_OPTS "$JVM_OPTS -javaagent:/prometheus/jmx_prometheus_javaagent-$JMX_EXPORTER_VERSION.jar=7070:/prometheus/cassandra-jmx-exporter.yml"

EXPOSE 7070
