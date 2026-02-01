provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_smtp" "example" {
  name     = "smtp-example"
  enabled  = true
  place    = "country"
  entities = [1]
  ip       = "smtp.example.com"
  port     = 587
  username = "monitor"
  password = "example-password"
}
