# Service integration process

## Development Requirements

* A `/healtcheck` endpoint
* Correct Logging tracing `X-Request-Id` is mandatory (`X-Real-IP` is also correctly set by the proxy if you need)
* The proxy also forward a specific field name `X-Plan-Metadata` containing the metadata defined in the bought product price. Doing so helps the service taking decisions based on the plan/product the user bought.
* The gateway also forward the stripe customer id in : `X-Stripe-Customer-Id`
* A unique name not containing any space or special character. it'll be your service slug
* A Dockerfile building a standalone container (if you need a database, embed it in your docker)

## Deployments

### Creating the gateway service

#### Free Service

If your service if free the payload below is sufficient to create your service:

```json
{
    "name": "amaury-brisou",
    "prefix": "/amaury-brisou",
    "domain": "blog.puzzledge.org",
    "host": "http://172.20.0.4"
}
```

#### Paid Service

If your service isn't free, you have to create a stripe pricing table. Stripe will give you this kind of snippet:

```html
<script async src="https://js.stripe.com/v3/pricing-table.js"></script>
<stripe-pricing-table pricing-table-id="{{.service.PricingTableKey}}"
    publishable-key="{{.service.PricingTablePublishableKey}}"
    client-reference-id="{{.service.ID}}"    
    >
</stripe-pricing-table>
```

retrieve `publishable-key` and `client-reference-id` to build the servicre creation payload like below:

```json
{
    "name": "hello",
    "required_roles": ["hello"],
    "prefix": "/hello",
    "domain": "hello.test",
    "host": "http://localhost:8092",
    "pricing_table_key": "replace with your",
    "pricing_table_publishable_key": "replace with your"
}
```

:warning: You also need to configure a stripe webhook to point to the gateway webhook: <https://gw.puzzledge.org/payment/webhook>

## Reserved routes

A list of service prefixes (and all sub routes) are reserved for internal usage:

* /pricing
* /auth
* /login
* /services
* /payment

:notebook: a service could be modified if in any case a reserved keyword would be added in the future.
