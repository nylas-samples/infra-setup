# Pubsub-setup script
This is a script in Go that will automatically provision the correct GCP resources to enable Google PubSub Message Sync. See our [public docs for more info](https://developer.nylas.com/docs/the-basics/provider-guides/google/connect-google-pub-sub/)

### Running the script
1. Switch to the correct gcp project
```
gcloud config set project $YOUR_PROJECT_NAME_HERE
```
2. Authenticate locally with gcp:
```
gcloud auth application-default login
```
3. Edit the main.go file in this directory. Make sure to set the `ENV` variable to the correct region, and the `ProjectId` constant to your GCP project name
4. Fetch the dependencies
```
go get .
```
5. Run the program
```
go run main.go
```
