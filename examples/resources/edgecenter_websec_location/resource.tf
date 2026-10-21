provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_websec_source_group" "example_com" {
  name = "example.com origins"

  source {
    source = "203.0.113.10"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "edgecenter_websec_source_group" "example_com_api" {
  name = "example.com api origins"

  source {
    source = "203.0.113.20"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "edgecenter_websec_resource" "example_com" {
  domain       = "example.com"
  source_group = edgecenter_websec_source_group.example_com.id

  options {
    tls_versions {
      value = ["TLSv1.2", "TLSv1.3"]
    }
  }
}

resource "edgecenter_websec_location" "example_com_api" {
  resource_id  = edgecenter_websec_resource.example_com.id
  name         = "api"
  location     = "^/api/"
  weight       = 10
  source_group = edgecenter_websec_source_group.example_com_api.id

  options {
    allowed_http_methods {
      value = ["GET", "POST"]
    }

    asn_acl {
      policy_type     = "allow"
      excepted_values = [64496, 64511]
    }

    antibot_flood_challenge {
      value = "ANTIBOT_CHALLENGE_JS"
    }
  }
}
