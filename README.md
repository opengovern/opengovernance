<p align="center">
  <a href="https://opengovernance.io">
    <img src="https://github.com/kaytu-io/website/blob/34af0c464c3a75b1382b63ae4d0f8024f008c858/connectors/icons/open-governance.svg" alt="OpenGovernance">
  </a>
</p>

<p align="center"> <em>🚀 Full Stack Governance - govern across clouds, platforms 🚀 Steampipe Compatible, 🚀 high-performance security controls, 🚀 scalable across clouds and platforms, 🚀 compliance, security, operations under one roof.</em> </p>

OpenGovernance streamline governance, compliance, security, and operations across your cloud and on-premises environments. Built with developers in mind, it manages policies in Git, supports easy parameterization, and allows for straightforward customization to meet your specific requirements.

![App Screenshot](https://raw.githubusercontent.com/kaytu-io/open-governance/b714c9bce4bd59e8bc4305007f88d856aeb360fe/screenshots/app%20-%20screenshot%201.png)

Unlike traditional governance tools that can be complex to set up and maintain, OpenGovernance is user-friendly and easy to operate. You can have your governance framework up and running in a few minutes minutes without dealing with intricate configurations.

OpenGovernance can replace legacy compliance systems by providing a unified interface, reducing the need for multiple separate installations. It supports managing standards like SOC2 and HIPAA, ensuring your organization stays compliant with less effort.

By optimizing your compliance and governance processes, OpenGovernance helps reduce operational costs.

## 🌟 Features:
- **Centralized Multi-Cloud Governance**: Manage AWS, Azure, and GCP policies from one platform.
- **Steampipe Compatibility**: Leverage existing queries and data sources with ease.
- **Track History & Capture Evidence**: Keep an audit trail and ensure regulatory compliance.
- **Automated Compliance & Security**: Built-in policies for SOC 2, HIPAA, and more across multiple clouds.
- **Customizable Policy Controls**: Use simple SQL to define and enforce your standards.
- **Vendor-Neutral & Open Source**: Flexible integration with existing tools and platforms.
- **Role-Based Access Control (RBAC)**: Secure, fine-grained access management.
- **User-Friendly Interface**: Intuitive, multilingual UI
- **Dynamic Reporting & Dashboards**: Customizable dashboards and reports for governance insights.

## ⚡️ Quick start on Kubernetes:

### Add the Helm Repository:

```bash
helm repo add opengovernance https://kaytu-io.github.io/kaytu-charts && helm repo update
```

### Run Helm Install
```bash
helm install -n opengovernance opengovernance opengovernance/open-governance --create-namespace --timeout=10m
```

### Expose the app

```bash
kubectl port-forward -n opengovernance svc/nginx-proxy 8080:80
```
Navigate to http://localhost:8080/ in your browser.
To sign in, use admin@example.com as the username and password as the password.
