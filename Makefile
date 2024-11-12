
CRI=docker
REPO=localhost:5000
GO=go1.22.4

cleanup: stop
	$(CRI) stop registry

setup: deploy-repo

start:
	$(CRI) compose -f scripts/docker-compose.yml up -d

stop:
	$(CRI) compose -f scripts/docker-compose.yml down

deploy-repo:
	$(CRI) run -d --rm -p 5000:5000 --name registry registry:latest

build:
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/$(comp) cmd/$(comp)/main.go

push:
	-$(CRI) rmi $(REPO)/$(comp) -f
	$(CRI) build -f dockerfiles/Dockerfile.$(comp) -t $(REPO)/$(comp) . 
	$(CRI) push $(REPO)/$(comp)

build-push: build push

