provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_cdn_rule" "cdn_example_com_rule_1" {
  resource_id = edgecenter_cdn_resource.cdn_example_com.id
  name        = "All PNG images"
  rule        = "/folder/images/*.png"

  options {
    edge_cache_settings {
      default = "14d"
    }
    browser_cache_settings {
      value = "14d"
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
    ignore_query_string {
      value = true
    }
  }
}

resource "edgecenter_cdn_rule" "cdn_example_com_rule_2" {
  resource_id     = edgecenter_cdn_resource.cdn_example_com.id
  name            = "All JS scripts"
  rule            = "/folder/images/*.js"
  origin_protocol = "HTTP"

  options {
    redirect_http_to_https {
      enabled = false
      value   = true
    }
    gzip_on {
      enabled = false
      value   = true
    }
    query_params_whitelist {
      value = [
        "abc",
      ]
    }
  }
}

resource "edgecenter_cdn_origingroup" "origin_group_1" {
  name     = "origin_group_1"
  use_next = true
  origin {
    source  = "example.com"
    enabled = true
  }
}

resource "edgecenter_cdn_resource" "cdn_example_com" {
  cname               = "cdn.example.com"
  origin_group        = edgecenter_cdn_origingroup.origin_group_1.id
  origin_protocol     = "MATCH"
  secondary_hostnames = ["cdn2.example.com"]
}
