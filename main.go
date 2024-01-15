package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	debug   = flag.Bool("debug", false, "Enable debug logging")
	help    = flag.Bool("help", false, "Display usage information")
	version = flag.Bool("version", false, "Display version information")
)

// The CommitHash and Revision variables are set during building.
var (
	CommitHash = "<not set>"
	Revision   = "<not set>"
)

var (
	configCount = 0
)

// Topics describes a PubSub topic and its subscriptions.
type Topics map[string][]string

func versionString() string {
	return fmt.Sprintf("pubsubc - build %s (%s) running on %s", Revision, CommitHash, runtime.Version())
}

// debugf prints debugging information.
func debugf(format string, params ...interface{}) {
	if *debug {
		fmt.Printf(format+"\n", params...)
	}
}

// fatalf prints an error to stderr and exits.
func fatalf(format string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, os.Args[0]+": "+format+"\n", params...)
	os.Exit(1)
}

// warnf prints an error to stderr
func warnf(format string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, os.Args[0]+": WARNING "+format+"\n", params...)
}

// create a connection to the PubSub service and create topics and subscriptions
// for the specified project ID.
func create(ctx context.Context, projectID string, topics Topics) error {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		fatalf("Unable to create client to project %q: %s", projectID, err)
	}
	defer client.Close()

	debugf("Client connected with project ID %q", projectID)

	for topicID, subscriptions := range topics {

		debugf("  Checking for existing topic %q", topicID)
		topic := client.Topic(topicID)
		exists, err := topic.Exists(ctx)
		if err != nil {
			return fmt.Errorf("Failed to check exisitence of topic %q for project %q: %s", topicID, projectID, err)
		}

		if exists {
			debugf("  Topic %q already exists", topicID)
		} else {
			debugf("  Creating topic %q", topicID)
			topic, err = client.CreateTopic(ctx, topicID)
			if err != nil {
				return fmt.Errorf("Unable to create topic %q for project %q: %s", topicID, projectID, err)
			}
		}

		for _, subscription := range subscriptions {
			subscriptionParts := strings.Split(subscription, "+")
			subscriptionID := subscriptionParts[0]
			if len(subscriptionParts) > 1 {
				pushEndpoint := strings.Replace(subscriptionParts[1], "|", ":", 1)
				debugf("    Creating push subscription %q with target %q", subscriptionID, pushEndpoint)
				pushConfig := pubsub.PushConfig{Endpoint: "http://" + pushEndpoint}
				_, err = client.CreateSubscription(
					ctx,
					subscriptionID,
					pubsub.SubscriptionConfig{Topic: topic, PushConfig: pushConfig},
				)
				if err != nil {
					return fmt.Errorf("Unable to create push subscription %q on topic %q for project %q using push endpoint %q: %s", subscriptionID, topicID, projectID, pushEndpoint, err)
				}
			} else {
				debugf("    Creating pull subscription %q", subscriptionID)
				_, err = client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{Topic: topic})
				if err != nil {
					return fmt.Errorf("Unable to create subscription %q on topic %q for project %q: %s", subscriptionID, topicID, projectID, err)
				}
			}
		}
	}

	return nil
}

func processDockerLabelConfig() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		warnf("Unable to create Docker client: %s", err.Error())
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			debugf("Unable to connect to Docker: %s", err.Error())
			return
		}
		warnf("Unable to fetch Docker containers: %s", err.Error())
		return
	}

	debugf("Looking for Docker label configs")

	for _, container := range containers {
		debugf("Found container [%s] names %s", container.ID[:10], container.Names)
		for key, value := range container.Labels {
			labelKeyParts := strings.Split(key, ".")
			if "pubsubc" == labelKeyParts[0] {
				processConfigString(value, fmt.Sprintf("%s %s", container.ID[:10], key))
			}
		}
	}
}

func processConfigString(config string, sourceHint string) {
	configCount++

	// Separate the projectID from the topic definitions.
	configParts := strings.Split(config, ",")
	if len(configParts) < 2 {
		warnf("%s: Expected at least 1 topic to be defined", sourceHint)
		return
	}

	// Separate the topicID from the subscription IDs.
	topics := make(Topics)
	for _, part := range configParts[1:] {
		topicParts := strings.Split(part, ":")
		topics[topicParts[0]] = topicParts[1:]
	}

	// Create the project and all its topics and subscriptions.
	if err := create(context.Background(), configParts[0], topics); err != nil {
		warnf("%s: When creating resources: %s", sourceHint, err.Error())
	}
}

func processEnvConfig() {
	debugf("Looking for environment variable configs")

	// Cycle over the numbered PUBSUB_PROJECT environment variables.
	for i := 1; ; i++ {
		// Fetch the enviroment variable. If it doesn't exist, break out.
		currentEnv := fmt.Sprintf("PUBSUB_PROJECT%d", i)
		env := os.Getenv(currentEnv)
		if env == "" {
			break
		}
		processConfigString(env, currentEnv)
	}
}

func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Println()
		fmt.Println("Configure with environment variables:")
		fmt.Println(`   PUBSUB_PROJECT1="project1,topic1,topic2:subscription1,topic3:subscription2+endpoint1"`)
		fmt.Println()
		fmt.Println("Configure with Docker labels:")
		fmt.Println(`   pubsubc.config1="project1,topic1,topic2:subscription1,topic3:subscription2+endpoint1"`)
		fmt.Println()
		fmt.Printf(`Usage: %s`+"\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println()
	}

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Println(versionString())
		return
	}

	// Process any ENV variables & Docker labels
	processEnvConfig()
	processDockerLabelConfig()

	// If the discovered config count is zero, print the usage info.
	if 0 == configCount {
		fmt.Println("No Pub/Sub configurations found")
		flag.Usage()
		os.Exit(1)
	}
}
