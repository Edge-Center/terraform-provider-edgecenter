provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_websec_source_group" "example_com" {
  name        = "example.com origins"
  description = "Origins of example.com"

  source {
    source = "203.0.113.10"
    weight = 60
  }

  source {
    source       = "backup.example.com:8080"
    mode         = "backup"
    weight       = 40
    max_fails    = 3
    fail_timeout = 10
    comment      = "Second data center"
  }

  lifecycle {
    create_before_destroy = true
  }
}
