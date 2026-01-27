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



## PostgreSQL back up settings

Instructions: <http://127.0.0.1:65398/help/help/backup_dialog.html>

**3 files**
1. Full tar backup (file name: dbserver-pgAdmin backup1.sql)
2. Full plain backup (file name: dbserver-pgAdmin backup2.sql)
3. Schema build (file name: dbserver-pgAdmin backup3.sql)


### 1. Settings for Full tar backup (file name: dbserver-pgAdmin backup1.sql)

**General**
* File name: dbserver-pgAdmin backup1.sql
* Format: Tar

**Data options**
* Sections: Pre-data, Data, Post-data
* Type of objects: defaults - all off
* Do not save: defaults - all off

**Query Options**
* Use INSERT Commands
* On conflict do nothing to INSERT command
* Include CREATE DATABASE statement
* Include DROP DATABASE statement
* Include IF EXIST clause

**Table Options**
* defaults - all off

**Options**
* Disable: defaults - all off
* Miscellaneous: Verbose messages, remainder defaults - all off

**Objects**
* select public (all opbjects)



### 2. Settings for Full plain backup (file name: dbserver-pgAdmin backup2.sql)

**General**
* File name: dbserver-pgAdmin backup1.sql
* Format: Plain

**Data options**
* Sections: Pre-data, Data, Post-data
* Type of objects: defaults - all off
* Do not save: defaults - all off

**Query Options**
* Use INSERT Commands
* On conflict do nothing to INSERT command
* Include CREATE DATABASE statement
* Include DROP DATABASE statement
* Include IF EXIST clause

**Table Options**
* defaults - all off

**Options**
* Disable: defaults - all off
* Miscellaneous: Verbose messages, remainder defaults - all off

**Objects**
* select public (all opbjects)


### 3. Settings for Schema build (file name: dbserver-pgAdmin backup3.sql)

**General**
* File name: dbserver-pgAdmin backup1.sql
* Format: Plain

**Data options**
* Sections: defaults - all off
* Type of objects: Only schemas
* Do not save: defaults - all off

**Query Options**
* Use INSERT Commands
* On conflict do nothing to INSERT command
* Include CREATE DATABASE statement

**Table Options**
* defaults - all off

**Options**
* Disable: defaults - all off
* Miscellaneous: Verbose messages, remainder defaults - all off

**Objects**
* select public (all opbjects)





