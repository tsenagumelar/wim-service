-- public.master_device_type definition

-- Drop table

-- DROP TABLE public.master_device_type;

CREATE TABLE public.master_device_type (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(50) NOT NULL,
	type_name varchar(100) NOT NULL,
	description text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_device_type_code_key UNIQUE (code),
	CONSTRAINT master_device_type_pkey PRIMARY KEY (id)
);

-- Table Triggers

create trigger trigger_generate_device_type_code before
insert
    on
    public.master_device_type for each row execute function auto_generate_device_type_code();


-- public.master_role definition

-- Drop table

-- DROP TABLE public.master_role;

CREATE TABLE public.master_role (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(50) NOT NULL,
	role_name varchar(100) NOT NULL,
	description text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_role_code_key UNIQUE (code),
	CONSTRAINT master_role_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_master_role_active ON public.master_role USING btree (is_active) WHERE (is_deleted = false);

-- Table Triggers

create trigger trigger_generate_role_code before
insert
    on
    public.master_role for each row execute function auto_generate_role_code();


-- public.master_site definition

-- Drop table

-- DROP TABLE public.master_site;

CREATE TABLE public.master_site (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(50) NOT NULL, -- Unique site code identifier (e.g., SITE001, JKT-TOLL-01)
	site_name varchar(200) NOT NULL,
	site_location varchar(200) NULL,
	site_region varchar(100) NULL, -- Region/area of the site for grouping
	description text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_site_code_key UNIQUE (code),
	CONSTRAINT master_site_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_master_site_active ON public.master_site USING btree (is_active) WHERE (is_deleted = false);
CREATE INDEX idx_master_site_region ON public.master_site USING btree (site_region);
COMMENT ON TABLE public.master_site IS 'Master data for all sites in the multi-site architecture';

-- Column comments

COMMENT ON COLUMN public.master_site.code IS 'Unique site code identifier (e.g., SITE001, JKT-TOLL-01)';
COMMENT ON COLUMN public.master_site.site_region IS 'Region/area of the site for grouping';


-- public.master_vehicle_class definition

-- Drop table

-- DROP TABLE public.master_vehicle_class;

CREATE TABLE public.master_vehicle_class (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(50) NOT NULL,
	"type" varchar(100) NOT NULL,
	description varchar(150) NOT NULL,
	total_axle int4 NOT NULL,
	class_2_weight numeric(10, 2) NOT NULL,
	class_3_weight numeric(10, 2) NOT NULL,
	length numeric(10, 2) NOT NULL,
	width numeric(10, 2) NOT NULL,
	height numeric(10, 2) NOT NULL,
	image text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NOT NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT ck_axle_positive CHECK (((total_axle >= 1) AND (total_axle <= 20))),
	CONSTRAINT ck_dim_positive CHECK (((length > (0)::numeric) AND (width > (0)::numeric) AND (height > (0)::numeric))),
	CONSTRAINT ck_dim_reasonable CHECK (((length <= (40)::numeric) AND (width <= (5)::numeric) AND (height <= (6)::numeric))),
	CONSTRAINT ck_weight_positive CHECK (((class_2_weight >= (0)::numeric) AND (class_3_weight >= (0)::numeric))),
	CONSTRAINT master_vehicle_class_pkey PRIMARY KEY (id),
	CONSTRAINT uq_master_vehicle_class_code UNIQUE (code),
	CONSTRAINT uq_master_vehicle_class_type UNIQUE (type)
);
CREATE INDEX idx_vehicle_class_active ON public.master_vehicle_class USING btree (is_active) WHERE (is_active = true);
CREATE INDEX idx_vehicle_class_axle ON public.master_vehicle_class USING btree (total_axle);
CREATE INDEX idx_vehicle_class_type ON public.master_vehicle_class USING btree (type);

-- Table Triggers

create trigger trg_master_vehicle_class_updated before
update
    on
    public.master_vehicle_class for each row execute function set_updated_timestamp();
create trigger trigger_generate_vehicle_class_code before
insert
    on
    public.master_vehicle_class for each row execute function auto_generate_vehicle_class_code();


-- public.master_config definition

-- Drop table

-- DROP TABLE public.master_config;

CREATE TABLE public.master_config (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(100) NOT NULL,
	config_type varchar(100) NOT NULL,
	config_key varchar(100) NOT NULL,
	config_value varchar(255) NULL,
	description text NULL,
	sort_order int4 NULL DEFAULT 0,
	parent_code varchar(100) NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_config_code_key UNIQUE (code),
	CONSTRAINT master_config_config_type_config_key_key UNIQUE (config_type, config_key),
	CONSTRAINT master_config_pkey PRIMARY KEY (id),
	CONSTRAINT master_config_parent_code_fkey FOREIGN KEY (parent_code) REFERENCES public.master_config(code) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE INDEX idx_master_config_parent_code ON public.master_config USING btree (parent_code);
CREATE INDEX idx_master_config_type_key ON public.master_config USING btree (config_type, config_key);

-- Table Triggers

create trigger trigger_generate_config_code before
insert
    on
    public.master_config for each row execute function auto_generate_config_code();


-- public.master_device definition

-- Drop table

-- DROP TABLE public.master_device;

CREATE TABLE public.master_device (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(100) NOT NULL,
	device_name varchar(150) NOT NULL,
	device_type_id uuid NOT NULL,
	model varchar(100) NULL,
	serial_number varchar(100) NULL,
	description text NULL,
	"location" text NULL,
	status varchar(20) NULL DEFAULT 'ACTIVE'::character varying,
	ip_address varchar(50) NULL,
	mac_address varchar(50) NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_device_code_key UNIQUE (code),
	CONSTRAINT master_device_pkey PRIMARY KEY (id),
	CONSTRAINT master_device_status_check CHECK (((status)::text = ANY (ARRAY[('ACTIVE'::character varying)::text, ('MAINTENANCE'::character varying)::text, ('RETIRED'::character varying)::text]))),
	CONSTRAINT master_device_device_type_id_fkey FOREIGN KEY (device_type_id) REFERENCES public.master_device_type(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX idx_master_device_active ON public.master_device USING btree (is_active) WHERE (is_deleted = false);
CREATE INDEX idx_master_device_type ON public.master_device USING btree (device_type_id);

-- Table Triggers

create trigger trigger_generate_device_code before
insert
    on
    public.master_device for each row execute function auto_generate_device_code();


-- public.master_user definition

-- Drop table

-- DROP TABLE public.master_user;

CREATE TABLE public.master_user (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	code varchar(50) NOT NULL,
	username varchar(100) NOT NULL,
	password_hash text NOT NULL,
	full_name varchar(150) NOT NULL,
	badge_no varchar(50) NULL,
	phone_number varchar(30) NULL,
	email varchar(150) NULL,
	role_id uuid NOT NULL,
	profile_picture text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT master_user_code_key UNIQUE (code),
	CONSTRAINT master_user_pkey PRIMARY KEY (id),
	CONSTRAINT master_user_username_key UNIQUE (username),
	CONSTRAINT master_user_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.master_role(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX idx_master_user_role ON public.master_user USING btree (role_id);

-- Table Triggers

create trigger trigger_generate_user_code before
insert
    on
    public.master_user for each row execute function auto_generate_user_code();


-- public.transact_anpr_capture definition

-- Drop table

-- DROP TABLE public.transact_anpr_capture;

CREATE TABLE public.transact_anpr_capture (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	external_id varchar(100) NOT NULL,
	plate_no varchar(32) NOT NULL,
	confidence numeric(5, 2) NULL,
	captured_at timestamptz NULL,
	location_code varchar(100) NULL,
	camera_id varchar(100) NULL,
	minio_bucket varchar(100) NOT NULL,
	minio_date_folder varchar(8) NOT NULL,
	minio_xml_object text NOT NULL,
	minio_full_image_object text NOT NULL,
	minio_plate_image_object text NOT NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	site_id uuid NULL, -- Site where this capture occurred
	CONSTRAINT transact_anpr_capture_external_id_key UNIQUE (external_id),
	CONSTRAINT transact_anpr_capture_pkey PRIMARY KEY (id),
	CONSTRAINT fk_anpr_site FOREIGN KEY (site_id) REFERENCES public.master_site(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX idx_anpr_site ON public.transact_anpr_capture USING btree (site_id);

-- Column comments

COMMENT ON COLUMN public.transact_anpr_capture.site_id IS 'Site where this capture occurred';


-- public.transact_axle_capture definition

-- Drop table

-- DROP TABLE public.transact_axle_capture;

CREATE TABLE public.transact_axle_capture (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	external_id varchar(100) NOT NULL,
	plate_no varchar(32) NULL,
	captured_at timestamptz NULL,
	camera_id varchar(100) NULL,
	length_mm int4 NULL,
	total_wheels int4 NULL,
	total_axles int4 NULL,
	vehicle_category varchar(50) NULL,
	vehicle_body_type varchar(50) NULL,
	minio_bucket varchar(100) NOT NULL,
	minio_date_folder varchar(8) NOT NULL,
	minio_xml_object text NOT NULL,
	minio_image_object text NOT NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	site_id uuid NULL, -- Site where this measurement occurred
	CONSTRAINT transact_axle_capture_external_id_key UNIQUE (external_id),
	CONSTRAINT transact_axle_capture_pkey PRIMARY KEY (id),
	CONSTRAINT fk_axle_site FOREIGN KEY (site_id) REFERENCES public.master_site(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX idx_axle_site ON public.transact_axle_capture USING btree (site_id);

-- Column comments

COMMENT ON COLUMN public.transact_axle_capture.site_id IS 'Site where this measurement occurred';


-- public.user_login_history definition

-- Drop table

-- DROP TABLE public.user_login_history;

CREATE TABLE public.user_login_history (
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	user_id uuid NOT NULL,
	login_time timestamptz NULL DEFAULT now(),
	logout_time timestamptz NULL,
	ip_address varchar(100) NULL,
	user_agent text NULL,
	device_info text NULL,
	token_id text NULL,
	is_active bool NULL DEFAULT true,
	is_deleted bool NULL DEFAULT false,
	created_by uuid NULL,
	created_date timestamptz NULL DEFAULT now(),
	updated_by uuid NULL,
	updated_date timestamptz NULL DEFAULT now(),
	CONSTRAINT user_login_history_pkey PRIMARY KEY (id),
	CONSTRAINT user_login_history_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.master_user(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX idx_login_history_time ON public.user_login_history USING btree (login_time DESC);
CREATE INDEX idx_login_history_user ON public.user_login_history USING btree (user_id);