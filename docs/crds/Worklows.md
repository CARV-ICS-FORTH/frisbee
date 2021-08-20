“Fail fast”

Sometimes you may want to stop processing as soon as an error occurs. Perhaps encountering bad data is a symptom of
problems upstream that must be resolved, and there’s no point in continuing to try processing other messages.

This is the default behavior of Workflow.

Perhaps we can adopt policies from
https://www.confluent.io/blog/kafka-connect-deep-dive-error-handling-dead-letter-queues/