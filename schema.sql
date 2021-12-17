CREATE TABLE organizations (
    name VARCHAR PRIMARY KEY NOT NULL
);

CREATE TABLE repositories (
    name VARCHAR PRIMARY KEY NOT NULL,
    org_name VARCHAR NOT NULL
);

INSERT INTO organizations (name) VALUES ("osohq");
INSERT INTO repositories (name, org_name) VALUES ("oso", "osohq");