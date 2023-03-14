Terraform EdgeCenter Provider
------------------------------

<img src="https://edgecenter.ru/img/logo.svg" data-src="https://edgecenter.ru/img/logo.svg" alt="EdgeCenter" width="500px" width="500px"> 
====================================================================================

- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.13.x
-	[Go](https://golang.org/doc/install) 1.19 (to build the provider plugin)

Latest provider
------------
- [edge-center provider](https://registry.terraform.io/providers/Edge-Center/edgecenter/latest)

Building the provider
---------------------
```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers
$ cd $GOPATH/src/github.com/terraform-providers
$ git clone https://github.com/Edge-Center/terraform-provider-edgecenter.git
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-edgecenter
$ make build
```

### Override Terraform provider

To override terraform provider for development goals you do next steps: 

create Terraform configuration file
```shell
$ touch ~/.terraformrc
```

point provider to development path
```shell
provider_installation { 
 
  dev_overrides { 
      "local.edgecenter.ru/repo/edgecenter" = "/<dev-path>/terraform-provider-edgecenter/bin" 
  } 
 
  # For all other providers, install them directly from their origin provider 
  # registries as normal. If you omit this, Terraform will _only_ use 
  # the dev_overrides block, and so no other providers will be available. 
  direct {} 
}
```

add `local.edgecenter.ru/repo/edgecenter` to .tf configuration file
```shell
terraform {
  required_version = ">= 0.13.0"

  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
      version = "{version_number}"  # need to specify
    }
  }
}
```

Using the provider
------------------
To use the provider, prepare configuration files based on examples

```sh
$ cp ./examples/... .
$ terraform init
```

Testing
------------------
Remote: Tests are run with provided secrets envs in the GitHub repository.
Local: execute the command `make test_local_data_source` and `make test_local_resource`. For this command to work, you need to:
* Create a `.local.env` file and fill it with the necessary envs. 
* Run `make envs` to automatically fill the envs from Vault (don't forget to export `VAULT_TOKEN` to terminal).
* `make envs` requires the installation of `jq` and the `vault` binary. You can install them with the `make vault` and `make jq` commands, respectively.

Docs generating
------------------
To generate Terraform documentation, use the command `make docs`. This command uses the `terraform-plugin-docs` library to create provider documentation with examples and places it in the `docs` folder. These docs can be viewed on the provider registry page.

Debugging
------------------
There are two ways to debug the provider:
### VSCode debugging
1. Create a `launch.json` file:
   * In the Run view, click `create a launch.json file`.
   * Choose Go: Launch Package from the debug configuration drop-down menu. 
   * VS Code will create a `launch.json` file in a `.vscode` folder in your workspace.
2. Add a new configuration to `launch.json`:
   * The `address` argument must be equal to the `source` field from your `provider.tf`.
   ``` {
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

### using delve
1. Install the Delve library - [installation](https://github.com/go-delve/delve/tree/master/Documentation/installation)
2. Build binary without optimization or use `make build_debug` 
    ```shell
    go build -o bin/$(BINARY_NAME) -gcflags '-N -l'
    ```
3. Open the first terminal:
   * Run the binary with the debug option:
   ```shell
   dlv exec bin/terraform-provider-edgecenter -- -debug
   ```
   * Set a breakpoint for the create function with a resource that you want to debug, e.g,
      ```shell
      break resourceFloatingIPCreate
      ```
   * `continue`
   * Copy `TF_REATTACH_PROVIDERS` with its value from output
4. Open the second terminal:
   * Export `TF_REATTACH_PROVIDERS`:
     ```shell
     export TF_REATTACH_PROVIDERS='{"local.edgecenter.ru/repo/edgecenter":{...'
     ```
   * Launch ```terraform apply```
   * Debug with the `continue` command in the first terminal via `delve`

Thank You
