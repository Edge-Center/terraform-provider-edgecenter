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
    gzip_on {
      value = true
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
