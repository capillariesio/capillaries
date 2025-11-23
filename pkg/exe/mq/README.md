# CapiMQ

This is an experimental very simple message broker. Some highlights:
- connectionless HTTP communication
- consumer sends heartbeats, dead consumers detected by lack of those
- no queue/exchanges, everuthing in one space
- postpones batches for a specific node if one of the batches for that node retried; this was the main motivation behind this project
- no ambition to make it distributed/high-available
- API that allows to look into the state of the queue and manage it

1. Message pollution by itself does not take too much CPU. But if some instances are busy with only retrying (because of prefetch, for example), overall performance degrades: 2 instances are 100% busy processing batches, 2 instances are at 20% CPU doing only retries.
2. A poll-oriented messaging in general is slower than connection-oriented. connectionless -> polling -> delivery may occur later than we would like to ideally (so, on small data samples it is slower than connection-oriented AMQP 1.0)
3. heartbeats vs connection monitoring: heartbeats are simpler to implement, but are prone to false positives (a big chunk of data is being processed without sending heartbeat, so a client declared dead). Sending hearbeats is essential for poll-oriented messaging, but deciding when to send them for each kind of processor may not be a trivial task.


Capillaries 1.2:
- drops support for AMQP 0.9.1
- adds support for AMQP 1.0
- introduces experimental CapiMQ

Some thoughts below.

1. Product lock-in. For now, RabbitMQ is sponsored by Broadcom, ActiveMQ Artemis - by IBM and RedHat, ActiveMQ Classic maintainers **seem** to be somewhat affiliated with Talend. RabbitMQ is a bigger risk as it is written in Erlang, while ActiveMQ - in Java.

2. Protocol lock-in. AMQP 0.9.1 support by other message brokers: not really growing. AMQP 1.0: support is growing. 

3. It is tempting to use simple pull model for client, no need to mask HTTP behind the session/connection abstractions. HTTP connection pool is all we need.

4. Message pollution. None of the stock message brokers I am aware of solve the problem of 1000 batches hitting workers while the node is not ready, we need a smarter message broker that knows how to delay all messages for a specific node when needed.

5. On one hand, message broker is and will be the most critical component that is not allowed to fail. On the other hand, one-server message queue deployment covers even very powerful Capillaries setups, so no need to scale. So, probably, no need to have a distributed fail-safe mb, a solid one-machine solution is fine.

6. Need better queue mgmt capabilities - admins may want to see/delete messages by ks, run id and node. In the queue and in the wip.

7. Accept the fact that exactly-once message delivery is a myth.



