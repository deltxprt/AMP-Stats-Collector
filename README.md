[![goreleaser](https://github.com/deltxprt/AMP-Stats-Collector/actions/workflows/build%20application.yml/badge.svg)](https://github.com/deltxprt/AMP-Stats-Collector/actions/workflows/build%20application.yml)

# Introduction

Simple runner to send stats to an influxdb endpoint

# Configuration File

```yaml
url: "https://my.endpoint.com"
username: username
password: "superSecretPassword"
influxAddr: "https://influx.app:8086"
org: superOrg
bucket: AMP-bucket
token: "superlongtoken1234"
interval: 30s # ex: 10s, 5m, 1h
jaegerURL: "http://jaeger.app:14268/api/traces"
```

# docker

```yaml
docker run 
  -v /etc/stats-mon/config.yaml:/config/config.yaml 
  ghcr.io/deltxprt/AMP-Stats-Collector
```