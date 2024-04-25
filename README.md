# log_dispatcher

This is a small learning project for Go, Kafka, and Websockets.

## Summary

The dispatcher is a http server that ingests application logs (arbitrary JSON)
and relays them to multiple destinations: a local file, a web dashboard (via
websockets), and a Kafka message queue.

The server has three endpoints:

* `/` serves the files located in `static/`.
* `/subscribe` creates a new Websocket listener (used in `static/index.html`)
* `/publish` (POST only) takes the request body and relays it to all the
    various consumers.

## Use

1. Use docker-compose to create the app cluster (`docker-compose up` in this
    directory).
2. Open a browser with the example feed page: http://localhost:8080
3. Send a POST request to the `/publish` endpoint with any body text:

        curl -X POST -H "Content-Type: application/json" -d '{"key": "value", "foo": "bar"}' http://localhost:8080/publish
4. Observe the message received on the feed page, in the local file on the
    log_dispatcher container (`/app/service.log`), and in Kafka. In the Kafka
    container, run:

        kafka-console-consumer.sh --topic logs --from-beginning --bootstrap-server localhost:9092