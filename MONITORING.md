# Monitoring Configuration

This document describes how to enable and configure monitoring for the Imagor service.

## Prometheus Metrics

Prometheus metrics are enabled by default when the `PROMETHEUS_BIND` environment variable is set.

### Configuration

Add the following environment variables to your `docker-compose.yml`:

```yaml
environment:
  # Prometheus Metrics Configuration
  PROMETHEUS_BIND: ":5000"
  PROMETHEUS_PATH: "/metrics"
```

### Accessing Metrics

Once enabled, metrics are available at:
- **URL**: `http://localhost:5000/metrics`
- **Port**: 5000 (exposed in docker-compose.yml)

### Available Metrics

The following metrics are collected:

- `http_request_duration_seconds` - Histogram of HTTP request latencies
  - Labels: `code` (HTTP status code), `method` (HTTP method)

### Example Prometheus Configuration

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'imagor'
    static_configs:
      - targets: ['localhost:5000']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Sentry Integration

Sentry integration provides error tracking and performance monitoring.

### Configuration

Add the following environment variable to your `docker-compose.yml`:

```yaml
environment:
  # Sentry Configuration
  SENTRY_DSN: ${SENTRY_DSN}
```

### Setup

1. Create a Sentry project at [sentry.io](https://sentry.io)
2. Get your DSN from the project settings
3. Set the `SENTRY_DSN` environment variable

### Features

- **Error Tracking**: Automatic capture of panics and errors
- **Performance Monitoring**: Request timing and performance data
- **Breadcrumbs**: Contextual information leading to errors
- **Log Integration**: Structured logging with Sentry context

### Log Levels

- **Errors and above**: Sent to Sentry
- **Info and above**: Sent as breadcrumbs for context

## Environment Variables

Create a `.env` file with the following variables:

```bash
# AWS Configuration (existing)
AWS_ACCESS_KEY_ID=your_aws_access_key_id
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key

# Sentry Configuration
SENTRY_DSN=https://your-sentry-dsn@sentry.io/project-id

# Optional: Override default Prometheus settings
# PROMETHEUS_BIND=:5000
# PROMETHEUS_PATH=/metrics
```

## Docker Compose

The `docker-compose.yml` file has been updated to include:

- Prometheus metrics endpoint on port 5000
- Sentry DSN configuration
- Access logging enabled
- CORS enabled

## Verification

After starting the service:

1. **Check Prometheus metrics**: `curl http://localhost:5000/metrics`
2. **Check Sentry integration**: Look for Sentry initialization logs
3. **Test error tracking**: Generate an error to verify Sentry capture

## Troubleshooting

### Prometheus Metrics Not Available

- Verify `PROMETHEUS_BIND` is set
- Check that port 5000 is exposed in docker-compose.yml
- Ensure the service is running: `docker-compose ps`

### Sentry Not Working

- Verify `SENTRY_DSN` is correctly set
- Check Sentry project configuration
- Look for Sentry initialization errors in logs
