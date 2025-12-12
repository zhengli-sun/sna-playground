# Deliveroo Login/Sign-up Flows Audit

Author:[Zhengli Sun](mailto:zhengli.sun@doordash.com)  
Last updated: Dec 11, 2025

## Executive Summary

This document documents the existing architecture that currently serves login/signup flows' production traffic in Deliveroo, and the dependencies to support those flows. 

Two main services involved:  
**Orderweb (Rails monolith)**: Legacy service handling the majority of production authentication traffic  
  \- Password login, consumer social login (Apple, Google, Facebook), password reset, user management  
  \- Well-established but monolithic architecture  
    
**co-accounts (Go microservice)**: Modern service with partial production launch  
  \- Currently handles: Email checking, magic link/SMS OTP initiation, SAML SSO (enterprise)  
  \- Has infrastructure ready for: Consumer social login, password reset (not yet serving traffic)

---

## 1\. AUTHENTICATION METHODS SUMMARY

### Available Methods (Production)

| Method | Entry Point | Service | Status | Flow |
| :---- | :---- | :---- | :---- | :---- |
| **Password** | orderweb | orderweb | ✅ Production | Classic email \+ password |
| **Magic Link (Email)** | co-accounts → orderweb | orderweb | ✅ Production | Passwordless via email |
| **SMS OTP** | co-accounts → orderweb | orderweb \+ Sphinx | ✅ Production | Passwordless via SMS |
| **Apple Sign In** | orderweb | orderweb \+ Identity | ✅ Production | OAuth2 federated (consumer) |
| **Google Sign In** | orderweb | orderweb \+ Identity | ✅ Production | OAuth2 federated (consumer) |
| **Facebook Classic** | orderweb | orderweb \+ Identity | ✅ Production | OAuth2 federated (consumer) |
| **Facebook Limited** | orderweb | orderweb \+ Identity | ✅ Production | OAuth2 federated (consumer) |
| **SAML SSO** | co-accounts | co-accounts \+ Identity \+ DFW \+ orderweb | ✅ Production | Enterprise SSO |

### Available Methods (Future \- co-accounts Migration Target)

| Method | Entry Point | Service | Status | Notes |
| :---- | :---- | :---- | :---- | :---- |
| **Social Login (Apple/Google/FB)** | co-accounts | co-accounts \+ Identity \+ orderweb | 🚧 Ready but not in use | Infrastructure exists, consumer social login migration pending |

---

## 2\. ARCHITECTURE OVERVIEW

### 2.1 High-Level Auth Architecture

