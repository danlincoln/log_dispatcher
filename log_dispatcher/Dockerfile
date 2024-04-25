FROM golang:latest
WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
COPY static/ ./static/
RUN go build -v -o ./app .
CMD [ "./app" ]
