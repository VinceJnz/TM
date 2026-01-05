# Info

myuser
mypassword

## Env

<https://towardsdatascience.com/use-environment-variable-in-your-next-golang-project-39e17c3aaa66>


## Cookie handling

there's a whole bunch of settings needed to make cookies work correctly.
see chatGpt thread.

For cookies to be accepted in the browser:
Modern browsers will only accept session cookies via https
The wasm client needs to be served from an https server


## sqlx

Illustrated guide to SQLX
<https://jmoiron.github.io/sqlx/>




## data access management

There are 3 issues to consider:
1. Does the user have access to the resource (Table)?
    1. Not viable: Implement in the DB if the DB has the user ID. The current design does not provide the user ID to the DB sothis cant be implemented
    2. Implement in the handler. The handler has the user ID and resource name, but each hander will be required to run a query to determine access.
    3. Implement a wrapper handler to inspect/check the resource access. Simpler implementation than point 2.

2. What rows does the user have access to?
3. What columns does the user have access to?



User groups provide members access to records (aka resources). The records are filtered by ownership.
User groups have an admin flag. If this is set then members of the group can access all the records of the associated resources regardless of ownership.


## New queries

Count the number participants above and below max_participants threshold

```sql
WITH booking_order AS (
    SELECT 
        atb.trip_id,
        atbp.id as participant_id,
        ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atbp.id ASC) AS booking_position
    FROM public.at_booking_people atbp
	JOIN public.at_bookings atb ON atb.id=atbp.booking_id
)
SELECT 
    trip_id,
    SUM(CASE WHEN booking_position <= att.max_participants THEN 1 ELSE 0 END) AS before_threshold,
    SUM(CASE WHEN booking_position > att.max_participants THEN 1 ELSE 0 END) AS after_threshold,
	STRING_AGG(CASE WHEN booking_position <= att.max_participants THEN participant_id::text END, ', ') AS before_threshold_ids,
    STRING_AGG(CASE WHEN booking_position > att.max_participants THEN participant_id::text END, ', ') AS after_threshold_ids
FROM booking_order
JOIN public.at_trips att ON att.id=booking_order.trip_id
GROUP BY trip_id;
```

List participants above and below max_participants threshold

```sql
WITH booking_order AS (
    SELECT 
        atb.trip_id,
		atb.id as booking_id,
        atbp.id as participant_id,
        atbp.person_id as person_id,
	    ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atbp.id ASC) AS booking_position
    FROM public.at_booking_people atbp
	JOIN public.at_bookings atb ON atb.id=atbp.booking_id
)
SELECT 
    trip_id,
	booking_order.booking_id,
    participant_id,
	booking_order.person_id,
	stu.name,
    CASE 
        WHEN booking_position <= att.max_participants THEN 'before_threshold' 
        ELSE 'after_threshold' 
    END AS booking_status
FROM booking_order
JOIN public.at_trips att ON att.id=booking_order.trip_id
JOIN public.st_users stu ON stu.id=booking_order.person_id
ORDER BY trip_id, booking_position;
```


## Query to get the total costs of a booking

Generic booking list

```sql
SELECT att.id AS trip_id, att.trip_name, atb.id AS booking_id, atb.notes AS booking_notes, atb.owner_id, SUM(attc.amount) AS booking_cost, COUNT(stu.name) as person_count
FROM at_trips att
LEFT JOIN at_bookings atb ON atb.trip_id=att.id
LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
LEFT JOIN st_users stu ON stu.id=atbp.person_id
LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
						AND attc.member_status_id=stu.member_status_id
						AND attc.user_age_group_id=stu.user_age_group_id
GROUP BY att.id, att.trip_name, atb.id
ORDER BY att.trip_name, atb.id
```

My Booking list

```sql
SELECT att.trip_name, atb.*, ebs.status, SUM(attc.amount) AS booking_cost, COUNT(stu.name) as participants
FROM at_trips att
LEFT JOIN at_bookings atb ON atb.trip_id=att.id
LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
	 JOIN public.et_booking_status ebs on ebs.id=atb.booking_status_id
LEFT JOIN st_users stu ON stu.id=atbp.person_id
LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
						AND attc.member_status_id=stu.member_status_id
						AND attc.user_age_group_id=stu.user_age_group_id
WHERE atb.owner_id = $1 OR true=$2
GROUP BY att.id, att.trip_name, atb.id, ebs.status
ORDER BY att.trip_name, atb.id
```



## Money

Packages that support money in the database

<https://pkg.go.dev/github.com/anz-bank/decimal>

<https://pkg.go.dev/github.com/shopspring/decimal#NullDecimal>

Use postgress numeric type to store money.

<https://www.postgresql.org/docs/current/datatype-numeric.html>



## webAuthn query

```sql
SELECT id, user_id, credential_data, created, modified, credential_id, device_name, device_metadata,
	to_json(encode(credential_id, 'base64'))::text as credential_id_json,
	to_json(encode(credential_id, 'hex'))::text as credential_id_hex_json,
	to_json(convert_from(credential_id, 'UTF8'))::text as credential_id_text_json,
    array_to_string(
        array(
            SELECT '0x' || lpad(upper(to_hex(get_byte(credential_id, i))), 2, '0')
            FROM generate_series(0, octet_length(credential_id) - 1) i
        ),
        ', '
    ) || 
    '}' as credential_id_go_byte_slice,

	    '[]byte{' || 
    array_to_string(
        array(
            SELECT get_byte(credential_id, i)::text
            FROM generate_series(0, octet_length(credential_id) - 1) i
        ),
        ', '
    ) || 
    '}' as credential_id_go_decimal_byte_slice

	
	FROM public.st_webauthn_credentials
	--WHERE credential_id = '\xe2eb6e24f314ee802192e98fddbbc423'::bytea
```

## go error fix

`No packages found for open file D:\Users\Vince\Documents\GitHub\TM\api-server\localHandlers\handlerWEbAuthnMgnt\handlerWEbAuthnMgnt.go`

<https://github.com/golang/vscode-go/issues/2715>

go mod init
go clean -modcache
go mod tidy




`chown root:root wasm_exec.js main.wasm`



## Environment file
.env


```conf
# api-server .env file

# Application configuration
APP_TITLE=Trip Manager

# Database configuration
DB_TYPE=postgres
DB_USER=api_user
DB_PASSWORD=api_password
DB_NAME=mydatabase
DB_HOST=dbserver
DB_PORT=5432

# Server configuration
HOST=localhost
HTTP_PORT=8085
HTTPS_PORT=8086
CORE_ORIGINS=http://localhost:8086
API_PATH_PREFIX=/api/v1
SERVER_CA_CERT=
CLIENT_CA_CERT=
SERVER_KEY=/etc/certs/ssl/localhost.key
SERVER_CERT=/etc/certs/ssl/localhost.crt
CERT_OPTION=
LOG_FILE=

# Email configuration
EMAIL_ADDR=apptesting@gmail.com
EMAIL_TOKEN=/etc/certs/gmail/client_token.json
EMAIL_SECRET=/etc/certs/gmail/client_secret.json

# Payment configuration
PAYMENT_KEY=/etc/certs/stripe/payment_key.json

# oAuth configuration
# your_google_client_id    
GOOGLE_CLIENT_ID=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX.apps.googleusercontent.com
# your_google_client_secret
GOOGLE_CLIENT_SECRET=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
GOOGLE_REDIRECT_URL=https://localhost:8086/api/v1/auth/oauth/callback
# your_session_key
SESSION_KEY=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
CLIENT_REDIRECT_URL=https://localhost:8086
DEV=true
```

