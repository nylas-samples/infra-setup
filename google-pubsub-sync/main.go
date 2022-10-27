package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	pubsub "cloud.google.com/go/pubsub"
	"google.golang.org/api/googleapi"
	iam "google.golang.org/api/iam/v1"
)

// ENV
// valid values for ENV are
// * "STAGING"
// * "US"
// * "EU"
var ENV = "US"

// GCP Project ID
const ProjectId = "YOUR_GCP_PROJECT_ID_HERE"

func fetchOrCreateServiceAccount(ctx *context.Context, name string) (*iam.ServiceAccount, error) {
	service, err := iam.NewService(*ctx)

	if err != nil {
		return nil, fmt.Errorf("iam.NewService: %v", err)
	}
	serviceAccountUrl := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", ProjectId, name, ProjectId)
	account, err := service.Projects.ServiceAccounts.Get(serviceAccountUrl).Context(*ctx).Do()
	if err != nil {
		googleErr, ok := err.(*googleapi.Error)
		if ok && googleErr.Code == 404 {
			account = nil

			fmt.Printf("Could not find service account, creating...\n")
		} else {
			return nil, err
		}
	}

	if account == nil {
		request := &iam.CreateServiceAccountRequest{
			AccountId: name,
			ServiceAccount: &iam.ServiceAccount{
				DisplayName: name,
			},
		}

		projectUrl := fmt.Sprintf("projects/%s", ProjectId)

		account, err = service.Projects.ServiceAccounts.Create(projectUrl, request).Context(*ctx).Do()
		if err != nil {
			googleErr, ok := err.(*googleapi.Error)
			if ok && googleErr.Code == 409 && strings.Contains(googleErr.Message, "already exists") {
				fmt.Printf("Service account already exists, skipping create\n")
			}

			return nil, fmt.Errorf("Projects.ServiceAccounts.Create: %v", err)
		}

		if account == nil && account.Name != "" {
			if err != nil {
				return nil, fmt.Errorf("Projects.ServiceAccounts.Get: %v", err)
			}
		}
	}

	fmt.Printf("Created/fetched service account: %s\n", account.Email)

	setIamRequest := &iam.SetIamPolicyRequest{
		Policy: &iam.Policy{
			Bindings: []*iam.Binding{{
				Members: []string{"serviceAccount:" + account.Email},
				Role:    "roles/iam.serviceAccountTokenCreator",
			}},
		},
	}

	iamPolicy, err := service.Projects.ServiceAccounts.SetIamPolicy(account.Name, setIamRequest).Do()
	if err != nil {
		return nil, fmt.Errorf("Projects.ServiceAccounts.Create: %v", err)
	}

	fmt.Printf("added role to service account: %s\n", iamPolicy.Bindings[0].Role)

	return account, nil
}

func fetchOrCreateTopic(ctx *context.Context, topicID string) (*pubsub.Topic, error) {

	client, err := pubsub.NewClient(*ctx, ProjectId)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %v", err)
	}

	topic := client.Topic(topicID)

	topicExists, err := topic.Exists(*ctx)
	if err != nil || !topicExists {
		config := &pubsub.TopicConfig{}

		newTopic, err := client.CreateTopicWithConfig(*ctx, topicID, config)
		if err != nil {
			return nil, fmt.Errorf("CreateTopic: %v", err)
		}

		fmt.Printf("Topic created: %v\n", newTopic)

		return newTopic, nil
	}

	fmt.Printf("Topic already exists\n")

	return topic, nil
}

func validateSubscriptionConfig(subscriptionConfig *pubsub.SubscriptionConfig, topic *pubsub.Topic, config *pubsub.PushConfig) bool {
	existingTopic := subscriptionConfig.Topic
	existingPushConfig := subscriptionConfig.PushConfig

	existingAuth, ok := existingPushConfig.AuthenticationMethod.(*pubsub.OIDCToken)
	if !ok {
		return false
	}

	correctAuth, ok := pubsub.PushConfig{}.AuthenticationMethod.(*pubsub.OIDCToken)
	if !ok {
		return false
	}

	expirationDuration := subscriptionConfig.ExpirationPolicy

	if expirationDuration == nil || expirationDuration != time.Duration(0) {
		return false
	}

	return existingTopic.ID() == topic.ID() && existingPushConfig.Endpoint == config.Endpoint && existingAuth.ServiceAccountEmail == correctAuth.ServiceAccountEmail
}

