# GitHub Provider KOG Blueprint

This is a Krateo Blueprint that deploys the GitHub Provider KOG leveraging the [OASGen Provider](https://github.com/krateoplatformops/oasgen-provider) and using [OpenAPI Specifications (OAS) of the GitHub REST API](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.2022-11-28.yaml).
This provider allows you to manage GitHub resources such as repositories, collaborators, etc. in a cloud-native way using the Krateo platform.

## Summary

- [Requirements](#requirements)
- [Project structure](#project-structure)
- [How to install](#how-to-install)
  - [Full provider installation](#full-provider-installation)
  - [Single resource installation](#single-resource-installation)
- [Latest versions](#latest-versions)
- [Supported resources](#supported-resources)
  - [Resource details](#resource-details)
    - [BranchProtection](#branchprotection)
    - [Repo](#repo)
    - [Collaborator](#collaborator)
    - [TeamRepo](#teamrepo)
    - [Workflow](#workflow)
    - [RunnerGroup](#runnergroup)
  - [Resource examples](#resource-examples)
- [Authentication](#authentication)
- [GitHub Enterprise Server Support](#github-enterprise-server-support)
- [Configuration](#configuration)
  - [Configuration resources](#configuration-resources)
  - [values.yaml](#valuesyaml)
  - [Verbose logging](#verbose-logging)
- [Chart structure](#chart-structure)
- [Troubleshooting](#troubleshooting)

## Requirements

[OASGen Provider](https://github.com/krateoplatformops/oasgen-provider) should be installed in your cluster. Follow the related Helm Chart [README](https://github.com/krateoplatformops/oasgen-provider-chart) for installation instructions.
Note that a standard installation of Krateo contains the OASGen Provider.

Minimum `oasgen-provider-chart` version required: `0.9.0` (which is shipped with Krateo 2.7.0).

## Project structure

This project is composed by the following folders:
- **github-provider-kog-*-blueprint**: Helm charts that deploys single resources supported by this provider. These charts are useful if you want to deploy only one of the supported resources.
- **github-provider-kog-blueprint**: a Helm chart that can deploy all resources supported by this provider. It is useful if you want to manage multiple of the supported resources.
- **plugins**: a folder that is a monorepo containing multiple Go plugins. The plugins are used to resolve some inconsistencies of the GitHub REST API. If needed, they are deployed as part of the Helm chart of the specific resource.

## How to install

### Full provider installation

To install the **github-provider-kog-blueprint** Helm chart (full provider), use the following command:

```sh
helm install github-provider-kog github-provider-kog \
  --repo https://marketplace.krateo.io \
  --namespace <release-namespace> \
  --create-namespace \
  --version 1.2.0 \
  --wait
```

> [!NOTE]
> Due to the nature of the providers leveraging the [OASGen Provider](https://github.com/krateoplatformops/oasgen-provider), this chart will install a set of RestDefinitions that will in turn trigger the deployment of a set controllers in the cluster. These controllers need to be up and running before you can create or manage resources using the Custom Resources (CRs) defined by this provider. This may take a few minutes after the chart is installed. The RestDefinitions will reach the condition `Ready` when the related CRDs are installed and the controllers are up and running.

You can check the status of the RestDefinitions with the following commands:

```sh
kubectl get restdefinitions.ogen.krateo.io --all-namespaces | awk 'NR==1 || /github/'
```
You should see output similar to this:
```sh
NAMESPACE       NAME                               READY   AGE
krateo-system   github-provider-kog-collaborator   False   59s
krateo-system   github-provider-kog-repo           False   59s
krateo-system   github-provider-kog-runnergroup    False   59s
krateo-system   github-provider-kog-teamrepo       False   59s
krateo-system   github-provider-kog-workflow       False   59s
```

You can also wait for a specific RestDefinition (`github-provider-kog-repo` in this case) to be ready with a command like this:
```sh
kubectl wait restdefinitions.ogen.krateo.io github-provider-kog-repo --for condition=Ready=True --namespace krateo-system --timeout=300s
```

Note that the names of the RestDefinitions and the namespace where the RestDefinitions are installed may vary based on your configuration.

### Single resource installation

To manage a single resource, you can install the specific Helm chart for that resource. For example, to install the `github-provider-kog-repo` resource, you can use the following command:

```sh
helm install github-provider-kog-repo github-provider-kog-repo \
  --repo https://marketplace.krateo.io \
  --namespace <release-namespace> \
  --create-namespace \
  --version 1.1.0 \
  --wait
```

> [!NOTE]
> Blueprint version may vary based on the resource you want to install.

## Latest versions

The following table shows the latest versions of the charts for each resource supported by this provider and the main chart that can deploy all resources:

| Main chart      | Version |
|-----------------|---------|
| github-provider-kog-blueprint | 1.2.0   |

| Resource        | Version |
|-----------------|---------|
| github-provider-kog-branchprotection | 1.1.0   |
| github-provider-kog-collaborator | 1.1.0   |
| github-provider-kog-repo | 1.0.0   |
| github-provider-kog-teamrepo | 1.1.0   |
| github-provider-kog-workflow | 1.0.0   |
| github-provider-kog-runnergroup | 1.0.0   |

## Supported resources

This chart supports the following resources and operations:

| Resource        | Get  | Create | Update | Delete |
|-----------------|------|--------|--------|--------|
| BranchProtection | ✅   | ✅     | ✅     | ✅     |
| Collaborator    | ✅   | ✅     | ✅     | ✅     |
| Repo            | ✅   | ✅     | ✅     | ✅     |
| TeamRepo        | ✅   | ✅     | ✅     | ✅     |
| Workflow        | 🚫 Not applicable   | ✅     | 🚫 Not applicable    | 🚫 Not applicable     |
| RunnerGroup     | ✅   | ✅     | ✅     | ✅     |

> [!NOTE]  
> 🚫 *"Not applicable"* indicates that the operation is not supported by this provider because it probably does not make sense for the resource type.  For example, GitHub Workflow runs are typically not updated or deleted directly; they are triggered and if a new run is needed, a new workflow run is created.

The resources listed above are Custom Resources (CRs) defined in the `github.ogen.krateo.io` API group. They are used to manage GitHub resources in a Kubernetes-native way, allowing you to create, update, and delete GitHub resources using Kubernetes manifests.

### Resource details

#### BranchProtection

The `BranchProtection` resource allows you to manage branch protection rules for GitHub repositories.
You can enforce policies such as requiring pull request reviews, status checks, linear history, and restricting who can push to a branch.

This resource uses a plugin (a proxy service deployed by the chart) to normalize the GitHub API response, since GitHub returns boolean fields (e.g., `enforce_admins`, `allow_force_pushes`) wrapped in objects like `{ "enabled": true }` rather than as plain booleans. The plugin unwraps these transparently so the controller can perform reconciliation without misleading drifts detected due to differences in the API response and the desired state defined in the CRs.

An example of a BranchProtection resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: BranchProtection
metadata:
  name: test-branchprotection-full
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-branchprotection-config
    namespace: default
  owner: krateoplatformops-test
  repo: branchprotection-tester-02
  branch: main
  allow_deletions: false
  allow_force_pushes: false
  allow_fork_syncing: false
  block_creations: true
  enforce_admins: true
  lock_branch: false
  required_conversation_resolution: true
  required_linear_history: true
  required_status_checks:
    strict: true
    checks:
      - context: "ci/build"
        app_id: -1
      - context: "ci/lint"
        app_id: -1
      - context: "security-scan"
        app_id: -1
      - context: "unit-tests"
        app_id: -1
  required_pull_request_reviews:
    required_approving_review_count: 2
    dismiss_stale_reviews: true
    require_code_owner_reviews: true
    require_last_push_approval: true
    dismissal_restrictions:
      users:
        - vicentinileonardo
      teams: []
      apps: []
    bypass_pull_request_allowances:
      users:
        - vicentinileonardo
      teams: [] 
      apps: []
  restrictions:
    users:
      - vicentinileonardo
    teams: []
    apps: []
```

Other examples of BranchProtection resources with different settings can be found in the [/samples/branchprotection](./github-provider-kog-blueprint/samples/branchprotection/) folder of the main chart.

You can find more details about the `BranchProtection` resource in the [Branch Protection documentation](https://docs.github.com/en/rest/branches/branch-protection?apiVersion=2022-11-28) of the GitHub REST API.

> [!NOTE]  
> The field `required_status_checks.contexts` which is available in the GitHub REST API but marked as deprecated is NOT supported by this provider in favor of the `required_status_checks.checks` field, which allows you to specify the status checks with more details (e.g., `app_id` of the check).

#### Repo

The `Repo` resource allows you to create, update, and delete GitHub repositories. 
You can specify the repository name, description, visibility (public or private), and other settings that can be seen in the [GitHub REST API documentation](https://docs.github.com/en/rest/repos?apiVersion=2022-11-28) and the selected OpenAPI Specification in the `/assets` folder of this chart.

An example of a Repo resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: Repo
metadata:
  name: test-repo
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-repo-config
    namespace: default 
  org: krateoplatformops-test
  name: test-repo
  description: A short description of the repository set by Krateo
  visibility: public
  has_issues: true
```

#### Collaborator 

The `Collaborator` resource allows you to add and remove collaborators from a GitHub repository. 
You can specify the username of the collaborator and the permission level among `admin`, `pull`, `push`, `maintain`, and `triage`.
Using any other value will result in an error or continuous reconciliation loops.
Updating a collaborator's permission level is also supported.

In addition, this resource supports adding "external collaborators" to a repository, meaning users who are not members of the organization that owns the repository.
In this case, an invitation will be sent to the user with the specified permission level.
Updating and deleting invitations is supported through the same `Collaborator` resource.
You can verify whether the user is directly added as a collaborator or if the invitation is pending by checking the `message` field in the Collaborator resource status.
Note that the `Collaborator` resource will remain in a `Pending` state until the user accepts the invitation.

An example of a `Collaborator` resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: Collaborator
metadata:
  name: add-collaborator
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-collaborator-config
    namespace: default 
  owner: krateoplatformops-test
  repo: collaborator-tester
  username: vicentinileonardo
  permission: pull
```

#### TeamRepo

The `TeamRepo` resource allows you to manage team access to GitHub repositories. 
You can specify the `team_slug`, repository name, and permission level among `admin`, `pull`, `push`, `maintain`, and `triage`.
Using any other value will result in an error or continuous reconciliation loops.
Updating a collaborator's permission level is also supported.

An example of a TeamRepo resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: TeamRepo
metadata:
  name: test-teamrepo
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-teamrepo-config
    namespace: default 
  org: krateoplatformops-test
  owner: krateoplatformops-test
  team_slug: testteam
  repo: teamrepo-tester
  permission: pull
```

#### Workflow

The `Workflow` resource allows you to trigger GitHub Actions workflow runs (`workflow_dispatch`). 
You can specify the repository name, workflow file name, and any input parameters required by the workflow. 
You must configure your GitHub Actions workflow to run when the [`workflow_dispatch` webhook](https://docs.github.com/en/webhooks/webhook-events-and-payloads#workflow_dispatch) event occurs. 
The `inputs` must configured in the workflow file.
Please refer to the [GitHub REST API documentation](https://docs.github.com/en/rest/actions/workflows?apiVersion=2022-11-28#create-a-workflow-dispatch-event) for more information.

An example of a Workflow resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-tester
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-workflow-config
    namespace: default 
  owner: krateoplatformops-test
  repo: workflow-tester
  workflow_id: test.yaml
  ref: main
  inputs:
    environment: development
    version: "v1.2.3"
    debug_enabled: "false"
    custom_message: "Test 04/06 at 13:42 from Krateo"
```

#### RunnerGroup

The `RunnerGroup` resource allows you to manage GitHub runner groups. You can specify the runner group name, and any additional settings required by the runner group such as `visibility` and `allows_public_repositories`.

An example of a RunnerGroup resource is:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: RunnerGroup
metadata:
  name: runnergroup-test
  namespace: default
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: my-runnergroup-config
    namespace: default
  name: runner-test-by-krateo
  org: krateoplatformops-test
  allows_public_repositories: false
```

### Resource examples

You can find example resources for each supported resource type in the `/samples` folder of the chart.

## Authentication

The authentication to the GitHub REST API is managed using 2 kinds of resources (both are required):

- **Kubernetes Secret**: This resource is used to store the GitHub Personal Access Token (PAT) that is used to authenticate with the GitHub REST API. The PAT should have the necessary permissions to manage the resources you want to create or update.

In order to generate a GitHub token, follow this instructions: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-personal-access-token-classic

Example of a Kubernetes Secret that you can apply to your cluster:
```sh
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: gh-token
  namespace: default
type: Opaque
stringData:
  token: <PAT>
EOF
```

Replace `<PAT>` with your actual GitHub Personal Access Token.

- **\<Resource\>Configuration**: These resource can reference the Kubernetes Secret and are used to authenticate with the GitHub REST API. They must be referenced with the `configurationRef` field of the resources defined in this chart. The configuration resource can be in a different namespace than the resource itself.

Note that the specific configuration resource type depends on the resource you are managing. For instance, in the case of the `Repo` resource, you would need a `RepoConfiguration`.

An example of a `RepoConfiguration` resource that references the Kubernetes Secret, to be applied to your cluster:
```sh
kubectl apply -f - <<EOF
apiVersion: github.ogen.krateo.io/v1alpha1
kind: RepoConfiguration
metadata:
  name: my-repo-config
  namespace: default
spec:
  authentication:
    bearer:
      # Reference to a secret containing the bearer token
      tokenRef:
        name: gh-token        # Name of the secret
        namespace: default    # Namespace where the secret exists
        key: token            # Key within the secret that contains the token
EOF
```

Then, in the `Repo` resource, you can reference the `RepoConfiguration` resource as follows:
```yaml
apiVersion: github.ogen.krateo.io/v1alpha1
kind: Repo
metadata:
  name: test-repo
spec:
  configurationRef:
    name: my-repo-config
    namespace: default
  org: krateoplatformops-test
  name: test-repo
```

More details about the configuration resources in the [Configuration resources](#configuration-resources) section below.

## GitHub Enterprise Server Support

All resources support GitHub Enterprise Server (GHE) in addition to github.com. The GitHub API base URL is configurable per sub-chart via the `githubApiBaseUrl` value.
By default, the API base URL is set to `https://api.github.com`, which means that if you don't specify a different URL, the provider will manage resources in github.com.

The URL must be a full base URL with no trailing slash (e.g., `https://ghe.corp.com/api/v3`).

### Full provider installation (umbrella chart)

Pass `githubApiBaseUrl` for each sub-chart you want to point at GHE. Because Helm propagates sub-chart values by nesting them under the sub-chart key, you can use a values file:

```yaml
# ghe-values.yaml
github-provider-kog-branchprotection-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
github-provider-kog-collaborator-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
github-provider-kog-repo-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
github-provider-kog-runnergroup-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
github-provider-kog-teamrepo-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
github-provider-kog-workflow-blueprint:
  githubApiBaseUrl: "https://ghe.corp.com/api/v3"
```

```sh
helm install github-provider-kog github-provider-kog \
  --repo https://marketplace.krateo.io \
  --namespace krateo-system \
  --create-namespace \
  --version 1.2.0 \
  -f ghe-values.yaml \
  --wait
```

When left empty (the default), all sub-charts fall back to `https://api.github.com`.

### Single resource installation

For a single blueprint, pass `githubApiBaseUrl` directly:

```sh
helm install github-provider-kog-repo github-provider-kog-repo \
  --repo https://marketplace.krateo.io \
  --namespace krateo-system \
  --create-namespace \
  --version 1.1.0 \
  --set githubApiBaseUrl=https://ghe.corp.com/api/v3 \
  --wait
```

### How it works

- **Plugin-backed resources** (`BranchProtection`, `Collaborator`, `TeamRepo`): `githubApiBaseUrl` is injected as the `GITHUB_API_BASE_URL` environment variable into the plugin `Deployment`. The plugin uses it for every outbound GitHub API request and for its readiness probe. In addition the `servers[0].url` field of the OAS asset that the OASGen Provider uses to generate the controller is templated with `githubApiBaseUrl`.
- **Direct-API resources** (`Repo`, `RunnerGroup`, `Workflow`): `githubApiBaseUrl` is only templated into the `servers[0].url` field of the OAS asset that the OASGen Provider uses to generate the controller. No plugin is involved.

## Configuration

### Configuration resources

Each resource type (e.g., `Repo`, `Collaborator`, `TeamRepo`) requires a specific configuration resource (e.g., `RepoConfiguration`, `CollaboratorConfiguration`, `TeamRepoConfiguration`) to be created in the cluster.
Currently, the supported configuration resources are:
- `BranchProtectionConfiguration`
- `RepoConfiguration`
- `CollaboratorConfiguration`
- `TeamRepoConfiguration`
- `WorkflowConfiguration`
- `RunnerGroupConfiguration`

These configuration resources are used to store the authentication information (i.e., reference to the Kubernetes Secret containing the GitHub PAT) and other configuration options for the resource type.
You can find examples of these configuration resources in the `/samples/configs` folder of the chart.
Note that a single configuration resource can be used by multiple resources of the same type.
For example, you can create a single `RepoConfiguration` resource and reference it in multiple `Repo` resources.
Note that some configuration resources may contain just the authentication information, while others may contain additional configuration options related to the specific resource type.

### values.yaml

You can customize the **github-provider-kog-blueprint** chart (main chart) by modifying the `values.yaml` file.
For instance, you can select which resources the provider should support in the oncoming installation.
This may be useful if you want to limit the resources managed by the provider to only those you need, reducing the overhead of managing unnecessary controllers. For instance, if you only need to manage `Repo` and `BranchProtection` resources, you can disable the other resources by setting the various `enabled` fields to `false` in the `values.yaml` file, as shown below:
```yaml
github-provider-kog-branchprotection-blueprint:
  enabled: true
github-provider-kog-collaborator-blueprint:
  enabled: false
github-provider-kog-repo-blueprint:
  enabled: true
github-provider-kog-teamrepo-blueprint:
  enabled: false
github-provider-kog-workflow-blueprint:
  enabled: false
github-provider-kog-runnergroup-blueprint:
  enabled: false
```

> [!NOTE]  
> The default configuration of the main chart enables all resources supported by the chart.

### Verbose logging

In order to enable verbose logging for the controllers, you can add the `krateo.io/connector-verbose: "true"` annotation to the metadata of the resources you want to manage, as shown in the examples above. 
This will enable verbose logging for those specific resources, which can be useful for debugging and troubleshooting as it will provide more detailed information about the operations performed by the controllers.

## Charts structure

Main components of the charts:

- **RestDefinitions**: These are the core resources needed to manage resources leveraging the OASGen Provider. In this case, they refers to the OpenAPI Specification to be used for the creation of the Custom Resources (CRs) that represent GitHub resources.
They also define the operations that can be performed on those resources. Once the chart is installed, RestDefinitions will be created and as a result, specific controllers will be deployed in the cluster to manage the resources defined with those RestDefinitions.

- **ConfigMaps**: Refer directly to the OpenAPI Specification content in the `/assets` folder.

- **/assets** folder: Contains the selected OpenAPI Specification files for the GitHub REST API.

- **Deployment** (optional): Deploys a plugin that is used as a proxy to resolve some inconsistencies of the GitHub REST API. The specific endpoins managed by the plugin are described in the [plugins README](./plugins/README.md)

- **Service** (optional): Exposes the plugin described above, allowing the resource controllers to communicate with the GitHub REST API through the plugin, only if needed.

## Troubleshooting

For troubleshooting, you can refer to the [Troubleshooting guide](./blueprint/docs/troubleshooting.md) in the `/docs` folder of the blueprint (chart). 
It contains common issues and solutions related to this chart.

## Release process

For development purposes, please refer to the [Release guide](./docs/release.md) in the `/docs` folder of the chart for detailed instructions on how to release new versions of the chart and its components.
