FROM golang:1.18.3-alpine 
# Using Golang 1.18.3 On Alpine Linux 

LABEL Author="Klimushin Kirill"
RUN echo "Running Docker Build of the Web Infrastructure App"

# Initializing Project Directory 

CMD mkdir /project/dir/ 
WORKDIR /project/dir/ 

# Setting up Environment Variables 

ENV CGO_ENABLED=0 
ENV GOOS=linux 
ENV GOARCH=amd64 
ENV GIN_MODE=0

COPY . . 
# Installing Dependencies + Vendoring the Directory 
RUN go mod tidy && go mod vendor 

# Running Unittests 

RUN go test -v ./tests/vm/vm_test.go 
RUN go test -v ./tests/storage/storage_test.go 
RUN go test -v ./tests/suggestions/suggestions_test.go 
RUN go test -v ./tests/resources/resource_test.go 

# Building Application Main Package 
RUN go build -o ./main/. 
# Running Application, Once All Previous Steps Done Correctly
ENTRYPOINT ["go", "run", "./main/main.go"]
