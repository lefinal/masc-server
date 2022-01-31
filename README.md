# MASC Server

[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
![Go](https://github.com/LeFinal/masc-server/workflows/Go/badge.svg?branch=master)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/lefinal/masc-server)
[![GoReportCard example](https://goreportcard.com/badge/github.com/lefinal/masc-server)](https://goreportcard.com/report/github.com/lefinal/masc-server)
[![GitHub issues](https://img.shields.io/github/issues/lefinal/masc-server)](https://github.com/lefinal/masc-server/issues)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/lefinal/masc-server)

Backend for Mission Airsoft Control

## Configuration

Example config:
```json
{
  "log": {
    "stdout_log_level": "debug",
    "high_priority_output": "/tmp/high-priority-output.log",
    "debug_output": "/tmp/debug-output.log",
    "max_size": 8,
    "keep_days": 180
  },
  "db_conn": "postgres://masc:masc@localhost:5432/masc",
  "websocket_addr": ":8080",
  "mqtt_addr": "tcp://localhost:1883"
}
```