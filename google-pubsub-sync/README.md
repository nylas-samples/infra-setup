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

3. Edit the `main.go` file in this directory. Set the `ENV` variable to the correct region, and the `ProjectId` constant to your GCP project name

4. Fetch the dependencies.

```bash
go get .
```

5. Run the program.

```bash
go run main.go
```
