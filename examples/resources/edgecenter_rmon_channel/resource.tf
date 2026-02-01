provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_channel" "telegram_alerts" {
  receiver     = "telegram"
  token        = "123456:example-telegram-bot-token"
  channel_name = "rmon-alerts"
}
