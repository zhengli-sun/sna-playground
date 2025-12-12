# [RFC] Twilio Silent Network Auth (SNA) Integration

**Author:** [Zhengli Sun](mailto:zhengli.sun@doordash.com)  
**Created:** Dec 2025  
**Last updated:** Dec 2025  
**Status:** Draft

---

## Executive Summary

This RFC proposes integrating Twilio's Silent Network Auth (SNA) into Deliveroo's login/signup flows to enable frictionless phone number verification. Unlike traditional SMS OTP which requires users to wait for and enter a code, SNA verifies phone numbers directly with mobile carriers without any user action.

---

## 1. What is Silent Network Auth?

SNA (Silent Network Auth) is a carrier-based verification method that:
- Verifies a user's phone number possession directly with the mobile carrier
- Requires **no user interaction** (no code entry)
- Works only on **cellular data** (not Wi-Fi)
- Completes verification in **~4 seconds**
- Falls back to SMS OTP if unavailable

### How It Works (Simplified)
```
1. Backend initiates SNA verification → Twilio returns SNA URL
2. Mobile app invokes SNA URL (via cellular data)
3. Carrier verifies phone number → Twilio receives result
4. Backend checks verification status → Success/Failure
```

---

## 2. Why?

### Goals
- **Reduce friction**: Eliminate wait time and manual code entry for SMS OTP
- **Improve conversion**: Faster login/signup should improve user growth
- **Maintain security**: Carrier-based verification is secure and non-spoofable

### Similar Implementation: DoorDash
DoorDash implemented similar functionality using Sinch Seamless Verification:
- Reported reduced friction in login/signup flows
- Successfully integrated with fallback to SMS OTP
- Used SDK-based approach on mobile clients

---

## 3. High-Level Architecture

### Current SMS OTP Flow
```
User enters phone → co-accounts → Sphinx (SMS) → User enters code → orderweb validates
```

### Proposed SNA Flow
```
User enters phone → co-accounts → Twilio SNA → Mobile invokes URL → co-accounts checks status → orderweb
                                      ↓
                               [Fallback to SMS OTP if SNA fails]
```

### Key Components Affected

| Component | Changes Required |
|-----------|------------------|
| **co-accounts (Go)** | New Twilio SNA client, initiate/check verification |
| **Mobile Apps (iOS/Android)** | Invoke SNA URL via cellular, handle pre-checks |
| **orderweb (Rails)** | Minimal changes - accepts verified sessions |
| **Sphinx** | No changes - used as fallback |

---

## 4. How Existing Flows Change

### 4.1 Current: Existing User Login via SMS OTP
```
1. User enters email → co-accounts POST /check-email
2. User selects "Send code via SMS"
3. co-accounts → Sphinx: send SMS OTP
4. User receives SMS, enters code
5. orderweb validates code via Sphinx
6. User logged in
```

### 4.2 Proposed: Existing User Login via SNA (with fallback)

```
1. User enters phone number
2. User taps "Continue"
3. App shows loading state on button ("Verifying...") - NOT the OTP entry screen
4. App → co-accounts: POST /request-login (with sna=true)
5. co-accounts → Twilio: Start SNA verification
6. co-accounts → App: Returns { sna_url, verification_sid }
7. App invokes SNA URL via cellular data (background HTTP request)
8. App waits for SNA URL response (success/failure) - typically ~4 sec
9. App → co-accounts: POST /sna/verify { verification_sid, sna_result }
10. co-accounts → Twilio: Check verification status (confirm)
11. If APPROVED → co-accounts creates session → User logged in
    If FAILED → App transitions to OTP entry screen → Fallback to SMS OTP flow
```

**Key UX Point**: User sees a **loading button** (not OTP screen) while SNA is attempted. Only if SNA fails do they see the OTP entry screen. This approach is similar to DoorDash's "loading button UI" pattern.

### UX Flow (Loading Button Pattern)

