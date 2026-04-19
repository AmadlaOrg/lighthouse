# lighthouse

Intelligent notification/alerting CLI for the Amadla ecosystem.

Provides alert deduplication, exponential backoff, grouping, flap detection, rate limiting, and silencing to prevent notification storms.

## Usage

```bash
# Send an alert
lighthouse notify -f alert.yaml

# Send from stdin
echo '{"source":"waiter","name":"deploy_failed","severity":"critical","labels":{"svc":"api"},"annotations":{"summary":"Deploy failed"},"status":"firing"}' | lighthouse notify -f -

# Resolve an alert
lighthouse resolve -f alert.yaml

# Silence an alert
lighthouse silence <fingerprint> --for 2h --reason "maintenance"

# Show status
lighthouse status

# List plugins
lighthouse plugins
```

## Configuration

`~/.config/lighthouse/config.yaml`

## License

MIT
