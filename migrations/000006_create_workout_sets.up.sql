CREATE TABLE workout_sets
(
    id                  UUID PRIMARY KEY,
    workout_exercise_id UUID        NOT NULL REFERENCES workout_exercises (id) ON DELETE CASCADE,
    set_number          INT         NOT NULL,
    set_type            TEXT        NOT NULL DEFAULT 'normal',
    reps                INT,
    weight              NUMERIC(8, 2),
    notes               TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT workout_sets_workout_exercise_id_set_number_key
        UNIQUE (workout_exercise_id, set_number),

    CONSTRAINT workout_sets_set_number_check
        CHECK (set_number >= 0),

    CONSTRAINT workout_sets_reps_check
        CHECK (reps IS NULL OR reps >= 0),

    CONSTRAINT workout_sets_weight_check
        CHECK (weight IS NULL OR weight >= 0),

    CONSTRAINT workout_sets_set_type_check
        CHECK (set_type IN ('warm_up', 'normal', 'drop_set'))
);

CREATE INDEX idx_workout_sets_workout_exercise_id
    ON workout_sets (workout_exercise_id);