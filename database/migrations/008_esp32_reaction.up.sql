CREATE TABLE box_devices (
    box_id text PRIMARY KEY REFERENCES boxes(id) ON DELETE CASCADE,
    token_hash text NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    enabled boolean NOT NULL DEFAULT true,
    last_seen_at timestamptz,
    firmware_version text CHECK (firmware_version IS NULL OR char_length(firmware_version) BETWEEN 1 AND 64),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reaction_game_states (
    game_id uuid PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    next_trigger_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reaction_game_states_due_idx ON reaction_game_states(next_trigger_at)
    WHERE next_trigger_at IS NOT NULL;

CREATE TABLE reaction_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    box_id text NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('solo', 'duel')),
    status text NOT NULL CHECK (status IN (
        'awaiting_assignment', 'awaiting_device', 'armed',
        'resolved', 'expired', 'cancelled'
    )),
    button_s2_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    button_s3_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    delay_ms integer NOT NULL CHECK (delay_ms BETWEEN 2000 AND 6000),
    winner_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    false_start_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    reaction_ms integer CHECK (reaction_ms IS NULL OR reaction_ms >= 0),
    awarded_points integer NOT NULL DEFAULT 0 CHECK (awarded_points >= 0),
    result_event_id text UNIQUE CHECK (
        result_event_id IS NULL OR char_length(result_event_id) BETWEEN 1 AND 160
    ),
    scheduled_at timestamptz NOT NULL DEFAULT now(),
    assigned_at timestamptz,
    armed_at timestamptz,
    resolved_at timestamptz,
    expires_at timestamptz NOT NULL,
    CHECK (button_s2_player_id IS NULL OR button_s3_player_id IS NULL OR button_s2_player_id <> button_s3_player_id),
    CHECK (winner_player_id IS NULL OR status = 'resolved'),
    CHECK (false_start_player_id IS NULL OR status = 'resolved'),
    CHECK (result_event_id IS NULL OR status = 'resolved')
);
CREATE UNIQUE INDEX one_active_reaction_per_game ON reaction_challenges(game_id)
    WHERE status IN ('awaiting_assignment', 'awaiting_device', 'armed');
CREATE INDEX reaction_challenges_expiry_idx ON reaction_challenges(expires_at)
    WHERE status IN ('awaiting_assignment', 'awaiting_device', 'armed');
CREATE INDEX reaction_challenges_game_idx ON reaction_challenges(game_id, scheduled_at DESC);

CREATE TABLE device_commands (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    box_id text NOT NULL REFERENCES box_devices(box_id) ON DELETE CASCADE,
    challenge_id uuid NOT NULL UNIQUE REFERENCES reaction_challenges(id) ON DELETE CASCADE,
    type text NOT NULL CHECK (type = 'reaction_arm'),
    payload jsonb NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'acknowledged', 'completed', 'expired')),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    acknowledged_at timestamptz,
    completed_at timestamptz,
    CHECK (jsonb_typeof(payload) = 'object')
);
CREATE INDEX device_commands_poll_idx ON device_commands(box_id, created_at)
    WHERE status IN ('pending', 'acknowledged');
