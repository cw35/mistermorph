# Demos: Embedding `mistermorph`

This folder contains two demos showing how to embed `mistermorph` into another project.

## 1) Embed as a Go library

See `demo/embed-go/`.

It now includes three runtime modes in one program:
- `task`: one-shot in-process task run
- `telegram`: start Telegram bot via integration API
- `slack`: start Slack Socket Mode bot via integration API

## 2) Embed as a CLI subprocess

See `demo/embed-cli/`.