func getEndpoint() (string, error) {
	if ENV == "US" {
		return "https://gmailrealtime.us.nylas.com", nil
	} else if ENV == "EU" {
		return "https://gmailrealtime.eu.nylas.com", nil
	} else if ENV == "STAGING" {
		return "https://gmailrealtime-stg.us.nylas.com", nil
	}

	return "", errors.New("supplied environment that Nylas does not support")
}

func fetchOrCreateSubscription(ctx *context.Context, subID string, topic *pubsub.Topic, serviceAccount *iam.ServiceAccount) (*pubsub.Subscription, error) {
	endpoint, err := getEndpoint()
	if err != nil {
		return nil, fmt.Errorf("did not create a subscription since %s", err.Error())
	}

	client, err := pubsub.NewClient(*ctx, ProjectId)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(subID)

	subExists, err := sub.Exists(*ctx)
	if err != nil {
		fmt.Printf("Error checking if subscription exists %v\n", err)
	}

	pushConfig := pubsub.PushConfig{
		Endpoint: endpoint,
		AuthenticationMethod: &pubsub.OIDCToken{
			ServiceAccountEmail: serviceAccount.Email,
		},
	}

	if subExists {
		fmt.Printf("Subscription already exists, validating subscription...\n")

		var existingConfig pubsub.SubscriptionConfig

		existingConfig, err = sub.Config(*ctx)
		if err != nil {
			fmt.Printf("Failed to validate existing subscription config with error %v\n", err)
			return nil, err
		}

		if validateSubscriptionConfig(&existingConfig, topic, &pushConfig) {
			fmt.Printf("Subscription is valid\n")
			return sub, nil
		} else {
			_, err = sub.Update(*ctx, pubsub.SubscriptionConfigToUpdate{
				PushConfig:       &pushConfig,
				ExpirationPolicy: time.Duration(0), // never expire
			})
			if err != nil {
				fmt.Printf("Failed to update existing subscription config with error %v\n", err)
				return nil, err
			}
			fmt.Printf("Subscription is not valid, updating...\n")
			return sub, nil
		}
	}

	sub, err = client.CreateSubscription(*ctx, subID, pubsub.SubscriptionConfig{
		Topic:            topic,
		PushConfig:       pushConfig,
		ExpirationPolicy: time.Duration(0), //never expire
	})
	if err != nil {
		return nil, fmt.Errorf("CreateSubscription: %v\n", err)
	}

	fmt.Printf("Created subscription: %v\n", sub)

	return sub, nil
}

func main() {
	ctx := context.Background()
	serviceAccount, err := fetchOrCreateServiceAccount(&ctx, "nylas-gmail-realtime")
	if err != nil {
		fmt.Printf("Failed to create service account with error %v, exiting\n", err)
		os.Exit(1)
	}

	topic, err := fetchOrCreateTopic(&ctx, "nylas-gmail-realtime")

	if err != nil {
		fmt.Printf("Failed to create topic with error %v, exiting\n", err)
		os.Exit(1)
	}

	iamHandler := topic.IAM()

	policy, err := iamHandler.Policy(ctx)
	if err != nil {
		fmt.Printf("Failed to fetch topic IAM policy with error %v, exiting\n", err)
		os.Exit(1)
	}

	policy.Add("serviceAccount:gmail-api-push@system.gserviceaccount.com", "roles/pubsub.publisher")

	err = iamHandler.SetPolicy(ctx, policy)
	if err != nil {
		fmt.Printf("Failed to set the topic IAM policy with error %v, exiting\n", err)
		os.Exit(1)
	}

	_, err = fetchOrCreateSubscription(&ctx, "push-nylas-gmail-realtime-sub", topic, serviceAccount)

	if err != nil {
		fmt.Printf("Failed to create subscription with error %v, exiting\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully setup GCP project %s for realtime google email sync", ProjectId)
}
