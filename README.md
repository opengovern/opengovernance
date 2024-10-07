# OpenGovernance

![App Screenshot](https://raw.githubusercontent.com/kaytu-io/open-governance/b714c9bce4bd59e8bc4305007f88d856aeb360fe/screenshots/app%20-%20screenshot%202.png)

OpenGovernance is a platform designed to help streamline compliance, security, and operations across your cloud and on-premises environments. Built with developers in mind, it manages policies in Git, supports easy parameterization, and allows for straightforward customization to meet your specific requirements.

Unlike traditional governance tools that can be complex to set up and maintain, OpenGovernance is user-friendly and easy to operate. You can have your governance framework up and running in under two minutes without dealing with intricate configurations.

OpenGovernance can replace legacy compliance systems by providing a unified interface, reducing the need for multiple separate installations. It supports managing standards like SOC2 and HIPAA, ensuring your organization stays compliant with less effort.

By optimizing your compliance and governance processes, OpenGovernance helps reduce operational costs. Below, we share results from implementing OpenGovernance in our production environments, highlighting improved efficiency and lower operational overhead compared to older governance solutions.

Explore how OpenGovernance can help manage your compliance, security, and operations more effectively.




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
