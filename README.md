# Pub/Sub Emulator Config
Creates topics and subscriptions for Google Pub/Sub local emulation. The code expects an environment variable named `PUBSUB_PROJECT1` containing a comma separated string. The first item in the the string is the project name and subsequent items are the topics. Each topic is a colon delimited string where the first item is the topic name and subsequent items are subscriptions to that topic.

### Example:
```
PUBSUB_PROJECT1=project-name,topic1,topic2:subscription1:subscription2
```

## Push Subscriptions
The subscription string can be used to create a push subscription by appending the push endpoint to it separated by a `+`.

### Example:
```
PUBSUB_PROJECT1=project-name,topic:push-subscription+endpoint
```

### TODO:
- Push subscriptions currently only support HTTP; it would be good to support HTTP _and_ HTTPS
- Push subscription Ack Deadline is explicitly set to 60s; should be configurable