```
┌─────────────────────────┐     ┌─────────────────────────┐
│                         │     │                         │
│  What's your phone?     │     │  What's your phone?     │
│  ┌───────────────────┐  │     │  ┌───────────────────┐  │
│  │ +44 7857 166752   │  │     │  │ +44 7857 166752   │  │
│  └───────────────────┘  │     │  └───────────────────┘  │
│                         │     │                         │
│  ┌───────────────────┐  │     │  ┌───────────────────┐  │
│  │    Continue       │──┼────>│  │  ⟳ Verifying...   │  │
│  └───────────────────┘  │     │  └───────────────────┘  │
│                         │     │   (button disabled)     │
└─────────────────────────┘     └─────────────────────────┘
      User taps Continue              SNA in progress
                                      (< 5 seconds)
                                            │
                    ┌───────────────────────┴───────────────────────┐
                    ▼                                               ▼
          ┌─────────────────────────┐                 ┌─────────────────────────┐
          │                         │                 │                         │
          │      Welcome back!      │                 │   Enter the code we     │
          │                         │                 │   sent to your phone    │
          │      ✓ Verified         │                 │   ┌───────────────────┐ │
          │                         │                 │   │     _ _ _ _       │ │
          │                         │                 │   └───────────────────┘ │
          │                         │                 │                         │
          └─────────────────────────┘                 └─────────────────────────┘
               SNA SUCCESS                                  SNA FAILED
             (User logged in)                          (Fallback to SMS OTP)
```

### Timeout Handling

- **Timeout**: 5 seconds (configurable)
- If SNA doesn't complete within timeout → assume failure → fallback to SMS OTP
- Prevents user from waiting indefinitely

### Sequence Diagram
```
┌──────┐          ┌─────────────┐          ┌───────┐          ┌────────┐
│ User │          │  Mobile App │          │co-acct│          │ Twilio │
└──┬───┘          └──────┬──────┘          └───┬───┘          └───┬────┘
   │                     │                     │                  │
   │  Tap "Continue"     │                     │                  │
   │────────────────────>│                     │                  │
   │                     │                     │                  │
   │      [Show "Verifying..." loading screen] │                  │
   │                     │                     │                  │
   │                     │ POST /request-login │                  │
   │                     │────────────────────>│                  │
   │                     │                     │                  │
   │                     │                     │ Start SNA        │
   │                     │                     │─────────────────>│
   │                     │                     │                  │
   │                     │                     │ { sna_url }      │
   │                     │                     │<─────────────────│
   │                     │                     │                  │
   │                     │ { sna_url, sid }    │                  │
   │                     │<────────────────────│                  │
   │                     │                     │                  │
   │                     │ GET sna_url (cellular)                 │
   │                     │───────────────────────────────────────>│
   │                     │                     │                  │
   │                     │ SNA result (success/fail)              │
   │                     │<───────────────────────────────────────│
   │                     │                     │                  │
   │                     │ POST /sna/verify    │                  │
   │                     │────────────────────>│                  │
   │                     │                     │                  │
   │                     │                     │ Check status     │
   │                     │                     │─────────────────>│
   │                     │                     │                  │
   │                     │                     │ { approved }     │
   │                     │                     │<─────────────────│
   │                     │                     │                  │
   │                     │ { success, token }  │                  │
   │                     │<────────────────────│                  │
   │                     │                     │                  │
   │    [If success: Show "Logged in"]         │                  │
   │    [If failed: Show OTP entry screen]     │                  │
   │<────────────────────│                     │                  │
```

### 4.3 New User Signup
Similar flow - SNA can be used for phone verification during signup.

---

## 5. Technical Approach

### 5.1 co-accounts Changes

**New Twilio SNA Client:**
```go
// Start SNA verification
func (c *TwilioClient) StartSNAVerification(phoneNumber string) (*SNAResponse, error) {
    // POST to Twilio Verify API with channel=sna
    // Returns: verification_sid, sna_url
}

// Check verification status
func (c *TwilioClient) CheckSNAStatus(phoneNumber string) (*StatusResponse, error) {
    // POST to Twilio Verification Check API
    // Returns: status (approved/pending/failed), error_codes
}
```

**New/Modified Endpoints:**

```
POST /request-login
Request:
{
  "phone_number": "+447857166752",
  "primary_mechanism": "sna",  // or "sms" for fallback
  "device_context": {
    "is_cellular": true,
    "is_wifi": false,
    "platform": "ios"
  }
}

Response (SNA):
{
  "verification_sid": "VExxxxx",
  "sna_url": "https://mi.dnlsrv.com/...",
  "mechanism": "sna",
  "timeout_seconds": 10
}

Response (SMS fallback):
{
  "mechanism": "sms",
  "contact": "+44***166752"
}
```

