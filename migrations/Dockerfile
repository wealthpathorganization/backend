FROM flyway/flyway:10-alpine

# Copy SQL migrations to Flyway's default location
COPY db/migration/ /flyway/sql/

# Flyway will use environment variables:
# FLYWAY_URL, FLYWAY_USER, FLYWAY_PASSWORD

ENTRYPOINT ["flyway", "migrate"]
