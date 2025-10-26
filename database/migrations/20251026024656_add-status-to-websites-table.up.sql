ALTER TABLE websites ADD status TEXT DEFAULT 'active' NOT NULL;

CREATE INDEX idx_websites_status ON websites(status);
