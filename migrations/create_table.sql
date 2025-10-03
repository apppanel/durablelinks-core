-- Create the apppanel_durable_links table
-- This matches the exact schema expected by the repository

CREATE TABLE IF NOT EXISTS apppanel_durable_links (
    -- Primary key and core fields
    id SERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    path VARCHAR(255) NOT NULL,
    link TEXT NOT NULL,
    is_unguessable_path BOOLEAN DEFAULT FALSE NOT NULL,
    project_id UUID,

    -- Android parameters
    android_package_name VARCHAR(255),
    android_fallback_link TEXT,
    android_min_version VARCHAR(50),

    -- iOS parameters
    ios_fallback_link TEXT,
    ios_ipad_fallback_link TEXT,
    ios_app_store_id BIGINT,

    -- Social meta tags
    social_title VARCHAR(500),
    social_description TEXT,
    social_image_link TEXT,

    -- Marketing parameters (UTM)
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term VARCHAR(255),
    utm_content VARCHAR(255),

    -- iTunes Connect Analytics
    itunes_pt VARCHAR(255),
    itunes_at VARCHAR(255),
    itunes_ct VARCHAR(255),
    itunes_mt VARCHAR(50),

    -- Other platform parameters
    other_fallback_url TEXT,

    params_hash VARCHAR(64),

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() NOT NULL,

    -- Constraints
    CONSTRAINT unique_multi_tenant UNIQUE(project_id, host, path),
    CONSTRAINT unique_single_tenant UNIQUE(host, path) WHERE (project_id IS NULL)
);

CREATE INDEX IF NOT EXISTS idx_durable_links_find_existing
    ON apppanel_durable_links(host, link, params_hash, is_unguessable_path)
    WHERE is_unguessable_path = FALSE;

CREATE INDEX IF NOT EXISTS idx_durable_links_project_id
    ON apppanel_durable_links(project_id) WHERE project_id IS NOT NULL;

COMMENT ON COLUMN apppanel_durable_links.host IS 'The short link domain (e.g., example.page.link)';
COMMENT ON COLUMN apppanel_durable_links.path IS 'The short path/code (e.g., abc123)';
COMMENT ON COLUMN apppanel_durable_links.project_id IS 'Multi-tenant support: identifies which project/tenant owns this link';