provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_useractions_subscription_amqp" "subs" {
  connection_string = "amqps://guest:guest@192.168.123.20:5671/user_action_events"
  //exchange                    = "abce"
  receive_child_client_events = true
  routing_key                 = "foo"
}


