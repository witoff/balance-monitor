Balance Monitor
===============

Monitor the balance of a crypto address and alert when it's emptied. 

Currently only supports litecoin addresses (and not well at that!)

### Usage 

```shell
# Create your config file 
cp config.template.yaml config.yaml 

# Build 
docker build -t balance_monitor . 

# Run 
docker run \
    -e SENDGRID_API_KEY=xxx \ 
    balance_monitor
``` 


### Deploy

First, publish your image to your favorite location (e.g. dockerhub, container registry) and update `containers.image` in cron.yaml to match.

```shell 
# Publish 
IMAGE="gcr.io/${PROJECT_ID}/balance_monitor:master"
gcloud docker -- tag balance_monitor ${IMAGE}
gcloud docker -- push ${IMAGE}

# Deploy with Kubernetes
kubectl create -f cron.yaml
```

### TODO

* Support more alert providers and separate from main.go
* Make deployment eaisier
* Write tests & add CI 
* Remove config.yaml from Docker Image