![][image1]  
([link to mermaid](https://www.mermaidchart.com/app/projects/af6a0d93-4bd1-43b2-a6e7-5c8a1812ae60/diagrams/8c9a8f45-b508-45fc-836f-c5a7510d5d69/version/v0.1/edit))

### 2.2 Key Components

**Main Services:**

- **orderweb (Rails)**:  
    
  - Status: Primary authentication service for consumers  
  - Handles: Password login, password reset, consumer social login (Apple, Google, Facebook), user management, magic link generation  
  - Role: Monolithic service handling majority of consumer auth traffic


- **co-accounts (Go)**:  
    
  - Status: Partial migration complete  
  - Handles: Email checking, magic link/SMS OTP initiation, SAML SSO (enterprise)  
  - Role: Modern microservice gradually taking over auth responsibilities  
  - Note: Has infrastructure for consumer social login and password reset but not serving that traffic yet

**Core Services:**

- **Identity**: OAuth2/OIDC provider for federated authentication (Apple, Google, Facebook, SAML)  
- **Sphinx**: SMS verification and OTP delivery  
- **CATS**: Centralized session token management  
- **DFW**: Enterprise SSO (SAML) configuration

**Security & Analytics:**

- **Ravelin**: Real-time fraud detection and risk scoring  
- **StickyNotes**: User blocking and systematic risk control (SARC)  
- **Confettura**: Feature flags and A/B testing  
- **Franz/Kafka**: Event streaming for analytics

### 2.3 Migration Context

**Current State:**

- orderweb handles: Password login, password reset, consumer social login (Apple, Google, Facebook)  
- co-accounts handles: Email checking, magic link/SMS OTP initiation, SAML SSO (enterprise)  
- Magic link and SMS OTP validation still goes back to orderweb

---

## 3\. COMMON FLOWS SUMMARY

### 3.1 New User Sign-Up via Social Login (via orderweb)

```
1. User clicks "Sign in with Apple" on Consumer Web App
2. Consumer Web App → orderweb POST /api/auth/login
   (with federated_token_type=apple_id_token, federated_token=<token>)
3. orderweb → Identity Service: validate Apple token
4. orderweb → CreateUser: create new user account
5. orderweb → Sphinx: check verification if needed
6. orderweb → Sets session cookies
7. orderweb → Franz: publish signup event
8. orderweb → Returns user data
9. Consumer Web App: User logged in
```

### 3.2 Existing User Login via SMS OTP

```
1. User enters email on Consumer Web App
2. Consumer Web App → co-accounts POST /consumer/accounts/check-email
3. co-accounts → orderweb: fetch_user_data (found, has phone number)
4. co-accounts → Returns: registered=true, providers=['email', 'sms']
5. User selects "Send code via SMS"
6. Consumer Web App → co-accounts POST /request-login (primary_mechanism=sms)
7. co-accounts → Confettura: check AuthEmailSMSLoginFlow variant
8. co-accounts → orderweb: fetch_user_data (get phone number)
9. co-accounts → Sphinx: POST /internal/send_code (send SMS OTP)
10. User receives SMS with verification code
11. User enters verification code in Consumer Web App
12. Consumer Web App → orderweb: authenticate with OTP
13. orderweb → Sphinx: validate verification code
14. orderweb → Sets session cookies
15. orderweb → Franz: publish login event
16. orderweb → Returns user data, user logged in
```

### 3.3 Existing User Login via Magic Link

```
1. User enters email on Consumer Web App
2. Consumer Web App → co-accounts POST /check-email
3. co-accounts → orderweb: fetch_user_data (found, has 'email' provider)
4. co-accounts → Returns: registered=true, providers=['email']
5. User clicks "Send magic link"
6. Consumer Web App → co-accounts POST /request-login
7. co-accounts → orderweb: generate_magic_link (sends email)
8. User clicks link in email
9. Consumer Web App → orderweb GET /login/magic-link with passcode
10. orderweb → Sphinx: validate passcode
11. orderweb → Sets session, redirects to app
```

### 3.4 Enterprise User Login via SAML SSO (via co-accounts)

```
1. User enters corporate email on Consumer Web App
2. Consumer Web App → co-accounts POST /check-email
3. co-accounts → DFW: IsSSOEnabledForUser (true)
4. co-accounts → DFW: GetSSOSAMLParam
5. co-accounts → Returns SAML metadata
6. Consumer Web App: Redirect to IdP with SAML request
7. User authenticates at IdP
8. IdP → Redirects to assertion consumer URL with SAML response
9. Consumer Web App → co-accounts POST /consumer/accounts/auth/login (saml_assertion)
10. co-accounts → Identity: validate SAML token
11. co-accounts → orderweb: fetch_user_data or create_user
12. co-accounts → CATS: create_session_token
13. co-accounts → Sets cookies, returns user data
14. Consumer Web App: User logged in
```

### 3.6 Existing User Login via Password

```
1. User enters email + password on Consumer Web App
2. Consumer Web App → orderweb POST /api/auth/login
3. orderweb → Ravelin: score login attempt
4. orderweb → StickyNotes: check user blocks
5. orderweb → Identity/local: verify password
6. orderweb → Check if MFA required
7. If MFA: orderweb → Sphinx: send SMS code, wait for validation
8. orderweb → Sets session cookies
9. orderweb → Franz: publish login event
10. orderweb → Returns user data, user logged in
```

---

## 4\. PRODUCTION ENDPOINTS

### 4.1 co-accounts Endpoints

**Migration Status**: co-accounts is partially deployed. It handles discovery/initiation flows and SAML SSO, but most authentication still happens in orderweb.

#### `POST /consumer/accounts/check-email`

**Production Status**: Active, high volume

**Purpose**: Determine available authentication methods for a given email address

**Behavior**:

1. Checks if user's email domain has SSO enabled (via DFW service)  
2. If SSO enabled: Returns SAML metadata for redirect  
3. If not SSO: Queries orderweb for user account existence  
4. Returns available identity providers (`email`, `apple`, `google`, `facebook`, `saml`)  
5. Checks if magic links are blocked (via StickyNotes SARC)  
6. Publishes analytics event (known/unknown user email entered)

**Response**:

```json
{
  "registered": true,
  "identity_providers": ["email", "google", "apple"],
  "identity_provider_hint": "sign_in_with_google",
  "magic_links": true
}
```

**Dependencies**: orderweb, DFW, StickyNotes, Franz Analytics

---

#### `POST /request-login`

**Production Status**: Active

**Purpose**: Initiate passwordless login via magic link (email) or SMS OTP

**Behavior**:

1. Validates email and primary\_mechanism parameter (`email` or `sms`)  
2. Checks SSO status (short-circuits if SSO enabled)  
3. Fetches user data from orderweb (gets phone number if SMS selected)  
4. Checks Confettura feature flag (`AuthEmailSMSLoginFlow`) for cohort assignment  
5. Based on mechanism and cohort:  
   - **Email**: Calls orderweb to generate and send magic link  
   - **SMS**: Calls Sphinx to send OTP to user's phone  
6. Implements fallback logic (SMS→email if SMS fails)  
7. Publishes analytics events

**Request**:

```json
{
  "email": "user@example.com",
  "primary_mechanism": "sms",
  "is_fallback_option": false
}
```

**Dependencies**: orderweb, Sphinx, Confettura, DFW, Franz Analytics

---

#### `POST /consumer/accounts/auth/login`

**Production Status**: Active for SAML SSO only, consumer social login not active

**Purpose**: Federated login endpoint (Apple, Google, Facebook, SAML)

**Behavior**:

- SAML SSO (Enterprise): Active \- Handles enterprise SAML authentication  
    
  1. Receives `saml_assertion` from Identity Provider  
  2. Validates token with Identity service  
  3. Fetches or creates user in orderweb  
  4. Creates session via CATS  
  5. Returns authenticated user


- Consumer Social Login (Apple/Google/Facebook): Not active  
    
  - Infrastructure complete but consumer social login still routes to orderweb  
  - Migration from orderweb `/api/auth/login` pending

**Request (SAML)**:

```json
{
  "federated_token_type": "saml_assertion",
  "federated_token": "<saml_response>",
  "client_name": "consumer_web_app"
}
```

**Dependencies**: Identity, orderweb, CATS, Franz Analytics

---

### 4.2 orderweb Endpoints

**Production Status**: Primary authentication service handling majority of production traffic

#### `POST /api/auth/login`

**Production Status**: Active, high volume

**Purpose**: Primary endpoint for password and consumer social login

**Behavior**: This endpoint handles MULTIPLE authentication methods:

**1\. Password Authentication** (when `password` param present):

- Receives email and password  
- Validates password against Identity service  
- Runs fraud detection (Ravelin)  
- Checks user blocks (StickyNotes SARC)  
- Determines if MFA required  
- If MFA: sends SMS code via Sphinx  
- Creates session and sets cookies  
- Returns user data

**2\. Consumer Social Login** (when `federated_token` param present):

- Receives federated\_token\_type (`apple_id_token`, `google_id_token`, `facebook_access_token`, `facebook_id_token`)  
- Validates token with Identity service  
- Extracts user email and profile  
- Creates user if doesn't exist  
- Sets session cookies  
- Returns user data

**Request (Password)**:

```json
{
  "email": "user@example.com",
  "password": "********"
}
```

**Request (Social Login)**:

```json
{
  "federated_token_type": "apple_id_token",
  "federated_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "verification_secret": "optional-sphinx-secret"
}
```

**Dependencies**: Identity, Ravelin, StickyNotes, Sphinx, Franz

**Migration Note**: Consumer social login will eventually migrate to co-accounts. SAML SSO has migrated to co-accounts.

---

#### `POST /api/internal/co_accounts/fetch_user_data`

**Production Status**: Active, high volume (\~1,609 requests/min)

**Purpose**: Primary user lookup endpoint for co-accounts integration

**Behavior**:

1. Looks up user by email address or user ID  
2. Returns comprehensive user data including:  
   - User ID and DRN (Deliveroo Resource Name)  
   - Email, mobile phone number  
   - Registration status (true/false)  
   - Employee status  
   - Available identity providers (linked accounts)  
3. Does NOT authenticate \- pure data retrieval

**Response**:

```json
{
  "user_id": 12345,
  "user_drn": "drn:roo:user:gb:12345",
  "email": "user@example.com",
  "mobile": "+44xxxxxxxxx",
  "registered": true,
  "employee": false,
  "providers": ["email", "google", "apple"]
}
```

**Key Insight**: The 1:1 call ratio with check-email validates tight integration \- every email check triggers a user lookup.

---

#### `POST /api/internal/co_accounts/create_user`

**Production Status**: Active (sign-ups only)

**Purpose**: Create new user account during sign-up flow

**Behavior**:

1. Validates user data (email, name, password if provided)  
2. Checks for duplicate email addresses  
3. Creates security principal in Identity service  
4. Saves user to database with marketing preferences  
5. Sends verification data to Sphinx (if phone provided)  
6. Publishes user creation event to Segment analytics  
7. Sends fraud detection data to Ravelin  
8. Returns newly created user with DRN

**Request**:

```json
{
  "email": "newuser@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "country": "gb",
  "locale": "en",
  "mobile": "+44xxxxxxxxx",
  "marketing_email": true
}
```

**Dependencies**: Identity, Sphinx, Segment, Ravelin

---

#### `POST /api/internal/co_accounts/update_user`

**Production Status**: Active, low volume (\~1 request/min)

**Purpose**: Update user details after authentication

**Behavior**:

1. Updates user locale preference  
2. Updates last login timestamp  
3. Updates device information  
4. Minimal traffic confirms it's only called post-authentication, not on every check

**Request**:

```json
{
  "user_id": 12345,
  "locale": "en-GB",
  "tld": "co.uk"
}
```

---

#### `POST /internal/generate_magic_link`

**Production Status**: Active (called by co-accounts)

**Purpose**: Generate passwordless login link and send via email

**Behavior**:

1. Generates time-limited passcode  
2. Creates magic link URL with passcode  
3. Sends email via Messaging service with magic link  
4. Passcode valid for limited time (typically 15-30 minutes)  
5. Single-use passcode (consumed on successful validation)

**Request**:

```json
{
  "email": "user@example.com",
  "page_in_progress": "login",
  "redirect_path": "/"
}
```

**Dependencies**: Messaging service, Sphinx (for passcode generation)

---

#### `GET /login/magic-link`

**Production Status**: Active

**Purpose**: Validate magic link passcode and authenticate user

**Behavior**:

1. Extracts passcode from URL parameters  
2. Validates passcode with Sphinx service  
3. If valid: Creates user session  
4. Sets authentication cookies  
5. Redirects to specified path or home  
6. Passcode is consumed (single-use)

**Query Parameters**:

```
?email=user@example.com&passcode=ABC123&redirect_path=/menu
```

**Dependencies**: Sphinx

---

#### `POST /api/password_reset` (family of endpoints)

**Production Status**: Active

**Purpose**: Password reset flow

**Endpoints**:

- `GET /api/password_reset/` \- Show password reset form  
- `POST /api/password_reset/` \- Initiate reset (send email)  
- `GET /api/password_reset/entry` \- Validate reset token  
- `POST /api/password_reset/entry` \- Submit new password

**Behavior**:

1. User requests password reset  
2. Validates email exists  
3. Creates password reset token  
4. Sends reset email via Messaging service with link  
5. User clicks link, validates token  
6. User submits new password  
7. Password updated in Identity service

**Dependencies**: Identity, Messaging service, Franz Analytics

---

### 4.3 Production Traffic Summary

**High-Volume Endpoints:**

- `orderweb POST /api/auth/login` \- Primary consumer authentication (password \+ social login)  
- `co-accounts POST /consumer/accounts/check-email` \- Auth method discovery (\~1,630 req/min)  
- `orderweb POST /api/internal/co_accounts/fetch_user_data` \- User lookup (\~1,609 req/min)  
- `co-accounts POST /consumer/accounts/auth/login` \- SAML SSO (enterprise only)

**Service Responsibilities (Current Production State):**

| Flow Type | Discovery (co-accounts) | Authentication |
| :---- | :---- | :---- |
| **Password Login** | Direct to orderweb | orderweb `/api/auth/login` |
| **Password Reset** | Direct to orderweb | orderweb `/api/password_reset/` |
| **Social Login (Apple/Google/FB)** | Check-email returns providers | orderweb `/api/auth/login` with federated\_token |
| **SAML SSO (Enterprise)** | Check-email \+ DFW | co-accounts `/consumer/accounts/auth/login` |
| **Magic Link** | Request-login initiates | orderweb `/login/magic-link` validates |
| **SMS OTP** | Request-login sends SMS | orderweb password auth with OTP |

**Migration Status:**

- SAML SSO migrated to co-accounts (enterprise auth)  
- Consumer social login and password reset still in orderweb  
- co-accounts has infrastructure for consumer social login and password reset but not active

---

## 5\. MICROSERVICE DEPENDENCIES & ENDPOINTS

### 5.1 Sphinx (Phone Verification Service)

**Purpose**: SMS verification, OTP, and phone-based authentication

**Endpoints used by orderweb:**

- `POST /verification_secret_usable` \- Check if verification secret is valid  
- `POST /add_account_to_verification` \- Associate account with verification  
- `POST /internal/send_code` \- Send SMS verification code  
- `POST /internal/validate_code` \- Validate SMS verification code

**Endpoints used by co-accounts:**

- `POST /verification_secret_usable` \- Check secret validity  
- `POST /add_account_to_verification` \- Link account to verification  
- `POST /internal/send_code` \- Send SMS OTP

**Authentication**: Basic Auth (username/password)

---

### 5.2 Identity Service

**Purpose**: OAuth2/OIDC identity provider, manages security principals and credentials

**Endpoints used by co-accounts:**

- `POST /o/token/authenticate` \- Authenticate federated tokens (Apple/Google/Facebook/SAML)  
- `POST /oauth/token` \- OAuth2 token endpoint  
  - **Production Volume**: \~1 request/min  
- `POST /o/oauth/introspect` \- Introspect federated tokens  
- `POST /v1/security_principals` \- Create security principal for user  
- `POST /v1/security_principals/{uid}/credentials` \- Create password credential  
- `DELETE /v1/credentials/{uid}` \- Delete credential  
- `PATCH /v1/credentials/{uid}` \- Update credential  
- `GET /id/for/account/drn:roo:user:{country}:{guid}` \- Look up user identity by DRN

**Authentication**: Basic Auth (username/password)

**Token Types Supported:**

- `apple_id_token` \- Apple Sign In  
- `google_id_token` \- Google Sign In  
- `facebook_access_token` \- Facebook Classic Login  
- `facebook_id_token` \- Facebook Limited Login  
- `saml_token` \- Enterprise SSO

---

### 5.3 CATS (Consumer Auth Token Service)

**Purpose**: Session token management and authentication

**Endpoints used by co-accounts (gRPC):**

- `CreateToken` \- Create new session token for user  
- `RefreshToken` \- Refresh/validate existing token  
- `DeleteToken` \- Delete specific token  
- `DeleteTokenInstance` \- Delete token instance by hash  
- `DeleteAllTokensByUser` \- Revoke all tokens for user

**Authentication**: gRPC with circuit breaker

**Response**: Returns both token (for cookie) and JWT

---

### 5.4 DFW (Deliveroo For Work/SSO Association Service)

**Purpose**: Enterprise SSO configuration and management

**Endpoints used by co-accounts (gRPC):**

- `GetSSOSAMLParam` \- Get SAML configuration for domain  
- Integration with Confettura for SSO enablement flags

**Response Includes:**

- Identity Provider SSO URL  
- Identity Provider Issuer  
- Service Provider Issuer  
- Assertion Consumer Service URL  
- Sign AuthN Requests flag  
- Client Name

**SSO Flow:**

1. Check if user's email domain has SSO enabled (Confettura flag)  
2. Fetch SAML parameters from DFW  
3. Generate SAML request  
4. Redirect to IdP for authentication  
5. Handle SAML response at assertion consumer service URL

---

### 5.5 StickyNotes Service

**Purpose**: User blocking/flagging for security reasons (SARC \- Systematic Account Risk Control)

**Endpoints used by orderweb:**

- Check if user has active SARC note (blocks magic links)  
- Manage user blocks and risk flags

**Endpoints used by co-accounts:**

- `HasActiveSarcNote(user_drn)` \- Check if user is blocked

---

### 5.6 Confettura (Feature Flag Service)

**Purpose**: Feature flags and A/B testing

**Flags used in auth flows:**

- `sms_verification_account_creation` \- Enable SMS verification for new accounts  
- `disable_employee_password_signup` \- Block employee email signup  
- `auth_email_sms_login_flow` \- Determine email vs SMS login flow  
- `dfw-sso` (namespace: dfw) \- Enable SSO for specific domains

**Integration**: Actor-based feature flags with user/request context

---

### 5.7 Ravelin (Fraud Detection Service)

**Purpose**: Real-time fraud detection and risk scoring

**Used in orderweb:**

- `Purchasing::Ravelin::CheckoutService` \- Score authentication attempts  
- Track login attempts, device fingerprints, behavioral signals  
- Block high-risk authentication attempts

**Integration**: HTTP API with transaction scoring

---

### 5.8 Franz/Kafka (Event Streaming)

**Purpose**: Event publishing for analytics and downstream services

**Events published during auth:**

- `ConsumerKnownUserEmailEntered` \- Existing user entered email  
- `ConsumerUnknownUserEmailEntered` \- New user entered email  
- `VerificationCodeSendAttempted` \- SMS OTP send attempted  
- `MagicLinkLoginEmailSent` \- Magic link email sent  
- `MagicLinkLoginEmailFailed` \- Magic link email failed  
- Login success/failure events with metadata

**Integration**: Kafka producer through Franz client

---

### 5.9 Messaging Service

**Purpose**: Email delivery for transactional emails

**Used for:**

- Magic link emails  
- Password reset emails  
- Welcome emails  
- Verification emails

---

## 6\. KEY SECURITY CONTROLS

### 6.1 Fraud Detection (Ravelin)

- All password logins scored by Ravelin  
- Device fingerprinting  
- Behavioral analysis  
- Risk-based blocking

### 6.2 Account Blocking (StickyNotes)

- SARC notes block magic link authentication  
- Systematic risk control  
- Admin-managed blocks

### 6.3 Rate Limiting (Sphinx)

- SMS OTP rate limiting  
- Verification cap per phone number  
- Prevents abuse of verification system

### 6.4 MFA (Multi-Factor Authentication)

- SMS-based MFA for high-risk logins  
- Forced MFA for certain user segments  
- Challenges issued via Sphinx

### 6.5 Account Takeover Detection (Beancounter)

- Monitors for suspicious login patterns  
- Triggers additional verification steps  
- Integrated into password authentication flow

### 6.6 Session Management (CATS)

- Secure session token generation  
- Token rotation on refresh  
- Device-based token management  
- Ability to revoke all sessions

[image1]: <data:image/png;base64,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>