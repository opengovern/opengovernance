# OpenGovernance

## Introduction

OpenGovernance centralizes control of your entire tech stack with a unified inventory, enforcing compliance, streamlining operations, and driving faster, secure deployments.
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

Before generating docs, you need to install [`swag`](https://github.com/swaggo/echo-swagger#start-using-it) CLI tool.
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
