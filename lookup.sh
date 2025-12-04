curl -X GET "https://lookups.twilio.com/v2/PhoneNumbers/$PHONE_NUMBER?Fields=line_type_intelligence" \
  -u $TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN | jq