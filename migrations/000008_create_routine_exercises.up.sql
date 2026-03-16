CREATE TABLE routine_exercises
(
    id                UUID PRIMARY KEY,
    routine_id        UUID        NOT NULL REFERENCES routines (id) ON DELETE CASCADE,
    exercise_id       UUID        NOT NULL REFERENCES exercises (id),
    sort_order        INT         NOT NULL,
    warm_up_set_count INT         NOT NULL DEFAULT 0,
    set_count         INT         NOT NULL,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT routine_exercises_routine_id_sort_order_key
        UNIQUE (routine_id, sort_order),

    CONSTRAINT routine_exercises_warm_up_set_count_check
        CHECK (warm_up_set_count >= 0),

    CONSTRAINT routine_exercises_set_count_check
        CHECK (set_count > 0)
);

CREATE INDEX idx_routine_exercises_routine_id ON routine_exercises (routine_id);
CREATE INDEX idx_routine_exercises_exercise_id ON routine_exercises (exercise_id);