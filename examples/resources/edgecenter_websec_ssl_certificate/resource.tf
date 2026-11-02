provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

variable "certificate" {
  type = string
}

variable "private_key" {
  type      = string
  sensitive = true
}

variable "certificates_chain" {
  type    = string
  default = ""
}

resource "edgecenter_websec_ssl_certificate" "example_com" {
  name               = "example.com ${substr(sha256(var.certificate), 0, 8)}"
  certificate        = var.certificate
  private_key        = var.private_key
  certificates_chain = var.certificates_chain

  lifecycle {
    create_before_destroy = true
  }
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

resource "edgecenter_websec_resource" "example_com" {
  domain          = "example.com"
  source_group    = edgecenter_websec_source_group.example_com.id
  ssl_certificate = edgecenter_websec_ssl_certificate.example_com.id

  options {
    tls_versions {
      value = ["TLSv1.2", "TLSv1.3"]
    }
  }
}
