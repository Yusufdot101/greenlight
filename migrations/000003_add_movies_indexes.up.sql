CREATE INDEX IF NOT EXISTS movies_title_idx ON movies USING GIN
(to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS movies_genres_idx ON movies USING GIN(genres)

CREATE INDEX IF NOT EXISTS movies_year_idx ON movies USING GIN(year);

CREATE INDEX IF NOT EXISTS movies_runtime_idx ON movies USING GIN(runtime);
