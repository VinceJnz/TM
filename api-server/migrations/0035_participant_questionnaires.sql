CREATE TABLE at_trip_questionnaires (
    trip_id integer NOT NULL REFERENCES at_trips(id) ON DELETE CASCADE,
    revision integer NOT NULL CHECK (revision > 0),
    questions jsonb NOT NULL CHECK (jsonb_typeof(questions) = 'array'),
    PRIMARY KEY (trip_id, revision)
);

ALTER TABLE et_trip_type
    ADD COLUMN default_questions jsonb NOT NULL DEFAULT '[
        {"key":"need_transport","label":"Do you need transport?","type":"yes_no","required":true},
        {"key":"offer_transport","label":"Can you offer transport?","type":"yes_no","required":true},
        {"key":"transport_details","label":"Transport details, including one-way arrangements or seats available","type":"text","required":false},
        {"key":"stayed_before","label":"Have you stayed at the lodge or hut before?","type":"yes_no","required":false}
    ]'
        CHECK (jsonb_typeof(default_questions) = 'array');

ALTER TABLE at_bookings
    ADD COLUMN requires_participant_details boolean NOT NULL DEFAULT false;
ALTER TABLE at_bookings
    ALTER COLUMN requires_participant_details SET DEFAULT true;

ALTER TABLE at_booking_people
    ADD COLUMN submitted_name text,
    ADD COLUMN mobile_phone text,
    ADD COLUMN home_address text,
    ADD COLUMN membership_category text CHECK (membership_category IN ('member', 'non_member', 'affiliated_member')),
    ADD COLUMN emergency_contact_name text,
    ADD COLUMN emergency_contact_phone text,
    ADD COLUMN dietary_requirements text,
    ADD COLUMN questionnaire_revision integer CHECK (questionnaire_revision > 0),
    ADD COLUMN trip_answers jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(trip_answers) = 'object'),
    ADD COLUMN trip_answer_history jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(trip_answer_history) = 'object');
