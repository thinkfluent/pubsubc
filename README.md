# Pub/Sub Emulator Config
Creates topics and subscriptions for Google Pub/Sub local emulation.

## Usage

You can define topics & subscriptions in two ways
- **Environment variables** using `PUBSUB_PROJECT1=...`
- **Docker labels** using `pubsubc.config1=...`

### Environment Variables
The code looks for environment consecutive variables like `PUBSUB_PROJECT1` containing a comma separated string. 

The first item in the the string is the project name and subsequent items are the topics. 

Each topic is a colon delimited string where the first item is the topic name and subsequent items are subscriptions to that topic.

#### Example:
```
PUBSUB_PROJECT1=project-name,topic1,topic2:subscription1:subscription2
PUBSUB_PROJECT2=project-two,topicA,topicB:subscriptionX:subscriptionY
```

### Push Subscriptions
The subscription string can be used to create a push subscription by appending the push endpoint to it separated by a `+`.

#### Examples:
**NOTE** We cannot use `:` in the URLs as it conflicts with the pre-existing topic delimiter. So we replace `|` with `:` for you. If you do not provide a protocol, we assume http.

```
PUBSUB_PROJECT1=project-name,topic:push-subscription+endpoint
```
```
PUBSUB_PROJECT1=project-name,topic:push-subscription+https|//endpoint/path
```
```
PUBSUB_PROJECT1=project-name,topic:push-subscription+http|//endpoint|8080/path
```

## Docker Labels
When using this tool as part of a larger collection of applications, we support reading project/topic/subscription 
configurations directly from the Docker daemon, using the labels of other containers.

This means that this single container can support many projects without needing custom configuration for each one. 
Instead, your application dockerfiles define their Pub/Sub needs in a declarative way. Other projects like Traefik 
adopt this paradigm too.

For example (partial Dockerfile):
```yaml
version: '3.7'
services:

  # Your application service
  myapp:
    image: busybox:latest
    environment:
      - PUBSUB_EMULATOR_HOST=pubsub-emulator:8681
    labels:
      - 'pubsubc.config1=project-one,topic1,topic2:subscription1:subscription2'
      - 'pubsubc.config2=project-two,topic:push-subscription+endpoint'

  # Pub/Sub Emulator. We mount the docker socket so we can user the Docker API
  pubsub-emulator:
    image: fluentthinking/gcloud-pubsub-emulator:latest
    ports:
      - '8681:8681'
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

### TODO:
- Push subscriptions currently only support HTTP; it would be good to support HTTP _and_ HTTPS
- Push subscription Ack Deadline is explicitly set to 60s; should be configurable