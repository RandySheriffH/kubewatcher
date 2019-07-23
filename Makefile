all: ; go build -o ./bin/kubewatcher main.go
docker-build: ; docker build . -t randysheriff/kubewatcher
docker-push: ; docker push randysheriff/kubewatcher
clean: ; rm -rf ./bin
