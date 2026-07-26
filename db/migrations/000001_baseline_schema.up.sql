BEGIN;

CREATE FUNCTION public.record_initial_service_request_status() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
                BEGIN
                        INSERT INTO service_request_status_history (
                                service_request_id,
                                previous_status,
                                new_status,
                                changed_by_role,
                                changed_by_user_id,
                                comment,
                                created_at
                        )
                        VALUES (
                                NEW.id,
                                NULL,
                                NEW.status,
                                'customer',
                                NEW.customer_id,
                                'Request submitted',
                                NEW.created_at
                        );

                        RETURN NEW;
                END;
                $$;


CREATE TABLE public.customers (
    id integer NOT NULL,
    name character varying(255),
    email character varying(255),
    password_hash character varying(255)
);


CREATE SEQUENCE public.customers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.customers_id_seq OWNED BY public.customers.id;

CREATE TABLE public.doers (
    id integer NOT NULL,
    name character varying(255),
    email character varying(255),
    password_hash character varying(255),
    category character varying(255),
    description text,
    zipcode character varying(50),
    radius integer,
    facebook character varying(255),
    tiktok character varying(255),
    instagram character varying(255),
    flyer_url character varying(255)
);


CREATE SEQUENCE public.doers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.doers_id_seq OWNED BY public.doers.id;


CREATE TABLE public.notifications (
    id bigint NOT NULL,
    recipient_role character varying(20) NOT NULL,
    recipient_id integer NOT NULL,
    notification_type character varying(80) NOT NULL,
    title character varying(255) NOT NULL,
    message text NOT NULL,
    action_url character varying(500) DEFAULT ''::character varying NOT NULL,
    reference_type character varying(80) DEFAULT ''::character varying NOT NULL,
    reference_id character varying(255) DEFAULT ''::character varying NOT NULL,
    is_read boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    read_at timestamp with time zone,
    CONSTRAINT notifications_recipient_role_check CHECK (((recipient_role)::text = ANY ((ARRAY['customer'::character varying, 'doer'::character varying])::text[])))
);


CREATE SEQUENCE public.notifications_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.notifications_id_seq OWNED BY public.notifications.id;


CREATE TABLE public.password_reset_tokens (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    role character varying(20) NOT NULL,
    token_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT password_reset_role_check CHECK (((role)::text = ANY ((ARRAY['customer'::character varying, 'doer'::character varying])::text[])))
);


CREATE SEQUENCE public.password_reset_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.password_reset_tokens_id_seq OWNED BY public.password_reset_tokens.id;


CREATE TABLE public.rsvps (
    event_id character varying(255) NOT NULL,
    customer_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.service_request_status_history (
    id bigint NOT NULL,
    service_request_id bigint NOT NULL,
    previous_status character varying(20),
    new_status character varying(20) NOT NULL,
    changed_by_role character varying(20) NOT NULL,
    changed_by_user_id integer NOT NULL,
    comment text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT service_request_status_history_changed_by_role_check CHECK (((changed_by_role)::text = ANY ((ARRAY['customer'::character varying, 'doer'::character varying, 'system'::character varying])::text[]))),
    CONSTRAINT service_request_status_history_new_status_check CHECK (((new_status)::text = ANY ((ARRAY['pending'::character varying, 'accepted'::character varying, 'rejected'::character varying, 'cancelled'::character varying, 'completed'::character varying])::text[])))
);


CREATE SEQUENCE public.service_request_status_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.service_request_status_history_id_seq OWNED BY public.service_request_status_history.id;


CREATE TABLE public.service_request_submission_tokens (
    token_hash character(64) NOT NULL,
    customer_id integer NOT NULL,
    service_id character varying(64) NOT NULL,
    service_request_id bigint,
    request_fingerprint character(64),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone
);


CREATE TABLE public.service_requests (
    id bigint NOT NULL,
    service_id character varying(64) NOT NULL,
    service_title character varying(255) NOT NULL,
    service_price integer NOT NULL,
    customer_id integer NOT NULL,
    doer_id integer NOT NULL,
    message text NOT NULL,
    requested_date date NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    doer_response text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT service_requests_message_check CHECK (((char_length(message) >= 10) AND (char_length(message) <= 2000))),
    CONSTRAINT service_requests_service_price_check CHECK ((service_price >= 0)),
    CONSTRAINT service_requests_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'accepted'::character varying, 'rejected'::character varying, 'cancelled'::character varying, 'completed'::character varying])::text[])))
);


CREATE SEQUENCE public.service_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.service_requests_id_seq OWNED BY public.service_requests.id;


