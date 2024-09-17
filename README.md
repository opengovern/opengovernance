# OpenGovernance

OpenGovernance streamlines governance across multi-cloud and multi-platform environments by centralizing discovery, policy enforcement, compliance checking, and change tracking. Your infrastructure, regardless of where it resides, remains secure, optimized, and compliant.

Think of it as a Customer Data Platform for your entire tech stack. It provides a unified view—cloud providers, Kubernetes clusters, code repositories, security tools—allowing for effective governance from a single platform.

![enter image description here](https://docs.opengovernance.io/~gitbook/image?url=https://content.gitbook.com/content/flsJtdaedb8TrA13g8H6/blobs/flOcrYsPU0eBrF73O6tN/Screenshot%2520by%2520Dropbox%2520Capture.png&width=768&dpr=4&quality=100&sign=424ffc86&sv=1)

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