```
POST /sna/verify
Request:
{
  "verification_sid": "VExxxxx",
  "phone_number": "+447857166752",
  "sna_client_result": "success" | "failure" | "timeout"
}

Response:
{
  "status": "approved" | "pending" | "failed",
  "error_code": 60519,  // if failed
  "fallback_to": "sms"  // if failed
}
```

### 5.2 Mobile App Changes

**Pre-checks before SNA (determines if SNA is attempted):**
```swift
// iOS example
func canAttemptSNA() -> Bool {
    return !isWiFiConnected() &&
           isCellularDataEnabled() &&
           !isVPNActive() &&
           !isPrivateRelayEnabled()
}
```

**Login Flow (Pseudocode):**
```swift
func onContinueTapped(phoneNumber: String) {
    // 1. Show loading screen
    showLoadingScreen("Verifying your phone...")
    
    // 2. Check if SNA is possible
    let mechanism = canAttemptSNA() ? "sna" : "sms"
    
    // 3. Request login
    let response = await api.requestLogin(
        phoneNumber: phoneNumber,
        mechanism: mechanism,
        deviceContext: getDeviceContext()
    )
    
    if response.mechanism == "sna" {
        // 4. Invoke SNA URL
        let snaResult = await invokeSNAUrl(response.snaUrl)
        
        // 5. Verify with backend
        let verifyResponse = await api.verifySNA(
            verificationSid: response.verificationSid,
            snaResult: snaResult
        )
        
        if verifyResponse.status == "approved" {
            // 6a. Success - user logged in
            navigateToHome()
        } else {
            // 6b. Failed - fallback to SMS OTP
            await api.requestLogin(phoneNumber, mechanism: "sms")
            showOTPEntryScreen()
        }
    } else {
        // SMS flow - show OTP entry screen directly
        showOTPEntryScreen()
    }
}

func invokeSNAUrl(_ url: String) async -> String {
    // Make HTTP GET request via cellular
    // This triggers carrier verification
    do {
        let response = await URLSession.shared.data(from: URL(string: url)!)
        return "success"
    } catch {
        return "failure"
    }
}
```

**Key Points:**
- App decides SNA vs SMS based on device state (Wi-Fi, VPN, etc.)
- App invokes SNA URL and waits for response (~4 sec)
- App tells backend the result, backend confirms with Twilio
- On SNA failure, app seamlessly transitions to OTP screen

### 5.3 Region Configuration

Twilio requires specific regions for different countries:

| Country | Twilio Region | API Endpoint |
|---------|---------------|--------------|
| UK (+44) | IE1 (Dublin) | `verify.dublin.ie1.twilio.com` |
| US (+1) | US1 (Default) | `verify.twilio.com` |
| AU (+61) | AU1 (Sydney) | `verify.sydney.au1.twilio.com` |

---

## 6. Fallback Strategy

SNA may fail due to:
- User on Wi-Fi (not cellular)
- Unsupported carrier
- VPN/Private Relay enabled
- eSIM cache issues

**Fallback approach:**
1. Attempt SNA first (transparent to user)
2. If SNA fails within 5 seconds → fallback to SMS OTP
3. User experience: minimal delay before falling back

---

## 7. Key Differences from DoorDash/Sinch Approach

| Aspect | DoorDash (Sinch) | Deliveroo (Twilio) |
|--------|------------------|-------------------|
| **SDK Required** | Yes (Sinch SDK) | No (HTTP request only) |
| **Verification** | SDK handles | App invokes URL directly |
| **Backend** | Risk-Gateway | co-accounts |
| **Fallback** | RDP managed | co-accounts → Sphinx |

**Advantage of Twilio approach**: No SDK dependency, simpler mobile integration.

---

## 8. Open Questions / TODOs

### 🔴 TODO 1: Sign-up Flow - In Scope or Out?

**Question**: Should we include new user sign-up in the initial implementation?

**Current state**: This RFC focuses on login flow for existing users.

