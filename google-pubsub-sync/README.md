# Pubsub-setup script
Nylas maintains this Go script that automatically provisions the correct GCP resources for you so you can enable Google PubSub Message Sync. See the [Nylas documentation on PubSub](https://developer.nylas.com/docs/the-basics/provider-guides/google/connect-google-pub-sub/) for more information.

### Running the script

1. Switch to the GCP project you want to add PubSub Message Sync to.

```bash
gcloud config set project $YOUR_PROJECT_NAME_HERE
```

2. Authenticate locally with GCP.

```bash
gcloud auth application-default login
```

3. Fetch the dependencies.

```bash
go get .
```

4. Run the program.

```bash
go run main.go --projectId $YOUR_PROJECT_NAME_HERE
```
## Flags
| Flag name  | Description                                                             | Example         |
|------------|-------------------------------------------------------------------------|-----------------| 
| `projectId` | The GCP Project to run this script against                              | test-project-id |
| `env`       | The Nylas environment to run against. Valid values are: us, eu, staging | us               |
