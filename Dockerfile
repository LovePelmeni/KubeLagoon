FROM golang:1.18.3-alpine 
LABEL Author="Klimushin Kirill"
CMD mkdir /project/dir/ 
WORKDIR /project/dir/ 

ENV CGO_ENABLED=0 
ENV GOOS=linux 
ENV GOARCH=amd64 

COPY . . 
RUN go mod tidy && go mod vendor && go mod test -v ./tests/.. 
RUN go build -o ./main/. 
ENTRYPOINT ["go", "run", "./main/main.go"]