CREATE TABLE public.sessions (
    token_hash character varying(64) NOT NULL,
    role character varying(20) NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT sessions_role_check CHECK (((role)::text = ANY ((ARRAY['doer'::character varying, 'customer'::character varying])::text[])))
);


ALTER TABLE ONLY public.customers ALTER COLUMN id SET DEFAULT nextval('public.customers_id_seq'::regclass);


ALTER TABLE ONLY public.doers ALTER COLUMN id SET DEFAULT nextval('public.doers_id_seq'::regclass);


ALTER TABLE ONLY public.notifications ALTER COLUMN id SET DEFAULT nextval('public.notifications_id_seq'::regclass);


ALTER TABLE ONLY public.password_reset_tokens ALTER COLUMN id SET DEFAULT nextval('public.password_reset_tokens_id_seq'::regclass);


ALTER TABLE ONLY public.service_request_status_history ALTER COLUMN id SET DEFAULT nextval('public.service_request_status_history_id_seq'::regclass);


ALTER TABLE ONLY public.service_requests ALTER COLUMN id SET DEFAULT nextval('public.service_requests_id_seq'::regclass);

ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_email_key UNIQUE (email);


ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.doers
    ADD CONSTRAINT doers_email_key UNIQUE (email);


ALTER TABLE ONLY public.doers
    ADD CONSTRAINT doers_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_token_hash_key UNIQUE (token_hash);


ALTER TABLE ONLY public.rsvps
    ADD CONSTRAINT rsvps_pkey PRIMARY KEY (event_id, customer_id);


ALTER TABLE ONLY public.service_request_status_history
    ADD CONSTRAINT service_request_status_history_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.service_request_submission_tokens
    ADD CONSTRAINT service_request_submission_tokens_pkey PRIMARY KEY (token_hash);


ALTER TABLE ONLY public.service_requests
    ADD CONSTRAINT service_requests_pkey PRIMARY KEY (id);


ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token_hash);


CREATE INDEX idx_notifications_recipient ON public.notifications USING btree (recipient_role, recipient_id, created_at DESC);

CREATE INDEX idx_notifications_unread ON public.notifications USING btree (recipient_role, recipient_id, created_at DESC) WHERE (is_read = false);


CREATE INDEX idx_password_reset_tokens_expiry ON public.password_reset_tokens USING btree (expires_at);


CREATE INDEX idx_password_reset_tokens_hash ON public.password_reset_tokens USING btree (token_hash);


CREATE INDEX idx_rsvps_customer ON public.rsvps USING btree (customer_id, created_at DESC);


CREATE INDEX idx_service_request_history_request ON public.service_request_status_history USING btree (service_request_id, created_at, id);


CREATE INDEX idx_service_request_tokens_customer ON public.service_request_submission_tokens USING btree (customer_id, created_at DESC);


CREATE INDEX idx_service_request_tokens_expiry ON public.service_request_submission_tokens USING btree (expires_at) WHERE (consumed_at IS NULL);


CREATE INDEX idx_service_requests_customer ON public.service_requests USING btree (customer_id, created_at DESC);


CREATE INDEX idx_service_requests_customer_service ON public.service_requests USING btree (customer_id, service_id, created_at DESC);


CREATE INDEX idx_service_requests_doer ON public.service_requests USING btree (doer_id, created_at DESC);


CREATE INDEX idx_service_requests_status ON public.service_requests USING btree (status);


CREATE INDEX idx_sessions_expiration ON public.sessions USING btree (expires_at);


CREATE INDEX idx_sessions_user ON public.sessions USING btree (role, user_id);


CREATE UNIQUE INDEX ux_notifications_delivery ON public.notifications USING btree (recipient_role, recipient_id, notification_type, reference_type, reference_id);


CREATE TRIGGER trg_service_request_initial_status AFTER INSERT ON public.service_requests FOR EACH ROW EXECUTE FUNCTION public.record_initial_service_request_status();


ALTER TABLE ONLY public.service_request_status_history
    ADD CONSTRAINT service_request_status_history_service_request_id_fkey FOREIGN KEY (service_request_id) REFERENCES public.service_requests(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.service_request_submission_tokens
    ADD CONSTRAINT service_request_submission_tokens_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


ALTER TABLE ONLY public.service_request_submission_tokens
    ADD CONSTRAINT service_request_submission_tokens_service_request_id_fkey FOREIGN KEY (service_request_id) REFERENCES public.service_requests(id) ON DELETE SET NULL;


ALTER TABLE ONLY public.service_requests
    ADD CONSTRAINT service_requests_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


ALTER TABLE ONLY public.service_requests
    ADD CONSTRAINT service_requests_doer_id_fkey FOREIGN KEY (doer_id) REFERENCES public.doers(id) ON DELETE CASCADE;

COMMIT;