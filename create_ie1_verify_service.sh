#!/bin/bash
# Create a Verify Service in IE1 region
# Usage: bash create_ie1_verify_service.sh

# Make sure to set your credentials first
if [ -z "$TWILIO_ACCOUNT_SID" ] || [ -z "$TWILIO_AUTH_TOKEN" ]; then
    echo "Error: Please source env.sh first to set credentials"
    echo "Usage: source env.sh && bash create_ie1_verify_service.sh"
    exit 1
fi

curl -X POST "https://verify.dublin.ie1.twilio.com/v2/Services" \
--data-urlencode "FriendlyName=My IE1 Verify Service" \
-u $TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN
