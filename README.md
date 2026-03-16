# httpclient-lib-go

[![Join the #publiccode channel](https://img.shields.io/badge/Slack%20channel-%23publiccode-blue.svg?logo=slack)](https://developersitalia.slack.com/messages/CAM3F785T)
[![Get invited](https://slack.developers.italia.it/badge.svg)](https://slack.developers.italia.it/)

This is a Go common lib for [crawler](https://github.com/italia/developers-italia-backend), [publiccode-parser-go](https://github.com/italia/publiccode-parser-go) and [publiccode-validator](https://github.com/italia/publiccode-validator)

publiccode.yml is an international standard for describing public software. It is expected to be published in the root of open source repositories. This parser performs syntactic and semantic validation according to the official spec.

## Usage

### Package-level functions

Quickest way to get started:

```go
resp, err := httpclient.GetURL("https://example.com", nil)
resp, err := httpclient.PostURL("https://example.com", headers, body)
resp, err := httpclient.Request("https://example.com", "PUT", headers, body)
```

### Custom HTTP client

If you need more control (eg. custom transport, shared client, test doubles) you can instantiate a `Client` directly via `NewClient`:

```go
httpClient := &http.Client{
    Timeout:   30 * time.Second,
    Transport: myCustomTransport,
}
client := httpclient.NewClient(httpClient)

resp, err := client.GetURL("https://example.com", nil)
resp, err := client.PostURL("https://example.com", headers, body)
resp, err := client.Request("https://example.com", "PUT", headers, body)
```

Anything implementing the `HTTPDoer` interface (`Do(*http.Request) (*http.Response, error)`) works.

## Contributing

Contributing is always appreciated.
Feel free to open issues, fork or submit a Pull Request.
If you want to know more about how to add new fields, check out [CONTRIBUTING.md](CONTRIBUTING.md). In order to support other country-specific extensions in addition to Italy some refactoring might be needed.

## See also

* [Developers Italia backend & crawler](https://github.com/italia/developers-italia-backend) - a Go crawler for PC.
* [publiccode-parser-go](https://github.com/italia/publiccode-parser-go) - a Go parser and validator for publiccode.yml files.
* [publiccode-validator](https://github.com/italia/publiccode-validator) - a REST API server for publiccode validation.

## Maintainers

This software is maintained by the [Developers Italia](https://developers.italia.it/) team.

## License

© 2020 Team per la Trasformazione Digitale - Presidenza del Consiglio dei Minstri

Licensed under the AGPL-3.0.
The version control system provides attribution for specific lines of code.
