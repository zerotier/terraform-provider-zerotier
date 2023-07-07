FROM debian:latest

RUN apt-get update -qq && apt-get install iputils-ping netcat-traditional curl gnupg procps -y

RUN curl -sSL https://install.zerotier.com | bash

ENTRYPOINT ["/entrypoint.sh"]
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh
CMD []
