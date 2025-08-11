Terraform EdgeCenter Provider
------------------------------

<img alt="EdgeCenter" src="https://edgecenter.ru/files/marketing/logo_EC_black.svg" width="500px"/>
====================================================================================

- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)

Latest provider
------------
- [edge-center provider](https://registry.terraform.io/providers/Edge-Center/edgecenter/latest)


Using the provider
------------------
### Requirements


-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13

### Configuring terraform

#### Create Terraform configuration file
```shell
$ touch ~/.terraformrc
```
For using terraform without VPN, you should configure mirror by
[manual](https://edgecenter.ru/knowledge-base/storage/terraform) or
configure mirror in `~/.terraformrc` file:

```terraform
provider_installation {

   network_mirror {
      url = "https://terraform-mirror.yandexcloud.net/"
      include = ["registry.terraform.io/*/*"]
   }
   # For all other providers, install them directly from their origin provider 
   # registries as normal. If you omit this, Terraform will _only_ use 
   # the dev_overrides block, and so no other providers will be available. 
   direct {
      exclude = ["registry.terraform.io/*/*"]
   }
}
```
### Project initializing

#### Prepare module configuration file
It is necessary to add provider settings according to the instructions https://developer.hashicorp.com/terraform/language/providers/requirements:
Each Terraform module must declare which providers it requires, so that Terraform can install and use them.

To use the provider, prepare configuration file `provider.tf` in the module directory.

```terraform
terraform {
  required_version = ">= 0.13.0"

  required_providers {
    edgecenter = {
       source = "Edge-Center/edgecenter"
       version = "{version_number}"  # need to specify (choose from https://github.com/Edge-Center/terraform-provider-edgecenter/releases)
    }
  }
}


provider edgecenter {
  edgecenter_platform_api = "https://api.edgecenter.ru/iam"
  edgecenter_cloud_api = "https://api.edgecenter.ru/cloud"
  permanent_api_token = "{your_permanent_token}" # need to specify (you can create it on the page https://accounts.edgecenter.ru/profile/api-tokens)
}
```
#### Initialize working directory
Run terraform init in the module directory.
```bash
terraform init
```
This command initializes a working directory containing Terraform configuration files. This is the first command that 
should be run after writing a new Terraform configuration or cloning an existing one from version control. It is safe to run this command multiple times

### Writing modules files

Create module files using the examples provided in the `./examples` folder.


Development
-----------
### Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13
-   [vault](https://developer.hashicorp.com/vault/install)
-	[Go](https://golang.org/doc/install) 1.23 (to build the provider plugin)



### Initialization of the project

#### Setting Vault envs
Before initializing the project you should to set VAULT_ADDR and VAULT_TOKEN envs:
```bash
export VAULT_ADDR=<you can ask address from Cloud team (matermost tag: @devcloud)>
export VAULT_TOKEN=<you can generate this token at the ${VAULT_ADDR} page>
```
#### Initialize
Initialize the project using this command:
```sh
$ make init
```

### Building the provider

```sh
$ make build
```

### Using built Terraform provider

To use the built provider from the previous step, you should add the dev_overrides option to the `~/.terraformrc` file:

```terraform
provider_installation { 
 
  dev_overrides { 
      "local.edgecenter.ru/repo/edgecenter" = "/<dev-path>/terraform-provider-edgecenter/bin" 
  } 
  
  network_mirror {
    url = "https://terraform-mirror.yandexcloud.net/"
    include = ["registry.terraform.io/*/*"]
  }
 
  # For all other providers, install them directly from their origin provider 
  # registries as normal. If you omit this, Terraform will _only_ use 
  # the dev_overrides block, and so no other providers will be available. 
    direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
```
Then you should use `"local.edgecenter.ru/repo/edgecenter"` source in required_provider settings in the `provider.tf` file:

```terraform
terraform {
  required_version = ">= 0.13.0"

  required_providers {
    edgecenter = {
       source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}


provider edgecenter {
  edgecenter_platform_api = "https://api.edgecenter.ru/iam"
  edgecenter_cloud_api = "https://api.edgecenter.ru/cloud"
  permanent_api_token = "{your_permanent_token}" # need to specify (you can create it on the page https://accounts.edgecenter.ru/profile/api-tokens)
}
```

Testing
------------------
### Remote 
Tests are run with provided secrets envs in the GitHub repository.
### Local 
For testing cloud data sources
```bash
make test_cloud_data_source
```
For testing cloud resources
```bash
make test_cloud_resource
```
For testing not cloud (storage, cdn, dns) data sources and resources:
```bash
make test_not_cloud
```

Lint
----------
To lint project code, use:
```bash
make linters
```

Docs generating
------------------
To generate Terraform documentation, use the command `make docs`. This command uses the `terraform-plugin-docs` library to create provider documentation with examples and places it in the `docs` folder. These docs can be viewed on the provider registry page.


Debugging
------------------
There are two ways to debug the provider:
### Goland debugging
1. Add new go build configuration with field ```program arguments```:  
```-debug -address=local.edgecenter.ru/repo/edgecenter```
2. Run this configuration;
3. Export generated TF_REATTACH_PROVIDERS env from debug output:
```bash
export TF_REATTACH_PROVIDERS='{"local.edgecenter.ru/repo/edgecenter":{"Protocol":"grpc","ProtocolVersion":5,"Pid":65114,"Test":true,"Addr":{"Network":"unix","String":"/var/folders/g4/q8cpkx6n7gg_1cvr0lrtkd4w0000gq/T/plugin1422839680"}}}'
```
4. Set a breakpoint in your code and apply the Terraform config:
```bash
terraform apply
```
5. Debugging.

### VSCode debugging
1. Create a `launch.json` file:
   * In the Run view, click `create a launch.json file`.
   * Choose Go: Launch Package from the debug configuration drop-down menu. 
   * VS Code will create a `launch.json` file in a `.vscode` folder in your workspace.
2. Add a new configuration to `launch.json`:
   * The `address` argument must be equal to the `source` field from your `provider.tf`.
   ```json 
   {
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Terraform Provider",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "env": {},
            "args": [
                "-debug",
                "-address=local.edgecenter.ru/repo/edgecenter"
            ]
        }
    ]
   } 
   ```
3. Launch the debug mode: `Run > Start Debugging (F5)`.
4. Copy the `TF_REATTACH_PROVIDERS` env from the console and export it to the terminal as follows:
    ```shell
    export TF_REATTACH_PROVIDERS='{"local.edgecenter.ru/repo/edgecenter":{...'
    ```
5. Set a breakpoint in your code and apply the Terraform config: `terraform apply`.
6. Debugging.

Thank You!
