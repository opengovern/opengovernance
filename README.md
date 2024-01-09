# Kaytu Engine

## Introduction

The main mono repository for Kaytu microservices. Microservices are written into the `/services` folder
(Please note that migration is in progress which means some of the microservice are still in `/pkg`).

The main job queuing system is NATS (again migration is in progress from Kafka and RabbitMQ).
Jobs usually defined in the scheduler and then handle by their related services. The result
is put back for the schedular.

## Build

Run the build command

```bash
make build
```

The built binary is found in the ./build directory

## Executing

Start using the CLI by running command

```bash
./build/cloud-inventory aws --help
```

## Generating Swagger UI

Before generating docs, you need to install [`sway`](https://github.com/swaggo/echo-swagger#start-using-it) CLI tool.
To generate the Swagger documentation, run:

```bash
./scripts/generate_doc.sh
```

You can find ways to populate Swagger UI in [this](https://github.com/swaggo/swag#general-api-info) link.

## Private Golang Repository Permissions

If you try to build or download the dependencies, you will need permissions to internal repositories. To fix that, add the following log to your git configuration. This will instruct git to use your SSH key to download the private repositories instead of HTTPS.

```ssh
[url "ssh://git@github.com/kaytu-io/"]
    insteadOf = https://github.com/kaytu-io/
```

Or run the following command:

```bash
git config --global url."ssh://git@github.com/kaytu-io/".insteadOf https://github.com/kaytu-io/
```
