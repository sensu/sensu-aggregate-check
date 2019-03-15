# Sensu Go Aggregate Check Plugin
TravisCI: [![TravisCI Build Status](https://travis-ci.org/sensu/sensu-aggregate-check.svg?branch=master)](https://travis-ci.org/sensu/sensu-aggregate-check)

TODO: Description.

## Installation

Download the latest version of the sensu-aggregate-check from [releases][1],
or create an executable script from this source.

From the local path of the sensu-aggregate-check repository:

```
go build -o /usr/local/bin/sensu-aggregate-check main.go
```

## Configuration

Example Sensu Go definition:

```json
{
    "api_version": "core/v2",
    "type": "CheckConfig",
    "metadata": {
        "namespace": "default",
        "name": "sensu-aggregate-check"
    },
    "spec": {
        "...": "..."
    }
}
```

## Usage Examples

Help:

```
The Sensu Go Event Aggregates Check plugin

Usage:
  sensu-aggregate-check [flags]

Flags:
  -h, --help                help for sensu-aggregate-check
  -l, --labels string       aggregate=foo,app=bar
  -n, --namespaces string   us-east-1,us-west-2 (default "default")
```

## Contributing

See https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md

[1]: https://github.com/sensu/sensu-aggregate-check/releases
