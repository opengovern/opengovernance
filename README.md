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

# Private Golang Repository Permissions
========

If you try to build or download the dependencies, you will need permissions to internal repositories. To fix that, add the following log to your git configuration. This will instruct git to use your SSH key to download the private repositories instead of HTTPS.


```
[url "ssh://git@gitlab.com/keibiengine/"]
	insteadOf = https://gitlab.com/keibiengine/
```

Or run the following command

```
git config --global url."ssh://git@gitlab.com/keibiengine/".insteadOf https://gitlab.com/keibiengine/
```
