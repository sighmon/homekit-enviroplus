GOCMD=go
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
GORUN=$(GOCMD) run

export GO111MODULE=on

build:
	$(GOGET)
	$(GOBUILD) homekit-enviroplus.go

run:
	$(GORUN) homekit-enviroplus.go
