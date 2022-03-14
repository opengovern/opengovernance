# Keibi Engine
=======

Introduction
---

This is a CLI to query cloud resources such EC2 instances from public cloud. For now only AWS is supported.

Build
---

Run the build command

    make build

The built binary is found in the ./build directory

Executing
--

Start using the CLI by running command

    ./build/cloud-inventory aws --help

# Generating Swagger UI
=========

Before generating docs, you need to [install]((https://github.com/swaggo/echo-swagger#start-using-it)) `swag` CLI tool.

To generate the Swagger documentation, run:

```sh
./scripts/generate_doc.sh
```

You can find ways to populate Swagger UI in [this](https://github.com/swaggo/swag#general-api-info) link.