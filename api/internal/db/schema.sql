CREATE TABLE IF NOT EXISTS leads (
    id INTEGER PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL COLLATE NOCASE UNIQUE,
    finance_amount REAL NOT NULL,
    salary REAL,
    state TEXT NOT NULL,
    vehicle_make TEXT,
    vehicle_model TEXT,
    vehicle_year INTEGER,
    created_at DATETIME NOT NULL
);
