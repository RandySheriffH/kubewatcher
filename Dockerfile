FROM debian
COPY ./bin/kubewatcher /kubewatcher
ENTRYPOINT /kubewatcher