**Options**:
| Option | Pros | Cons |
|--------|------|------|
| **A: Login only (Phase 1)** | Smaller scope, faster delivery, less risk | Doesn't improve sign-up conversion |
| **B: Login + Sign-up (Full)** | Complete solution, better overall UX | Larger scope, more testing needed |

**Recommendation**: TBD

---

### 🔴 TODO 2: Email-First vs Phone-First Flow

**Question**: Deliveroo currently uses **email** as the primary login identifier. How do we integrate SNA which requires a **phone number**?

**Current Deliveroo flow**:
```
1. User enters EMAIL
2. co-accounts checks email → returns available methods (password, magic link, SMS OTP)
3. If SMS OTP selected → uses phone number associated with account
```

**Challenge**: SNA needs phone number upfront, but users enter email first.

**Options**:
| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Keep email-first** | After email check, if user has phone, attempt SNA transparently | No UX change for users | Extra API call, complexity |
| **B: Add phone-first option** | New flow: "Continue with phone number" | Direct SNA path | Big UX change, user education |
| **C: Hybrid** | Email-first for existing, phone-first for new sign-ups | Best of both | Two different flows to maintain |

**Recommendation**: TBD

**Flow comparison**:
```
Option A (Email-first, current pattern):
  Email → Check → [has phone?] → SNA → Logged in
                       ↓
                  [no phone] → Magic link / Password

Option B (Phone-first, new pattern):
  Phone → SNA → Logged in
           ↓
       [failed] → SMS OTP
```

---

### 🔴 TODO 3: UI During Verification

**Question**: What exactly should the UI show while SNA verification is in progress?

**Options**:
| Option | Description | Example |
|--------|-------------|---------|
| **A: Loading button** | Button text changes to "Verifying..." with spinner | DoorDash pattern |
| **B: Full screen loader** | Overlay or new screen with "Verifying your phone..." | More prominent |
| **C: Subtle indicator** | Small spinner next to button, button stays enabled | Least intrusive |

**Considerations**:
- How long does verification take? (~4 sec typical)
- Should user be able to cancel?
- What happens if user backgrounds the app?
- Accessibility requirements?

**Recommendation**: TBD

**Mockup options**:
```
Option A: Loading Button          Option B: Full Screen
┌─────────────────────┐          ┌─────────────────────┐
│                     │          │                     │
│  +44 7857 166752    │          │    ⟳                │
│                     │          │                     │
│  ┌───────────────┐  │          │  Verifying your     │
│  │ ⟳ Verifying.. │  │          │  phone number...    │
│  └───────────────┘  │          │                     │
│                     │          │                     │
└─────────────────────┘          └─────────────────────┘
```

---

### Other Open Questions

4. **Eligibility criteria**: Should we check carrier support before attempting SNA?
5. **Metrics**: What success/failure rates should trigger changes?
6. **Rollout**: Feature flag by country/carrier first?
7. **Cost**: Twilio SNA pricing vs SMS OTP savings?

---

## 9. Dependencies

- **Twilio Account**: Need separate credentials for IE1 region (UK)
- **Verify Service**: Region-specific service configuration
- **Live Test Numbers**: For testing before carrier approval

---

## 10. Timeline (TBD)

| Phase | Description | Duration |
|-------|-------------|----------|
| 1 | co-accounts Twilio client | X weeks |
| 2 | Mobile app SNA invocation | X weeks |
| 3 | Integration testing | X weeks |
| 4 | Gradual rollout (UK first) | X weeks |

---

## 11. Next Steps

1. Review and discuss architecture approach
2. Define detailed API contracts
3. Estimate engineering effort
4. Plan rollout strategy

---

## References

- [Twilio SNA Testing Guide](https://www.twilio.com/docs/verify/sna-testing-guide)
- [Twilio SNA with Regions](https://www.twilio.com/docs/verify/using-verify-silent-network-auth-with-twilio-regions)
- [DoorDash Sinch Seamless Verification RFC](./[RFC]%20Sinch%20Seamless%20Verification%20.md)
- [Deliveroo Login/Sign-up Flows Audit](./Deliveroo%20Login_Sign-up%20Flows%20Audit.md)
- [SNA Playground (Testing Code)](https://github.com/zhengli-sun/sna-playground)

