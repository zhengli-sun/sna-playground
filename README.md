# Twilio SNA (Silent Network Auth) Testing

This repository contains Go programs for testing Twilio's [Silent Network Auth (SNA)](https://www.twilio.com/docs/verify/sna) verification flow.

## Prerequisites

1. **Twilio Account**: [Create a Twilio Account](https://www.twilio.com/try-twilio)
2. **API Access**: [Request access](https://www.twilio.com/console/verify/sna) to the SNA API and Live Test Number feature (granted within 24 business hours)
3. **Register Live Test Number**: 
   - After getting API access, you MUST register your specific phone number as a Live Test Number
   - Go to: Console > Verify > Services > [Your Service] > Silent Network Auth
   - Add your mobile number to the Live Test Numbers list
   - Without this, you'll get Error 60008: Unsupported Carrier
4. **Verify Service**: [Create a Verify Service](https://www.twilio.com/console/verify/services) and note the Service SID
5. **Mobile Device**: A smartphone with an approved carrier (e.g., Verizon, T-Mobile, AT&T, EE Limited)

## Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/zhengli-sun/sna-playground.git
   cd sna-playground
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Configure environment variables**:
   
   Create your local environment file from the template:
   ```bash
   cp env.sh.example env.sh
   ```
   
   Edit `env.sh` with your credentials for each region you want to test.
   
   **Important**: Different regions require different Twilio accounts/credentials!
   
   For each region, you need:
   - `TWILIO_ACCOUNT_SID`: From the region-specific [Twilio Console](https://console.twilio.com)
   - `TWILIO_AUTH_TOKEN`: From the region-specific console
   - `TWILIO_VERIFY_SERVICE_SID`: From region-specific [Verify Services](https://console.twilio.com/us1/develop/verify/services)
   - `PHONE_NUMBER`: Your mobile number in E.164 format (e.g., +447857166752 for UK)

4. **Load environment variables**:
   ```bash
   source env.sh
   ```
   
   The script will automatically load the correct credentials based on `TWILIO_REGION`.

## Usage

Run the program:
```bash
source env.sh  # or: . env.sh
go run step1_start_verification.go
```

### Expected Output

The program will:
1. Create a new SNA verification
2. Print the full API response
3. Extract and display the SNA URL

Example output:
```
Starting new SNA verification...

=== Verification Response ===
{
  "sid": "VEaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "service_sid": "VAaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "to": "+15017122661",
  "channel": "sna",
  "status": "pending",
  ...
}

=== SNA URL (use this in Step 2) ===
https://mi.dnlsrv.com/m/id/ANBByzx7?data=AAAglRRdNn02iTFWfDWwdTjOzM...

Note: This URL is unique and can only be used once. It expires in 10 minutes.
```

## Next Steps

After getting the SNA URL:

### Step 2: Invoke the SNA URL

Before tapping the URL, ensure your device is properly configured:

#### iOS Device Settings Checklist ✅

**CRITICAL**: For SNA to work on iOS, you must disable these features that interfere with carrier detection:

1. **Turn OFF Wi-Fi**
   - Settings → Wi-Fi → Toggle OFF
   - Ensure you're using cellular data only

2. **Turn OFF Dual SIM** (if applicable)
   - Use only the SIM card associated with the phone number being verified
   - Settings → Cellular → turn off the other SIM

3. **Turn OFF iCloud Private Relay**
   - Settings → [Your Name] → iCloud → Private Relay → Toggle OFF

4. **Turn OFF "Limit IP Address Tracking (Cellular)"**
   - Settings → Cellular → Toggle OFF "Limit IP Address Tracking"

5. **Turn OFF VPN Profile**
   - Settings → VPN → Disconnect or disable any active VPN
   - Or Settings → General → VPN & Device Management

6. **Turn OFF Safari "Hide IP Address"**
   - Settings → Safari → Hide IP Address → Toggle OFF
   - Or set to "Off" instead of "Trackers and Websites"

**💡 eSIM Cache Issue**: If SNA still fails after configuring the above settings, try this:
- Settings → Cellular → Turn OFF all eSIM cards
- Wait 10 seconds
- Turn the eSIM back ON (the one with the phone number you're verifying)
- Wait for the cellular connection to re-establish
- Then try the SNA URL again

This clears cached network/carrier information that may be interfering with SNA carrier detection.

**Why these settings matter**: SNA requires direct carrier network access. Features like Private Relay, VPN, and IP masking prevent the carrier from properly identifying your phone number, causing verification failures (Error -10 or 60519).

#### Steps to Invoke the URL:

1. Send the URL to yourself via email or Slack
2. Verify all iOS settings above are configured correctly
3. Tap the URL on your mobile device
4. Wait for the success page (or error message)

See the [Testing Guide Step 2](https://www.twilio.com/docs/verify/sna-testing-guide#step-2-invoke-the-sna-url) for more details.

**Important for email:**
- Some email clients (e.g., Microsoft Safe Links) may scan and invalidate the URL
- To work around this, change `https://...` to `hxxps://...` before sending
- Change it back to `https://...` on your device before tapping

**Slack workaround:**
- Create a code block first (Mac: Shift + Option + Command + C, Windows: Ctrl + Alt + Shift + C)
- Paste the SNA URL into the code block
- Send the message

### Step 3: Check the Verification Result

After invoking the SNA URL on your mobile device, run the verification check:

```bash
go run step3_check_verification.go
```

This will:
- Check if the SNA verification was successful
- Display the verification status (`approved` means success)
- Show any error codes if the verification failed
- Provide troubleshooting information

## Important Notes

- The SNA URL is **one-time use only** and expires in **10 minutes**
- When sending via email, be aware of security features like Microsoft Safe Links that may invalidate the URL
- Your mobile device must be on cellular data (not Wi-Fi) when invoking the URL
- The phone number must be associated with an approved carrier
- **CRITICAL**: For UK/European numbers, you MUST use the **IE1 (Ireland) region**, not US1
- **CRITICAL**: Register your phone number as a Live Test Number before testing

## Twilio Regions for SNA

Different phone numbers require different Twilio regions for SNA to work correctly:

| Phone Number Region | Twilio Region | How to Configure |
|---------------------|---------------|------------------|
| UK (+44) | `ie1` (Ireland) | Set `TWILIO_REGION="ie1"` in env.sh |
| Europe (most) | `ie1` (Ireland) | Set `TWILIO_REGION="ie1"` in env.sh |
| US/Canada (+1) | `us1` (United States) | Set `TWILIO_REGION="us1"` in env.sh |
| Australia (+61) | `au1` (Australia) | Set `TWILIO_REGION="au1"` in env.sh |

**CRITICAL**: 
- Using the wrong region will cause **Error 60008: Unsupported Carrier** even if the carrier is supported!
- **Each region may require different Twilio account credentials** - the `env.sh` script handles this automatically
- To switch regions, just change `TWILIO_REGION` at the top of `env.sh` and add the appropriate credentials

## Testing Without Carrier Approval

You can test SNA without carrier approval by:
- Using Twilio Region US1 for your requests
- Following the same steps in this guide
- Expecting an error code at the verification check step (this is normal for testing)

## Complete Testing Flow

1. **Step 1** - Start verification:
   ```bash
   source env.sh
   go run step1_start_verification.go
   ```
   → Copy the SNA URL from the output

2. **Step 2** - Invoke URL on mobile:
   - Send URL to yourself (email/Slack)
   - Turn off Wi-Fi (cellular only)
   - Tap the URL on your mobile device
   - Wait for success page

3. **Step 3** - Check verification:
   ```bash
   go run step3_check_verification.go
   ```
   → Should show status: "approved"

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Error 60008: Unsupported Carrier** | Register your phone number as a Live Test Number in Twilio Console |
| **Error -10 or 60519: Verification Failed/Pending** | 1) Check iOS settings (see Step 2 checklist) 2) Try turning OFF all eSIMs, wait 10s, turn back ON |
| 404 Not Found | Verification expired (10 min limit) - restart from Step 1 |
| URL doesn't work | Ensure Wi-Fi is OFF and using cellular data only |
| Email invalidated URL | Use `hxxps://` trick or Slack code block instead |
| Dual SIM issues | Ensure data connection is on the SIM matching the phone number |
| eSIM not working | Turn OFF all eSIMs, wait 10 seconds, turn back ON to clear carrier cache |
| Error codes present | Check [Error Dictionary](https://www.twilio.com/docs/verify/api/verification#sna-error-codes) |

## Resources

- [Twilio SNA Testing Guide](https://www.twilio.com/docs/verify/sna-testing-guide)
- [SNA Overview](https://www.twilio.com/docs/verify/sna)
- [SNA API Reference](https://www.twilio.com/docs/verify/api/verification)
- [SNA Error Codes](https://www.twilio.com/docs/verify/api/verification#sna-error-codes)
- [What is Silent Network Authentication?](https://www.twilio.com/blog/what-is-silent-network-authentication)

