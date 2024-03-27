provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_reservedfixedip" "fixed_ip" {
  project_id                    = 1
  region_id                     = 1
  type                          = "external"
  is_vip                        = true
  instance_ports_that_share_vip = ["8296f985-eb1e-4ac8-8a99-cd1156746d30", "41233b81-f42r-46d0-8043-759c8c542219"]
}