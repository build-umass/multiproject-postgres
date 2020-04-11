# Multitenant Postgres
This application allows 1 postgres database to host multiple different projects. Thus, multiple different projects can use the same Postgres database on AWS/Azure/Google Cloud without interfering/touching/seeing each other's data.

How?

Each Postgres database can have multiple databases inside it (Yes, I know this is weird/confusing). This is why when you connect to Postgres you need to specify a specific database to connect to.

This application, when supplied adminstrative connection credentials for a Postgres instance, *P*, a database name, *D*, and a username, *U*, will create a new database *D* in *P* and a user with username *U* (and a random password composed of adjectives and animal names). It will set up database access privileges, so only *U* and the admin user can connect to *D*. This preserves privacy and prevents one codebase from messing with the data of another codebase.

## Usage
Note: Most likely, you won't run this code yourself because it requires administrative credentials. Rather, you will ask an admin to run the script for you, and give you the output.