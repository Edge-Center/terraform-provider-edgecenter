provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_cdn_origingroup" "origin_group_1" {
  name     = "origin_group_1"
  use_next = true
  origin {
    source  = "example.com"
    enabled = true
  }
  origin {
    source  = "mirror.example.com"
    enabled = true
    backup  = true
  }

  authorization {
    access_key_id = "test_access_key_id"
    auth_type     = "aws_signature_v2"
    bucket_name   = "test_bucket_name"
    secret_key    = "keywqueiuqwiueiqweqwiueiqwiueuiqw"
  }

  consistent_balancing = true
}
