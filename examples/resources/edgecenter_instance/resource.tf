# Example 1
resource "edgecenter_instance" "instance1" {
  region_id       = var.region_id
  project_id      = var.project_id
  name            = "test-instance"
  flavor          = "g1-standard-2-4"
  keypair_name    = "test-keypair"
  server_group_id = "00000000-0000-0000-0000-000000000000"
  security_groups = ["00000000-0000-0000-0000-000000000000"]
  user_data       = "#cloud-config\npassword: ваш пароль\nchpasswd: { expire: False }\nssh_pwauth: True"

  metadata = {
    "key1" : "value1",
    "key2" : "value2",
  }

  volume {
    name           = "system"
    type_name      = "ssd_hiiops"
    size           = 30
    source         = "image"
    image_id       = "00000000-0000-0000-0000-000000000000"
    attachment_tag = "tag"
    boot_index     = 0
    metadata = {
      "tag" : "system"
    }
  }

  interface {
    type               = "any_subnet"
    network_id         = "00000000-0000-0000-0000-000000000000"
    floating_ip_source = "new"
  }

  interface {
    type               = "any_subnet"
    network_id         = "00000000-0000-0000-0000-000000000000"
    floating_ip_source = "existing"
    floating_ip        = "00000000-0000-0000-0000-000000000000"
  }

  interface {
    type       = "subnet"
    network_id = "00000000-0000-0000-0000-000000000000"
    subnet_id  = "00000000-0000-0000-0000-000000000000"
  }
}

# Example 2
# TBD with separate resource
