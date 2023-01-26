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

Debugging
------------------
There are two ways to debbugging the provider:
### vscode debugging
1. create a `launch.json` file
  - In the the Run view, click `create a launch.json file`
  - Choose Go: Launch Package from the debug configuration drop-down menu. 
  - VS Code will create a `launch.json` file in a `.vscode` folder in your workspace
2. add a new configuration to `launch.json`:
- the `address` arg must be equal to the `source` field from your `provider.tf`
```
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
3. launch the debug mode: `Run > Start Debugging (F5)`
4. copy `TF_REATTACH_PROVIDERS` env from the console and export it to the terminal as follows:
```
export TF_REATTACH_PROVIDERS='{"local.edgecenter.ru/repo/edgecenter":{...'
```
5. set a breakpoint in your code and apply the terraform config: `terraform apply`
6. debugging

### using delve
1. installing the delve lib - [installation](https://github.com/go-delve/delve/tree/master/Documentation/installation)
2. building binary without optimisation or use `make build_debug` 
```
go build -o bin/$(BINARY_NAME) -gcflags '-N -l'
```
3. open first terminal:
  - run the binary with debug option:
  ```
  dlv exec bin/terraform-provider-edgecenter -- -debug
  ```
  - set a breakpoint for the create function with a resource that want to debug, e.g,
   ```
   break resourceFloatingIPCreate
   ```
  - `continue`
  - copy `TF_REATTACH_PROVIDERS` with its value from output
5. open second terminal:
  - exporting `TF_REATTACH_PROVIDERS`:
  ```
  export TF_REATTACH_PROVIDERS='{"local.edgecenter.ru/repo/edgecenter":{...'
  ```
  - launch ```terraform apply```
  - debugging with the `continue` command in the first terminal via `delve`

Thank You
