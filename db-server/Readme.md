# Database notes



## Membership/User account details

Each user has the following states:

Login Account State
* 'new' - A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin.
* 'active' - An account that has been activated, and is currently active.
* 'disabled' - An account that has been disabled.
* 'reset' - The account is flagged for a password reset. The user will be informed at the next login.
* 'verified' - The email address has been verified. An Admin now needs to activate the account.
**Table name: et_user_account_status**

Membership flag
* 'yes' - They are a current paid up active member
* 'no' - They are not currently a member. This could mean 'expired', 'cancelled', 'non-member', or some other option ???
**Table name: et_member_status**

Age group
* 'infant' - Based on age
* 'child' - Based on age
* 'youth' - Based on age
* 'adult' - Based on age
* 'senior' - Based on age
* 'life' - Based on committiee decision
**Table name: et_user_age_groups**


## Querying trip costs

