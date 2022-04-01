CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	pin char(4),
	balance int
);

CREATE TABLE IF NOT EXISTS transactions (
	id int,
	amount int,
	user int,

	PRIMARY KEY(id),
	FOREIGN KEY(user) REFERENCES users(id)
);
