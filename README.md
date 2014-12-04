# Floki - a web framework for Go

Floki is Express-like framework based on the source code of Martini and Gin frameworks. It is currently undergoing heavy development. API is not stable yet but it's already used in a few live projects serving 10K+ visitors daily.

## Features
* Jade templates with live reloading
* Modular design
* Middleware support
* Performance (uses httprouter for routing, avoids use of reflection everywhere possible)
* Live code reloading through [floki-tool](https://github.com/go-floki/floki-tool)
* Environments support
* Graceful application restart (deploy new version without interrupting clients)
* OAuth2 providers: Google, Facebook, VKontakte
* GZip middleware
* Code generator from data models
* Assets external watchers spawning

## Modules
* [Sessions](https://github.com/go-floki/sessions)
* [Auth](https://github.com/go-floki/auth)
* [Assets](https://github.com/go-floki/assets)
* [Jade](https://github.com/go-floki/jade)
* [DB](https://github.com/go-floki/db)
