# import own subscription (pass 0 when no client_id is needed)
terraform import edgecenter_useractions_subscription_amqp.subs 0
# import subscription for a specific client using <client_id>
terraform import edgecenter_useractions_subscription_amqp.subs_for_client 124