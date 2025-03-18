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

resource "edgecenter_cdn_resource" "cdn_example_com" {
  cname               = "cdn.example.com"
  origin_group        = edgecenter_cdn_origingroup.origin_group_1.id
  origin_protocol     = "MATCH"
  secondary_hostnames = ["cdn2.example.com"]

  options {
    edge_cache_settings {
      default = "8d"
    }
    browser_cache_settings {
      value = "1d"
    }
    redirect_http_to_https {
      value = true
    }
    gzip_compression {
      enabled = true
      value = [
        "application/x-font-ttf",
        "text/javascript",
        "image/svg+xml",
        "image/x-icon",
      ]
    }
    cors {
      value = [
        "*"
      ]
    }
    rewrite {
      body = "/(.*) /$1"
    }
    image_stack {
      quality      = 80
      avif_enabled = true
      webp_enabled = false
      png_lossless = true
    }

    tls_versions {
      enabled = true
      value = [
        "TLSv1.2",
      ]
    }
  }
}
