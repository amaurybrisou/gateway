# Service integration process

## Requirements

Every service must handle the request header X-Request-Id.

* Correct Logging tracing X-Request-Id is mandatory
* A /healtcheck endpoint
* A webhook pointing to the gateway website address
* A unique name
