CREATE TABLE workout_exercises (
                                   id UUID PRIMARY KEY,
                                   workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
                                   exercise_id UUID NOT NULL REFERENCES exercises(id),
                                   sort_order INT NOT NULL,
                                   notes TEXT,
                                   created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

                                   CONSTRAINT workout_exercises_workout_id_sort_order_key
                                       UNIQUE (workout_id, sort_order),

                                   CONSTRAINT workout_exercises_sort_order_check
                                       CHECK (sort_order >= 0)
);

CREATE INDEX idx_workout_exercises_workout_id ON workout_exercises(workout_id);