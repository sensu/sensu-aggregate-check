[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-aggregate-check)
# Sensu Go Aggregate Check Plugin

- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Sensu Go](#sensu-go)
    - [Asset registration](#asset-registration)
    - [Check definition](#check-definition)
  - [Sensu Core](#sensu-core)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

### Overview

An aggregate makes it possible to treat the result of multiple disparate check executions executed across multiple disparate systems as a single result (Event). Aggregates are extremely useful in dynamic environments and/or environments that have a reasonable tolerance for failure. Aggregates should be used when a service can be considered healthy as long as a minimum threshold is satisfied (e.g. are at least 5 healthy web servers? are at least 70% of N processes healthy?).

This plugin allows you to query the Sensu Go Backend API for Events matching certain criteria (labels). This plugin generates a set of counters (e.g. events total, events in an OK state, etc) from the Events query and provides several CLI arguments to evaluate the computed aggregate counters (e.g. --warn-percent=75).

### Files

N/A

## Usage examples

### Help

```
The Sensu Go Event Aggregates Check plugin

Usage:
  sensu-aggregate-check [flags]

Flags:
  -H, --api-host string          Sensu Go Backend API Host (e.g. 'sensu-backend.example.com') (default "127.0.0.1")
  -k, --api-key string           Sensu Go Backend API Key
  -P, --api-pass string          Sensu Go Backend API Password (default "P@ssw0rd!")
  -p, --api-port string          Sensu Go Backend API Port (e.g. 4242) (default "8080")
  -u, --api-user string          Sensu Go Backend API User (default "admin")
  -l, --check-labels string      Sensu Go Event Check Labels to filter by (e.g. 'aggregate=foo')
  -C, --crit-count int           Critical threshold - count of Events in warning state
  -c, --crit-percent int         Critical threshold - % of Events in warning state
  -e, --entity-labels string     Sensu Go Event Entity Labels to filter by (e.g. 'aggregate=foo,app=bar')
  -h, --help                     help for sensu-aggregate-check
  -i, --insecure-skip-verify     skip TLS certificate verification (not recommended!)
  -n, --namespaces string        Comma-delimited list of Sensu Go Namespaces to query for Events (e.g. 'us-east-1,us-west-2') (default "default")
  -s, --secure                   Use TLS connection to API
  -t, --trusted-ca-file string   TLS CA certificate bundle in PEM format
  -W, --warn-count int           Warning threshold - count of Events in warning state
  -w, --warn-percent int         Warning threshold - % of Events in warning state
```

## Configuration

### Sensu Go
#### Asset registration

Assets are the best way to make use of this plugin. If you're not using an asset, please consider doing so! If you're using sensuctl 5.13 or later, you can use the following command to add the asset:

`sensuctl asset add sensu/sensu-aggregate-check`

If you're using an earlier version of sensuctl, you can download the asset definition from [this project's Bonsai asset index page][1] or one of the existing [releases][2] or create an executable script from this source.

#### Check definition

```yaml
api_version: core/v2
type: CheckConfig
metadata:
  namespace: default
  name: dummy-app-aggregate
spec:
  runtime_assets:
  - sensu-aggregate-check
  command: sensu-aggregate-check --api-user=foo --api-pass=bar --check-labels='aggregate=healthz,app=dummy' --warn-percent=75 --crit-percent=50
  subscriptions:
  - backend
  publish: true
  interval: 30
  handlers:
  - slack
  - pagerduty
  - email
```

### Sensu Core

N/A

## Installation from source and contributing

### Sensu Go

To build from source, from the local path of the sensu-aggregate-check repository:
```
go build
```

### Contributing

To contribute to this plugin, see [CONTRIBUTING](https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md)

[1]: https://bonsai.sensu.io/assets/sensu/sensu-aggregate-check
[2]: https://github.com/sensu/sensu-aggregate-check/releases

