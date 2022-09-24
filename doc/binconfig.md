# Toolbelt and Daemon configuration

Executables that use Capillaries need to be able to access the message queue (RabbitMQ) and the database (Cassandra). There are also some settings that may be helpful during troubleshooting and performance tuning in specific environments. All these settings are managed by EnvConfig (env_config.json file residing in the binary's directory).

## handler_executable_type
Name of the [queue](glossary.md#processor-queue) this executable consumes messages from.

## cassandra

Cassandra-related settings, mostly mimicking [gocql.ClusterConfig](#https://pkg.go.dev/github.com/gocql/gocql#ClusterConfig) settings.

### hosts
List of host names/addresses passed to gocql.NewCluster

### port
Port number (usually 9042), passed to gocql.ClusterConfig.Port

### username
As is, passed to gocql.PasswordAuthenticator

### password
As is, passed to gocql.PasswordAuthenticator

### keyspace_replication_config
The string passed to "CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION =" when a [keyspace](glossary.md#keyspace) is created

### writer_workers
Capillaries processors that write to [data tables](glossary.md#data-table) produce data at a rate much higher than a single-thread code writing to Cassandra can handle. Capillaries inserts into data and index from multiple threads, and the number of those threads is specified here. 10-20 threads may be considered conservative, 100 threads is more aggressive. Choose these settings according to your hardware environment specifics. 

### num_conns
Passed to gocql.ClusterConfig.NumConns

### timeout
Milliseconds, passed to gocql.ClusterConfig.Timeout

### connect_timeout
Milliseconds, passed to gocql.ClusterConfig.ConnectTimeout

## amqp
RabbitMQ settings, used in [github.com/rabbitmq/amqp091-go](#https://pkg.go.dev/github.com/rabbitmq/amqp091-go)

### url
RabbitMQ url, passed to amqp.Dial()

### exchange
Name of RabbitMQ exchange used by the daemon/toolbelt to send messages passed to amqp.Channel.ExchangeDeclare()

### prefetch_count
As is, passed to amqp.Channel.Qos()

### prefetch_size
As is, passed to amqp.Channel.Qos()

## custom_processors
Placeholder for [custom processor](glossary.md#table_custom_tfm_table) configurations.

## thread_pool_size
Number of threads processing RabbitMQ messages consumed by the binary. Choose this setting according to your hardware environment specifics.

Default: 5 threads

## ca_path
Path to the directory containing PEM certificates for all supported CAs. Required only if any of the following is referenced by HTTPS:
- script file
- script parameter file
- [tag_criteria_uri](glossary.md#tag_criteria_uri)

To obtain the PEM cert, navigate to the file URI with a browser, open certificate information, navigate to the root certificate, save it as DER/CER (say, digicert.cer), and convert it to pem using this command:
```
openssl x509 -inform der -in digicert.cer -out digicert.pem
```
and copy the result PEM file to ca_path location. Do not pollute ca_path directory with unused certificates.

## dead_letter_ttl
x-message-ttl setting passed to amqp.Channel.QueueDeclare(). After RabbitMQ detects a message that was consumed but not handled successfully (actively rejected or not acknowledged), it places the message in the dead letter queue, where it resides for dead_letter_ttl milliseconds and RabbitMQ makes another delivery attempt.

Default: 1000 milliseconds

Read more about [Capillaries dead-letter-exchange](qna.md#dead-letter-exchange).

## zap_config
Directly deserialized to [zap.Config](https://pkg.go.dev/go.uber.org/zap#Config)