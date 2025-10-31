# CapiMQ

This is an experimental very simple message broker. Some highlights:
- connectionless HTTP communication
- consumer sends heartbeats, dead consumers detected by lack of those
- no queue/exchanges, everuthing in one space
- has knowledge about Capillaries message internals, so it can potentially reduce traffic - postpones batches for a specific node if one of the batches for that node retried; this was the main motivation behind this project
- no ambition to make it distributed/high-available
- API that allows to look into the state of the queue and manage it

Observations so far:
- connectionless -> polling -> delivery may occur later than we would like to ideally (so, on small data samples it is slower than connection-oriented AMQP 1.0)
- hearbeats vs connection monitoring: heartbeats are simpler to implement, but are prone to false positives (a big chunk of data is being processed without sending heartbeat, so a client declared dead)

