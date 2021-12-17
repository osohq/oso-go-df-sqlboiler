example.db: schema.sql
	rm -f example.db
	sqlite3 example.db < schema.sql

models: example.db
	sqlboiler sqlite3