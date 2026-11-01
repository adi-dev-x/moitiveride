-- Enable PostGIS extension for geospatial support
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;

-- ─────────────────────────────────────────────────────────────────────────────
-- COMPANIES
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE companies (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(255) NOT NULL,
    contact_email VARCHAR(320),
    active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SHIFTS
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE shift_type AS ENUM ('MORNING', 'AFTERNOON', 'EVENING', 'NIGHT');

CREATE TABLE shifts (
    id         UUID       PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID       NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    start_time TIME       NOT NULL,
    end_time   TIME       NOT NULL,
    shift_type shift_type NOT NULL,
    active     BOOLEAN    NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shifts_company ON shifts(company_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- BOARDING / DROP POINTS
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE transport_points (
    id        UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name      VARCHAR(255) NOT NULL,
    lat       DECIMAL(10,8) NOT NULL,
    lng       DECIMAL(11,8) NOT NULL,
    geom      GEOMETRY(Point, 4326),
    point_type VARCHAR(20) NOT NULL CHECK (point_type IN ('BOARDING', 'DROP', 'OFFICE'))
);

CREATE INDEX idx_transport_points_geom ON transport_points USING GIST(geom);

-- ─────────────────────────────────────────────────────────────────────────────
-- EMPLOYEES
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE gender_type AS ENUM ('MALE', 'FEMALE', 'OTHER');

CREATE TABLE employees (
    id                  UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id          UUID          NOT NULL REFERENCES companies(id),
    shift_id            UUID          NOT NULL REFERENCES shifts(id),
    employee_code       VARCHAR(100)  NOT NULL,
    name                VARCHAR(255)  NOT NULL,
    gender              gender_type   NOT NULL,
    home_lat            DECIMAL(10,8) NOT NULL,
    home_lng            DECIMAL(11,8) NOT NULL,
    home_geom           GEOMETRY(Point, 4326),
    boarding_point_id   UUID          REFERENCES transport_points(id),
    drop_point_id       UUID          REFERENCES transport_points(id),
    pickup_window_start TIME,
    pickup_window_end   TIME,
    drop_window_start   TIME,
    drop_window_end     TIME,
    active              BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_employee_code_company UNIQUE (employee_code, company_id)
);

CREATE INDEX idx_employees_company   ON employees(company_id);
CREATE INDEX idx_employees_shift     ON employees(shift_id);
CREATE INDEX idx_employees_home_geom ON employees USING GIST(home_geom);
CREATE INDEX idx_employees_active    ON employees(active) WHERE active = TRUE;

-- Trigger to keep home_geom in sync with home_lat/home_lng
CREATE OR REPLACE FUNCTION sync_employee_geom() RETURNS TRIGGER AS $$
BEGIN
    NEW.home_geom := ST_SetSRID(ST_MakePoint(NEW.home_lng, NEW.home_lat), 4326);
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_employee_geom
BEFORE INSERT OR UPDATE OF home_lat, home_lng ON employees
FOR EACH ROW EXECUTE FUNCTION sync_employee_geom();

-- ─────────────────────────────────────────────────────────────────────────────
-- DRIVERS
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE drivers (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    phone      VARCHAR(20)  NOT NULL UNIQUE,
    license_no VARCHAR(50)  NOT NULL,
    active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- ESCORTS
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE escort_status AS ENUM ('AVAILABLE', 'ASSIGNED');

CREATE TABLE escorts (
    id         UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255)  NOT NULL,
    phone      VARCHAR(20)   NOT NULL UNIQUE,
    status     escort_status NOT NULL DEFAULT 'AVAILABLE',
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- VEHICLES
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE vehicle_status AS ENUM ('AVAILABLE', 'ON_ROUTE', 'MAINTENANCE');

CREATE TABLE vehicles (
    id           UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    registration VARCHAR(20)    NOT NULL UNIQUE,
    vehicle_type VARCHAR(50)    NOT NULL,
    capacity     INTEGER        NOT NULL CHECK (capacity > 0),
    driver_id    UUID           REFERENCES drivers(id),
    status       vehicle_status NOT NULL DEFAULT 'AVAILABLE',
    current_lat  DECIMAL(10,8),
    current_lng  DECIMAL(11,8),
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicles_status   ON vehicles(status);
CREATE INDEX idx_vehicles_capacity ON vehicles(capacity);

-- ─────────────────────────────────────────────────────────────────────────────
-- ROUTES
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE route_direction AS ENUM ('PICKUP', 'DROP');
CREATE TYPE route_status    AS ENUM ('PLANNED', 'ACTIVE', 'COMPLETED', 'CANCELLED');

CREATE TABLE routes (
    id               UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id       UUID            NOT NULL REFERENCES vehicles(id),
    shift_id         UUID            NOT NULL REFERENCES shifts(id),
    company_id       UUID            NOT NULL REFERENCES companies(id),
    route_date       DATE            NOT NULL,
    direction        route_direction NOT NULL,
    status           route_status    NOT NULL DEFAULT 'PLANNED',
    total_distance_km DECIMAL(8,2)   DEFAULT 0,
    total_duration_min INTEGER       DEFAULT 0,
    escort_required  BOOLEAN         NOT NULL DEFAULT FALSE,
    escort_id        UUID            REFERENCES escorts(id),
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routes_date      ON routes(route_date);
CREATE INDEX idx_routes_shift     ON routes(shift_id);
CREATE INDEX idx_routes_company   ON routes(company_id);
CREATE INDEX idx_routes_status    ON routes(status);
CREATE INDEX idx_routes_vehicle   ON routes(vehicle_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- ROUTE STOPS
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE route_stops (
    id                UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    route_id          UUID          NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    employee_id       UUID          NOT NULL REFERENCES employees(id),
    sequence_order    INTEGER       NOT NULL,
    stop_lat          DECIMAL(10,8) NOT NULL,
    stop_lng          DECIMAL(11,8) NOT NULL,
    stop_geom         GEOMETRY(Point, 4326),
    estimated_arrival TIMESTAMPTZ,
    actual_arrival    TIMESTAMPTZ,
    boarded           BOOLEAN       NOT NULL DEFAULT FALSE,

    CONSTRAINT uq_route_stop_seq UNIQUE (route_id, sequence_order),
    CONSTRAINT uq_route_employee UNIQUE (route_id, employee_id)
);

CREATE INDEX idx_route_stops_route    ON route_stops(route_id);
CREATE INDEX idx_route_stops_employee ON route_stops(employee_id);
CREATE INDEX idx_route_stops_geom     ON route_stops USING GIST(stop_geom);

CREATE OR REPLACE FUNCTION sync_stop_geom() RETURNS TRIGGER AS $$
BEGIN
    NEW.stop_geom := ST_SetSRID(ST_MakePoint(NEW.stop_lng, NEW.stop_lat), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_stop_geom
BEFORE INSERT OR UPDATE OF stop_lat, stop_lng ON route_stops
FOR EACH ROW EXECUTE FUNCTION sync_stop_geom();

-- ─────────────────────────────────────────────────────────────────────────────
-- ROUTING JOB LOG  (tracks each routing engine run for auditing)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TYPE job_status AS ENUM ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED');

CREATE TABLE routing_jobs (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID        REFERENCES companies(id),
    shift_id        UUID        REFERENCES shifts(id),
    route_date      DATE        NOT NULL,
    direction       route_direction NOT NULL,
    status          job_status  NOT NULL DEFAULT 'PENDING',
    employees_count INTEGER     DEFAULT 0,
    routes_count    INTEGER     DEFAULT 0,
    duration_ms     BIGINT,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
